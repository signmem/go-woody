package api

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/signmem/go-woody/db"
	"github.com/signmem/go-woody/g"
	"github.com/signmem/go-woody/tools"
)

// handleZone 全量重建 named forward zone 配置。
//
// Purpose:
//
//	Read every domain from pdns.domains (all types), rewrite the named
//	forward zone file (/etc/named/zone.conf by default) from scratch, then
//	hot-reload named (rndc reconfig, restart as fallback) so that the staged
//	hijack zones on named match PDNS exactly.
//
// Usage:
//
//	GET  /api/v2/zoneflush/   dry-run preview, no file or system change
//	POST /api/v2/zoneflush/   backup + rewrite zone file + reload named
//
// Response (JSON):
//
//	{"msg":"success","data":{"dry_run":false,"zone_count":17,
//	 "zone_file":"/etc/named/zone.conf","backup_file":"/etc/named/zone.conf.bak",
//	 "reloaded":true,"domains":["a.example.com",...]}}
//
// Safety notes:
//  1. Refuses to write an empty zone file (would silently drop every hijack
//     zone and expose staging clients to production IPs).
//  2. Backs up the current zone file to <zone_file>.bak before rewriting.
//  3. Rolls back to the backup and reloads again if named rejects the new
//     config, keeping on-disk config consistent with the running one.
//  4. Serialized with appendZoneForwards/removeZoneForwards via zoneMu.

// ZoneFlushResult is the JSON payload returned by /api/v2/zoneflush/.
type ZoneFlushResult struct {
	DryRun     bool     `json:"dry_run"`
	ZoneCount  int      `json:"zone_count"`
	ZoneFile   string   `json:"zone_file"`
	BackupFile string   `json:"backup_file,omitempty"`
	Reloaded   bool     `json:"reloaded"`
	Domains    []string `json:"domains"`
}

// handleZoneFlushPreview GET /api/v2/zoneflush/ 只读预览, 不落盘不 reload
func handleZoneFlushPreview(w http.ResponseWriter, r *http.Request) {
	names, err := loadZoneNames()
	if err != nil {
		g.Logger.Errorf("handleZoneFlushPreview() %v", err)
		RenderFailJson(w, http.StatusInternalServerError, err.Error())
		return
	}

	RenderDataJson(w, ZoneFlushResult{
		DryRun:    true,
		ZoneCount: len(names),
		ZoneFile:  g.Config().ZoneFile,
		Reloaded:  false,
		Domains:   names,
	})
}

// handleZoneFlushPost POST /api/v2/zoneflush/ 执行全量重建
func handleZoneFlushPost(w http.ResponseWriter, r *http.Request) {
	result, err := zoneFlush()
	if err != nil {
		g.Logger.Errorf("handleZoneFlushPost() zoneFlush failed: %v", err)
		RenderFailJson(w, http.StatusInternalServerError, err.Error())
		return
	}

	g.Logger.Infof("handleZoneFlushPost() zone file rebuilt: %d zone(s), file=%s, backup=%s",
		result.ZoneCount, result.ZoneFile, result.BackupFile)
	RenderDataJson(w, result)
}

// zoneFlush 全量重建流程: 读库 -> 备份 -> 原子重写 -> reload, 失败回滚。
// 调用方无需持有 zoneMu, 本函数内部加锁与增量写路径互斥。
func zoneFlush() (*ZoneFlushResult, error) {
	if !g.Config().Named {
		return nil, fmt.Errorf("named integration is disabled (config: named=false)")
	}
	if g.Config().DNS == nil || g.Config().DNS.IP == "" || g.Config().DNS.Port == "" {
		return nil, fmt.Errorf("pdns forwarder address is not configured (config: pdns.ip / pdns.port)")
	}
	if strings.TrimSpace(g.Config().ZoneFile) == "" {
		return nil, fmt.Errorf("zone file path is not configured (config: zone_file)")
	}

	names, err := loadZoneNames()
	if err != nil {
		return nil, err
	}

	// 生产安全护栏: 拒绝把 zone 文件刷成空。
	// 空文件会静默移除全部劫持转发域, 令 staging 客户端直连生产 IP。
	if len(names) == 0 {
		return nil, fmt.Errorf("no domains found in pdns.domains, refuse to write an empty zone file")
	}

	zoneMu.Lock()
	defer zoneMu.Unlock()

	zoneFile := g.Config().ZoneFile

	backupFile, err := backupZoneFile(zoneFile)
	if err != nil {
		return nil, fmt.Errorf("backup zone file failed: %w", err)
	}

	content := renderZoneConf(names)
	if g.Config().Debug {
		g.Logger.Infof("zoneFlush() rendered zone file content (%d bytes):\n%s", len(content), content)
	}

	if err := writeZoneFileAtomic(zoneFile, content); err != nil {
		return nil, fmt.Errorf("write zone file failed: %w", err)
	}

	// reconfig 后 flushname 全部劫持域, 防止旧的生产缓存应答掩盖劫持效果
	if err := reloadNamedAndFlush(names); err != nil {
		// named 拒绝新配置或已宕: 回滚到备份, 保持磁盘配置与运行态一致
		g.Logger.Errorf("zoneFlush() reload named failed: %v, rolling back %s", err, zoneFile)
		if backupFile != "" {
			if rbErr := copyFile(backupFile, zoneFile); rbErr != nil {
				g.Logger.Errorf("zoneFlush() rollback failed: %v", rbErr)
			} else if rbReloadErr := tools.ReloadNamed(); rbReloadErr != nil {
				g.Logger.Errorf("zoneFlush() reload after rollback failed: %v", rbReloadErr)
			}
		}
		return nil, fmt.Errorf("reload named failed (zone file rolled back): %w", err)
	}

	return &ZoneFlushResult{
		DryRun:     false,
		ZoneCount:  len(names),
		ZoneFile:   zoneFile,
		BackupFile: backupFile,
		Reloaded:   true,
		Domains:    names,
	}, nil
}

// loadZoneNames 读取 pdns.domains 中全部域名 (不区分 type) 并规范化:
// 去尾部根点、统一小写、去空白、去重、排序 (重复 zone 声明会导致 named
// 配置检查失败, 必须去重; 排序保证输出确定性, 便于 diff)。
func loadZoneNames() ([]string, error) {
	domains, err := db.GetAllDomains()
	if err != nil {
		return nil, fmt.Errorf("query domains failed: %w", err)
	}

	seen := make(map[string]struct{}, len(domains))
	names := make([]string, 0, len(domains))
	for _, d := range domains {
		if d == nil {
			continue
		}
		name := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(d.Name), "."))
		if name == "" {
			continue
		}
		if _, dup := seen[name]; dup {
			g.Logger.Infof("loadZoneNames() skip duplicate domain %s", name)
			continue
		}
		seen[name] = struct{}{}
		names = append(names, name)
	}
	sort.Strings(names)

	return names, nil
}

// renderZoneConf 生成 zone 文件全文, 头部带生成时间与来源说明。
func renderZoneConf(names []string) string {
	dns := g.Config().DNS

	var buf strings.Builder
	fmt.Fprintf(&buf, "# Generated by go-woody zoneflush at %s\n", time.Now().Format(time.RFC3339))
	fmt.Fprintf(&buf, "# Source: pdns.domains (all types), %d zone(s)\n", len(names))
	buf.WriteString("# DO NOT EDIT MANUALLY: call POST /api/v2/zoneflush/ to rebuild\n")
	for _, name := range names {
		fmt.Fprintf(&buf, "zone \"%s\" IN { type forward; forwarders { %s port %s; }; };\n",
			name, dns.IP, dns.Port)
	}
	return buf.String()
}

// reloadNamedAndFlush 热加载 named 并清理逐域缓存。
// 只以零参数调用 tools.ReloadNamed(), 同时兼容历史非变参签名
// (func ReloadNamed() error) 与新变参签名 (func ReloadNamed(...string) error);
// flushname 由本函数自行执行, 单条失败 (如名字本就不在缓存中) 无害忽略。
func reloadNamedAndFlush(names []string) error {
	if err := tools.ReloadNamed(); err != nil {
		return err
	}

	for _, name := range names {
		if name == "" {
			continue
		}
		_ = exec.Command("rndc", "flushname", name).Run()
	}
	return nil
}

// backupZoneFile 备份当前 zone 文件到 <zone_file>.bak (只保留最近一份)。
// 源文件不存在不算错误 (首次生成场景), 返回空备份路径。
func backupZoneFile(zoneFile string) (string, error) {
	if _, err := os.Stat(zoneFile); err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("stat %s failed: %w", zoneFile, err)
	}

	backup := zoneFile + ".bak"
	if err := copyFile(zoneFile, backup); err != nil {
		return "", err
	}
	return backup, nil
}

// copyFile 以源文件权限复制 src -> dst。
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("read %s failed: %w", src, err)
	}

	info, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("stat %s failed: %w", src, err)
	}

	if err := os.WriteFile(dst, data, info.Mode()); err != nil {
		return fmt.Errorf("write %s failed: %w", dst, err)
	}
	return nil
}

// writeZoneFileAtomic 以"同目录临时文件 + rename(2)"方式原子重写 zone 文件,
// 保留原文件权限并尽力保留属主 (chown best-effort), 与 removeZoneFromFile
// 相同的落盘策略, 避免写一半被 named 读到。
// 调用方必须持有 zoneMu。
func writeZoneFileAtomic(filePath, content string) error {
	var mode os.FileMode = 0644
	uid, gid := -1, -1
	if info, err := os.Stat(filePath); err == nil {
		mode = info.Mode()
		if st, ok := info.Sys().(*syscall.Stat_t); ok {
			uid, gid = int(st.Uid), int(st.Gid)
		}
	}

	tmpFile, err := os.CreateTemp(filepath.Dir(filePath), "."+filepath.Base(filePath)+".*.tmp")
	if err != nil {
		return fmt.Errorf("create temp file failed: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath) // rename 成功后为 no-op

	if _, err := tmpFile.WriteString(content); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("write temp file failed: %w", err)
	}
	if err := tmpFile.Sync(); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("sync temp file failed: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close temp file failed: %w", err)
	}

	if err := os.Chmod(tmpPath, mode); err != nil {
		return fmt.Errorf("chmod temp file failed: %w", err)
	}
	if uid >= 0 {
		if err := os.Chown(tmpPath, uid, gid); err != nil {
			g.Logger.Errorf("writeZoneFileAtomic() chown %s failed (ignored): %v", tmpPath, err)
		}
	}

	if err := os.Rename(tmpPath, filePath); err != nil {
		return fmt.Errorf("rename temp file failed: %w", err)
	}
	return nil
}

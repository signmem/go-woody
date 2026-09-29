package api

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"github.com/signmem/go-woody/g"
	"github.com/signmem/go-woody/tools"
)

// zoneMu 串行化所有 zone 文件写入与 named reload,
// 修复: 原实现并发请求会交错 append zone 文件、交错触发 systemctl restart,
// 批量导入时形成 restart 风暴
var zoneMu sync.Mutex

// ErrZoneNotFound 目标域名不在 zone 文件中 (删除场景可容忍, 只记 warning)
var ErrZoneNotFound = errors.New("zone entry not found in zone file")

// appendZoneForwards 为域名追加 forward zone 条目并热加载 named。
// 调用前必须保证 DB 变更已提交; 并发安全 (内部互斥)。
// reload 使用 rndc reconfig (失败回退 systemctl restart), 不再每次重启 named。
func appendZoneForwards(domainNames []string) error {
	if !g.Config().Named || len(domainNames) == 0 {
		return nil
	}

	zoneMu.Lock()
	defer zoneMu.Unlock()

	zoneFile := g.Config().ZoneFile
	defaultDns := g.Config().DNS

	var buf strings.Builder
	for _, name := range domainNames {
		fmt.Fprintf(&buf, "zone \"%s\" IN { type forward; forwarders { %s port %s; }; };\n",
			name, defaultDns.IP, defaultDns.Port)
	}

	f, err := os.OpenFile(zoneFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open zone file %s failed: %w", zoneFile, err)
	}

	if _, err = f.WriteString(buf.String()); err != nil {
		_ = f.Close()
		return fmt.Errorf("write zone file %s failed: %w", zoneFile, err)
	}

	if err = f.Close(); err != nil {
		return fmt.Errorf("close zone file %s failed: %w", zoneFile, err)
	}

	if err = tools.ReloadNamed(); err != nil {
		return fmt.Errorf("reload named failed: %w", err)
	}

	return nil
}

// removeZoneForwards 删除域名的 forward zone 条目并热加载 named。
// 条目缺失视为可容忍 (记 warning 不报错); 其余错误返回首个失败。
func removeZoneForwards(domainNames []string) error {
	if !g.Config().Named || len(domainNames) == 0 {
		return nil
	}

	zoneMu.Lock()
	defer zoneMu.Unlock()

	zoneFile := g.Config().ZoneFile
	removedAny := false
	var firstErr error

	for _, name := range domainNames {
		err := removeZoneFromFile(zoneFile, name)
		switch {
		case errors.Is(err, ErrZoneNotFound):
			g.Logger.Infof("removeZoneForwards() zone entry for %s not present, skip", name)
		case err != nil:
			g.Logger.Errorf("removeZoneForwards() remove zone entry %s failed: %v", name, err)
			if firstErr == nil {
				firstErr = err
			}
		default:
			removedAny = true
		}
	}

	if firstErr != nil {
		return firstErr
	}

	if removedAny {
		if err := tools.ReloadNamed(); err != nil {
			return fmt.Errorf("reload named failed: %w", err)
		}
	}

	return nil
}

// removeZoneFromFile 以"同目录临时文件 + rename"方式原子重写 zone 文件。
// 修复:
//  1. 临时文件改建在目标文件同目录, 避免 /tmp 跨文件系统 rename(2) EXDEV 失败;
//  2. 保留原文件权限, 并尽力保留属主 (chown best-effort);
//  3. 未命中条目返回 ErrZoneNotFound, 由调用方决定容忍策略。
//
// 调用方必须持有 zoneMu。
func removeZoneFromFile(filePath, domainName string) error {
	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return ErrZoneNotFound
		}
		return fmt.Errorf("open file %s failed: %w", filePath, err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("get file info failed: %w", err)
	}

	tmpFile, err := os.CreateTemp(filepath.Dir(filePath), "."+filepath.Base(filePath)+".*.tmp")
	if err != nil {
		return fmt.Errorf("create temp file failed: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath) // rename 成功后为 no-op

	target := fmt.Sprintf(`zone "%s"`, domainName)
	scanner := bufio.NewScanner(file)
	writer := bufio.NewWriter(tmpFile)

	removed := false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, target) {
			removed = true
			continue
		}
		if _, err := writer.WriteString(line + "\n"); err != nil {
			_ = tmpFile.Close()
			return fmt.Errorf("write to temp file failed: %w", err)
		}
	}

	if err := scanner.Err(); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("scan file failed: %w", err)
	}

	if err := writer.Flush(); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("flush writer failed: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close temp file failed: %w", err)
	}

	if !removed {
		return ErrZoneNotFound
	}

	if err := os.Chmod(tmpPath, fileInfo.Mode()); err != nil {
		return fmt.Errorf("set temp file permission failed: %w", err)
	}

	if st, ok := fileInfo.Sys().(*syscall.Stat_t); ok {
		if err := os.Chown(tmpPath, int(st.Uid), int(st.Gid)); err != nil {
			g.Logger.Errorf("removeZoneFromFile() chown %s failed (ignored): %v", tmpPath, err)
		}
	}

	if err := os.Rename(tmpPath, filePath); err != nil {
		return fmt.Errorf("replace original file failed: %w", err)
	}

	return nil
}

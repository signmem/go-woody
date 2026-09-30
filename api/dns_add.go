package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/signmem/go-woody/db"
	"github.com/signmem/go-woody/g"
)

// 说明: 为彻底消除"先查重后插入"的并发窗口 (TOCTOU), 建议在 MySQL 增加唯一索引:
//
//	ALTER TABLE records ADD UNIQUE KEY uk_records_name_type_content (name, type, content);
//
// 代码已兼容处理 1062 duplicate-key 错误, 加上索引后即为并发安全。

func dnsAdd(r *http.Request) (htmlMsg ReturnMsg, err error) {

	// 只处理 dns 增加功能

	if r.ContentLength == 0 {
		msg := fmt.Errorf("dnsAdd() Error: body is blank")
		g.Logger.Error(msg)
		htmlMsg.Msg = "Post data not valid, body is blank!"
		return htmlMsg, msg
	}

	if !isContentTypeJson(r) {
		msg := fmt.Errorf("dnsAdd() Error: body not json format")
		g.Logger.Error(msg)
		htmlMsg.Msg = "dnsAdd() Post data not valid, body not json format!"
		return htmlMsg, msg
	}

	defer func() {
		_ = r.Body.Close()
	}()

	body, err := io.ReadAll(r.Body)

	if err != nil {
		msg := fmt.Errorf("dnsAdd() Error: body read error")
		g.Logger.Error(msg)
		htmlMsg.Msg = "Post data not valid, body read error!"
		return htmlMsg, msg
	}

	var hostDict HostCreate

	err = json.Unmarshal(body, &hostDict)

	if err != nil {
		msg := fmt.Errorf("dnsAdd() Error: body json unmarshal error")
		g.Logger.Error(msg)
		htmlMsg.Msg = "Post data not valid, body json unmarshal format error!"
		return htmlMsg, msg
	}

	TrimAllStrings(&hostDict)

	if len(hostDict.Hosts) == 0 {
		msg := fmt.Errorf("dnsAdd() Error: HostCreate empty")
		g.Logger.Error(msg)
		htmlMsg.Msg = "Post data not valid, HostCreate empty!"
		return htmlMsg, msg
	}

	successAdd := 0
	falseAdd := 0

	// 修复: zone 条目统一收集, 全部 DB 提交后一次性加锁追加 + rndc reconfig,
	// 不再边加边写文件、每次请求 systemctl restart
	addedHosts := make([]string, 0, len(hostDict.Hosts))

	if g.Config().Debug {
		g.Logger.Debugf("[v1-add] dnsAdd() add %s", hostDict.String())
	}

	for _, host := range hostDict.Hosts {

		hostName := strings.TrimSpace(host.Hostname)
		ipaddr := strings.TrimSpace(host.IP)

		// 基础空值校验
		if hostName == "" || ipaddr == "" {
			g.Logger.Errorf("dnsAdd() Error: hostname or ip is empty")
			falseAdd += 1
			continue
		}

		if !isIPv4(ipaddr) {
			g.Logger.Errorf("dnsAdd() Error: %s not valid ipaddress", ipaddr)
			falseAdd += 1
			continue
		}

		if !db.IsValidHostname(hostName) {
			g.Logger.Errorf("dnsAdd() Error: %s not valid hostname", hostName)
			falseAdd += 1
			continue
		}

		dnsRecords, err := db.GetRecordsByHostName(hostName)

		if err != nil {
			g.Logger.Errorf("dnsAdd() get host %s records query err: %v", hostName, err)
			falseAdd += 1
			continue
		}

		ipExists := false
		for _, dnsRecord := range dnsRecords {
			if dnsRecord.Content == ipaddr && dnsRecord.Name == hostName {
				ipExists = true
				break
			}
		}

		if ipExists {
			g.Logger.Errorf("[v1-add] dnsAdd() Error: %s records exists", hostName)
			falseAdd += 1
			continue
		}

		if err := addSingleHost(host); err != nil {
			if errors.Is(err, errRecordExists) || isMySQLDuplicate(err) {
				g.Logger.Errorf("[v1-add] dnsAdd() Error: %s records exists (concurrent insert)", hostName)
			} else {
				g.Logger.Errorf("[v1-add] dnsAdd() add host %s error: %s", hostName, err)
			}
			falseAdd += 1
			continue
		}

		successAdd += 1
		addedHosts = append(addedHosts, hostName)

		if g.Config().Debug {
			g.Logger.Debugf("[v1-add] dnsAdd() Debug: add hostname %v", hostName)
		}
	}

	var addStatus DnsAddStatus
	addStatus.Success = successAdd
	addStatus.Failure = falseAdd

	htmlMsg.Msg = addStatus.String()

	// DB 已提交, 同步 named zone (内部互斥 + rndc reconfig)。
	// 修复: 失败时明确返回"部分成功"信息 (已入库数量 + zone 同步失败原因),
	// 便于运维补偿, 不再丢失已提交的成功计数
	if zoneErr := appendZoneForwards(addedHosts); zoneErr != nil {
		g.Logger.Errorf("dnsAdd() sync named zone failed: %v", zoneErr)
		htmlMsg.Msg = fmt.Sprintf("%s (WARNING: %d hosts saved to DB, "+
			"but named zone sync failed: %v)", addStatus.String(), successAdd, zoneErr)
		return htmlMsg, zoneErr
	}

	return htmlMsg, nil
}

func addSingleHost(host HostParams) (err error) {

	hostName := strings.TrimSpace(host.Hostname)
	ipaddr := strings.TrimSpace(host.IP)

	if db.DB == nil {
		g.Logger.Error("Database connection is nil - check if initDB() was called")
		return fmt.Errorf("database connection is not initialized")
	}

	tx, err := db.DB.Begin()
	if err != nil {
		return fmt.Errorf("addSingleHost() Error: failed to begin transaction: %w", err)
	}

	rollbackNeeded := true
	defer func() {
		if rollbackNeeded {
			// 只有失败才回滚
			if rollbackErr := tx.Rollback(); rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
				g.Logger.Errorf("addSingleHost() rollback error: %v", rollbackErr)
			}
		}
	}()

	// 只对 pdns.domains 表添加域名信息
	domainID, err := dnsDomainAdd(tx, hostName)

	if err != nil {
		return fmt.Errorf("addSingleHost() domain %s add error: %w", hostName, err)
	}

	if _, err = dnsHostAdd(tx, domainID, hostName, ipaddr); err != nil {
		if isMySQLDuplicate(err) {
			return fmt.Errorf("addSingleHost() host %s ip %s: %w", hostName, ipaddr, errRecordExists)
		}
		return fmt.Errorf("addSingleHost() dns %s add error: %w", hostName, err)
	}

	if err = db.UpdateSOA(tx, hostName); err != nil {
		return fmt.Errorf("addSingleHost() dns %s update SOA error: %w", hostName, err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("addSingleHost() db commit error: %w", err)
	}
	rollbackNeeded = false

	return nil
}

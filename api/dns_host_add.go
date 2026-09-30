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

func hostAdd(r *http.Request) (htmlMsg ReturnMsg, err error) {

	// 只处理 host 增加功能

	if !isContentTypeJson(r) {
		msg := fmt.Errorf("hostAdd() Error: body not json format")
		g.Logger.Error(msg)
		htmlMsg.Msg = "hostAdd() Post data not valid, body not json format!"
		return htmlMsg, msg
	}

	defer func() {
		_ = r.Body.Close()
	}()

	body, err := io.ReadAll(r.Body)

	if err != nil {
		msg := fmt.Errorf("hostAdd() Error: body read error")
		g.Logger.Error(msg)
		htmlMsg.Msg = "hostAdd()  Post data not valid, body read error!"
		return htmlMsg, msg
	}

	var hostDict HostCreate

	err = json.Unmarshal(body, &hostDict)

	if err != nil {
		msg := fmt.Errorf("hostAdd() Error: body json unmarshal error")
		g.Logger.Error(msg)
		htmlMsg.Msg = "hostAdd() Post data not valid, body json unmarshal format error!"
		return htmlMsg, msg
	}

	TrimAllStrings(&hostDict)

	if len(hostDict.Hosts) == 0 {
		msg := fmt.Errorf("hostAdd() Error: HostCreate empty")
		g.Logger.Error(msg)
		htmlMsg.Msg = "hostAdd() Post data not valid, HostCreate empty!"
		return htmlMsg, msg
	}

	successAdd := 0
	falseAdd := 0

	if g.Config().Debug {
		g.Logger.Debugf("hostAdd() add %s", hostDict.String())
	}

	for _, host := range hostDict.Hosts {

		hostName := strings.TrimSpace(host.Hostname)
		ipaddr := strings.TrimSpace(host.IP)

		// 基础空值校验
		if hostName == "" || ipaddr == "" {
			g.Logger.Errorf("hostAdd() Error: hostname or ip is empty")
			falseAdd += 1
			continue
		}

		if !isIPv4(ipaddr) {
			g.Logger.Errorf("hostAdd() Error: %s not valid ipaddress", ipaddr)
			falseAdd += 1
			continue
		}

		if !db.IsValidHostname(hostName) {
			g.Logger.Errorf("hostAdd() Error: %s not valid hostname", hostName)
			falseAdd += 1
			continue
		}

		dnsRecords, err := db.GetRecordsByHostName(hostName)

		if err != nil {
			// 修复: 原实现查询报错时静默 falseAdd, 不留任何日志
			g.Logger.Errorf("hostAdd() get host %s records query err: %v", hostName, err)
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
			g.Logger.Errorf("hostAdd() Error: %s records exists", hostName)
			falseAdd += 1
			continue
		}

		if err = addDomainHost(host); err != nil {
			if errors.Is(err, errRecordExists) || isMySQLDuplicate(err) {
				g.Logger.Errorf("hostAdd() Error: %s records exists (concurrent insert)", hostName)
			} else {
				g.Logger.Error(err)
			}
			falseAdd += 1
			continue
		}

		successAdd += 1
	}

	var addStatus DnsAddStatus
	addStatus.Success = successAdd
	addStatus.Failure = falseAdd

	htmlMsg.Msg = addStatus.String()

	return htmlMsg, nil
}

// GetParentDomain 返回主机名的父域名 (去掉第一段 label)。
// 两段及以下视为域名本身。
func GetParentDomain(host string) string {
	host = strings.TrimSpace(host)
	if host == "" {
		return ""
	}
	parts := strings.Split(host, ".")
	if len(parts) <= 2 {
		return host
	}
	parentParts := parts[1:]
	return strings.Join(parentParts, ".")
}

func addDomainHost(host HostParams) (err error) {

	hostName := strings.TrimSpace(host.Hostname)
	ipaddr := strings.TrimSpace(host.IP)

	g.Logger.Debugf("[v2-host-add] add host:%s  ip:%s", hostName, ipaddr)

	if db.DB == nil {
		g.Logger.Error("Database connection is nil - check if initDB() was called")
		return fmt.Errorf("database connection is not initialized")
	}

	tx, err := db.DB.Begin()
	if err != nil {
		g.Logger.Errorf("addDomainHost() Error: failed to begin transaction %s", err)
		return fmt.Errorf("addDomainHost() begin tx failed: %w", err)
	}

	rollbackNeeded := true

	defer func() {
		if rollbackNeeded {
			// 只有失败才回滚
			if rollbackErr := tx.Rollback(); rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
				g.Logger.Errorf("addDomainHost() rollback error: %v", rollbackErr)
			}
		}
	}()

	domainName := GetParentDomain(hostName)

	domainInfo, err := db.GetDomainsByName(domainName)

	if err != nil {
		return fmt.Errorf("addDomainHost() lookup parent domain %s failed: %w", domainName, err)
	}

	if domainInfo == nil || domainInfo.ID < 1 {
		return fmt.Errorf("addDomainHost() %s parent domain %s not found", hostName, domainName)
	}

	domainID := domainInfo.ID

	if _, err = dnsHostAdd(tx, domainID, hostName, ipaddr); err != nil {
		if isMySQLDuplicate(err) {
			return fmt.Errorf("addDomainHost() host %s ip %s: %w", hostName, ipaddr, errRecordExists)
		}
		return fmt.Errorf("addDomainHost() dns %s add error: %w", hostName, err)
	}

	if err = db.UpdateSOA(tx, domainName); err != nil {
		return fmt.Errorf("addDomainHost() dns %s update SOA error: %w", domainName, err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("addDomainHost() db commit error: %w", err)
	}
	rollbackNeeded = false

	g.Logger.Debugf("[v2-host-add] add host:%s  ip:%s success", hostName, ipaddr)

	return nil
}

package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/signmem/go-woody/db"
	"github.com/signmem/go-woody/g"
	"io"
	"mime"
	"net/http"
	"strings"
)

func hostAdd(r *http.Request) (htmlMsg ReturnMsg, err error) {

	// 只处理 dns 增加功能

	headerContentType := r.Header.Get("Content-Type")
	mediaType, _, err := mime.ParseMediaType(headerContentType)
	if err != nil || mediaType != "application/json" {
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

	if len(hostDict.Hosts)  == 0 {
		msg := fmt.Errorf("hostAdd() Error: HostCreate empty")
		g.Logger.Error(msg)
		htmlMsg.Msg = "hostAdd() Post data not valid, HostCreate empty!"
		return htmlMsg, msg
	}

	successAdd := 0
	falseAdd := 0

	if g.Config().Debug == true {
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

		if isIPv4(ipaddr) == false {
			g.Logger.Errorf("hostAdd() Error: %s not valid ipaddress", ipaddr)
			falseAdd += 1
			continue
		}

		if db.IsValidHostname(hostName) == false {
			g.Logger.Errorf("hostAdd() Error: %s not valid hostname",  hostName)
			falseAdd += 1
			continue
		}

		dnsRecords, err :=  db.GetRecordsByHostName(hostName)

		ipExists := false

		if err == nil {

			for _, dnsRecord := range dnsRecords  {

				if dnsRecord.Content == ipaddr && dnsRecord.Name == hostName {
					ipExists = true
					msg := fmt.Sprintf("hostAdd() Error: %s records exists", hostName)
					g.Logger.Error( msg )
					break
				}
			}

			if ipExists == true {
				falseAdd += 1
				continue
			}
		} else {
			falseAdd += 1
			continue
		}

		err = addDomainHost(host)
		if err != nil {
			g.Logger.Error(err)
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
	ipaddr   := strings.TrimSpace(host.IP)

	if db.DB == nil {
		g.Logger.Error("Database connection is nil - check if initDB() was called")
		return fmt.Errorf("database connection is not initialized")
	}

	tx, err := db.DB.Begin()
	if err != nil {
		g.Logger.Errorf("addDomainHost() Error: failed to begin transaction %s ", err)
		return fmt.Errorf("addDomainHost() begin tx failed: %w", err)
	}

	rollbackNeeded := true

	defer func() {
		if rollbackNeeded {
			// 只有失败才回滚
			if rollbackErr := tx.Rollback(); rollbackErr != nil && rollbackErr != sql.ErrTxDone {
				g.Logger.Errorf("addDomainHost() rollback error: %v", rollbackErr)
			}
		}
	}()


	domainName := GetParentDomain(hostName)

	domainInfo, err := db.GetDomainsByName(domainName)

	if err != nil  {
		return fmt.Errorf("lookup parent domain %s failed: %w", domainName, err)
	}

	if  domainInfo == nil || domainInfo.ID < 1 {
		return fmt.Errorf("addDomainHost() %s domain not found id <1", hostName)
	}

	// 只对 pdns.domains 表添加域名信息
	domain_id := domainInfo.ID

	_, err = dnsHostAdd(tx, domain_id, hostName, ipaddr)

	if err != nil {
		return fmt.Errorf("dns %s add error: %w", hostName, err)
	}

	err = db.UpdateSOA(tx, domainName)

	if err != nil {
		return fmt.Errorf("dns %s update SOA error: %w", domainName, err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("db commit error: %w", err)
	}
	rollbackNeeded = false

	return nil
}

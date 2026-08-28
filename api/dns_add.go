package api

import (
	"database/sql"
	"fmt"
	"github.com/signmem/go-woody/db"
	"github.com/signmem/go-woody/g"
	"github.com/signmem/go-woody/tools"
	"io"
	"mime"
	"net/http"
	"encoding/json"
	"os"
	"strings"
)

func dnsAdd(r *http.Request) (htmlMsg ReturnMsg, err error) {

	// 只处理 dns 增加功能

	if r.ContentLength == 0 {
		msg := fmt.Errorf("dnsAdd() Error: body is blank")
		g.Logger.Error(msg)
		htmlMsg.Msg = "Post data not valid, body is blank!"
		return htmlMsg, msg
	}

	headerContentType := r.Header.Get("Content-Type")
	mediaType, _, err := mime.ParseMediaType(headerContentType)
	if err != nil || mediaType != "application/json" {
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

	if len(hostDict.Hosts)  == 0 {
		msg := fmt.Errorf("dnsAdd() Error: HostCreate empty")
		g.Logger.Error(msg)
		htmlMsg.Msg = "Post data not valid, HostCreate empty!"
		return htmlMsg, msg
	}

	successAdd := 0
	falseAdd := 0

	var zoneBuf strings.Builder

	if g.Config().Debug == true {
		g.Logger.Debugf("dnsAdd() add %s", hostDict.String())
	}

	defaultDns := g.Config().DNS

	for _, host := range hostDict.Hosts {

		hostName := strings.TrimSpace(host.Hostname)
		ipaddr   := strings.TrimSpace(host.IP)

		// 基础空值校验
		if hostName == "" || ipaddr == "" {
			g.Logger.Errorf("dnsAdd() Error: hostname or ip is empty")
			falseAdd += 1
			continue
		}

		if isIPv4(ipaddr) == false {
			g.Logger.Errorf("dnsAdd() Error: %s not valid ipaddress", ipaddr)
			falseAdd += 1
			continue
		}

		if db.IsValidHostname(hostName) == false {
			g.Logger.Errorf("dnsAdd() Error: %s not valid hostname",  hostName)
			falseAdd += 1
			continue
		}

		dnsRecords, err :=  db.GetRecordsByHostName(hostName)

		ipExists := false

		if err == nil {

			for _, dnsRecord := range dnsRecords  {

				if dnsRecord.Content == ipaddr && dnsRecord.Name == hostName {
					ipExists = true
					g.Logger.Errorf("dnsAdd() Error: %s records exists", hostName)
					break
				}
			}

			if ipExists == true {
				falseAdd += 1
				continue
			}
		} else {
			falseAdd += 1
			g.Logger.Errorf("dnsAdd() get host %s records query err: %v", hostName, err)
			continue
		}

		if err := addSingleHost(host) ; err != nil {
			g.Logger.Errorf("dnsAdd() add host %s error: %s", host.Hostname, err)
			falseAdd += 1
			continue
		} else {
			successAdd += 1
			if g.Config().Debug == true {
				g.Logger.Debugf("dnsAdd() Debug: add hostname %v", hostName)
			}

			if g.Config().Named == true {
				forwaroders := fmt.Sprintf("zone \"%s\" IN { type forward; forwarders " +
					"{ %s port %s; }; };\n", hostName, defaultDns.IP, defaultDns.Port)
				zoneBuf.WriteString(forwaroders)
			}
		}
	}

	zoneFile := g.Config().ZoneFile

	if zoneBuf.Len() > 0 && g.Config().Named == true {
		f, err := os.OpenFile(zoneFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			msg := fmt.Sprintf("dnsAdd() zone file %s open fail", zoneFile)
			g.Logger.Error(msg)
			htmlMsg.Msg = msg
			return htmlMsg, err
		}

		_, err = f.WriteString(zoneBuf.String())
		_ = f.Close()

		if err != nil {
			msg := fmt.Sprintf("dnsAdd() zone file %s write fail", zoneFile)
			g.Logger.Error(msg)
			htmlMsg.Msg = msg
			return htmlMsg, err
		}

		err = tools.RestartNamed()
		if err != nil {
			msg := fmt.Sprintf("dnsAdd() Error: restart named %s", err)
			g.Logger.Error(msg)
			htmlMsg.Msg = msg
			return htmlMsg, err
		}

	}

	var addStatus DnsAddStatus
	addStatus.Success = successAdd
	addStatus.Failure = falseAdd

	htmlMsg.Msg = addStatus.String()

	return htmlMsg, nil
}


func addSingleHost(host HostParams) (err error) {

	hostName := host.Hostname
	ipaddr   := host.IP

	if db.DB == nil {
		g.Logger.Error("Database connection is nil - check if initDB() was called")
		return fmt.Errorf("database connection is not initialized")
	}

	tx, err := db.DB.Begin()
	if err != nil {
		return fmt.Errorf("addSingleHost() Error: failed to begin transaction")
	}

	rollbackNeeded := true
	defer func() {
		if rollbackNeeded {
			// 只有失败才回滚
			if rollbackErr := tx.Rollback(); rollbackErr != nil && rollbackErr != sql.ErrTxDone {
				g.Logger.Errorf("addSingleHost() rollback error: %v", rollbackErr)
			}
		}
	}()

	// 只对 pdns.domains 表添加域名信息
	domain_id, err := dnsDomainAdd(tx, hostName)

	if err != nil {
		return fmt.Errorf("dnsAdd() Error: domain %s add error: %s", hostName, err)
	}

	_, err = dnsHostAdd(tx, domain_id, hostName, ipaddr)

	if err != nil {
		return fmt.Errorf("dns %s add error: %s", hostName, err)
	}

	err = db.UpdateSOA(tx, hostName)

	if err != nil {
		return fmt.Errorf("dns %s update SOA error: %s", hostName, err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("db commit error: %s", err)
	}
	rollbackNeeded = false

	return nil
}

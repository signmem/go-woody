package api

import (
	"database/sql"
	"fmt"
	"github.com/pkg/errors"
	"github.com/signmem/go-woody/db"
	"github.com/signmem/go-woody/g"
	"io"
	"net/http"
	"encoding/json"
)

func dnsAdd(r *http.Request) (htmlMsg ReturnMsg, err error) {

	// 只处理 dns 增加功能

	if r.ContentLength == 0 {
		msg := fmt.Errorf("dnsAdd() Error: body is blank")
		g.Logger.Error(msg)
		htmlMsg.Msg = "Post data not valid, body is blank!"
		return htmlMsg, msg
	}

	headerContentTtype := r.Header.Get("Content-Type")
	if headerContentTtype != "application/json" {
		msg := fmt.Errorf("dnsAdd() Error: body not json format")
		g.Logger.Error(msg)
		htmlMsg.Msg = "Post data not valid, body not json format!"
		return htmlMsg, msg
	}

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
		msg := fmt.Errorf("dnsAdd() Error: body json unmarshar error")
		g.Logger.Error(msg)
		htmlMsg.Msg = "Post data not valid, body json unmarshar format error!"
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

	if g.Config().Debug == true {
		g.Logger.Debugf("dnsAdd() add %s", hostDict.String())
	}

	for _, host := range hostDict.Hosts {

		hostName := host.Hostname
		ipaddr := host.IP

		if isIPv4(ipaddr) == false {
			g.Logger.Errorf("dnsAdd() Error: %s not valid ipaddress", ipaddr)
			falseAdd += 1
			continue
		}

		// 验证主机名是否合法
		// _, domainDict, fullDomain,  err := db.HostCheck(hostName)

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
					msg := fmt.Sprintf("dnsAdd() Error: %s records exists", hostName)
					g.Logger.Error( msg )
					break
				}

			}

			if ipExists == true {
				falseAdd += 1
				continue
			}
		}

		if err := addSingleHost(host) ; err != nil {
			g.Logger.Errorf("dnsAdd() add host %s error: %s", host.Hostname, err)
			falseAdd += 1
		} else {
			successAdd += 1
			if g.Config().Debug == true {
				g.Logger.Debugf("dnsAdd() Debug: add hostname %v", hostName)
			}
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
		return errors.New("addSingleHost() Error: failed to begin transaction")
	}

	defer func() {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			msg := fmt.Sprintf("addSingleHost() Error: transaction rollback error")
			g.Logger.Errorf(msg)
		}
	}()

	// 只对 pdns.domains 表添加域名信息
	domain_id, err := dnsDomainAdd(tx, hostName)

	if err != nil {
		tx.Rollback()
		msg := fmt.Sprintf("dnsAdd() Error: domain %s add error: %s", hostName, err)
		return errors.New(msg)

	}

	_, err = dnsHostAdd(tx, domain_id, hostName, ipaddr)

	if err != nil {
		tx.Rollback()
		msg := fmt.Sprintf("dns %s add error: %s", hostName, err)
		return errors.New(msg)
	}

	err = db.UpdateSOA(tx, hostName)

	if err != nil {
		tx.Rollback()
		msg := fmt.Sprintf("dns %s update SOA error: %s", hostName, err)
		return errors.New(msg)
	}

	if err = tx.Commit(); err != nil {
		msg := fmt.Sprintf("db commit error: %s", err)
		return errors.New(msg)
	}

	return nil
}
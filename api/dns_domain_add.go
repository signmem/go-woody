package api

import (
	"database/sql"
	"fmt"
	"github.com/signmem/go-woody/tools"
	"io"
	"mime"
	"net/http"
	"github.com/signmem/go-woody/g"
	"github.com/signmem/go-woody/db"
	"encoding/json"
	"errors"
	"os"
	"strings"
)

func domainAdd(r *http.Request) (htmlMsg ReturnMsg, err error) {

	// 只处理 dns 增加功能

	//	if r.ContentLength == 0 {
	//		msg := fmt.Errorf("domainAdd() Error: body is blank")
	//		g.Logger.Error(msg)
	//		htmlMsg.Msg = "domainAdd() Post data not valid, body is blank!"
	//		return htmlMsg, msg
	//	}


	headerContentTtype := r.Header.Get("Content-Type")
	mediaType, _, err := mime.ParseMediaType(headerContentTtype)
	if err != nil || mediaType != "application/json" {
		msg := fmt.Errorf("domainAdd() Error: body not json format")
		g.Logger.Error(msg)
		htmlMsg.Msg = "domainAdd() Post data not valid, body not json format!"
		return htmlMsg, msg
	}

	defer func() {
		_ = r.Body.Close()
	}()

	body, err := io.ReadAll(r.Body)

	if err != nil {
		msg := fmt.Errorf("domainAdd() Error: body read error")
		g.Logger.Error(msg)
		htmlMsg.Msg = "domainAdd() Post data not valid, body read error!"
		return htmlMsg, msg
	}

	var DomainList DomainCreate

	err = json.Unmarshal(body, &DomainList)

	if err != nil {
		msg := fmt.Errorf("domainAdd() Error: body json unmarsharl error")
		g.Logger.Error(msg)
		htmlMsg.Msg = "domainAdd() Post data not valid, body json unmarsharl format error!"
		return htmlMsg, msg
	}

	if len(DomainList.Domains)  == 0 {
		msg := fmt.Errorf("domainAdd() Error: HostCreate empty")
		g.Logger.Error(msg)
		htmlMsg.Msg = "domainAdd() Post data not valid, HostCreate empty!"
		return htmlMsg, msg
	}

	successAdd := 0
	falseAdd := 0

	if g.Config().Debug == true {
		g.Logger.Debugf("domainAdd() add %s", DomainList.String())
	}


	tx, err := db.DB.Begin()
	if err != nil {
		msg := "domainAdd() Error: DB error"
		g.Logger.Error(msg)
		htmlMsg.Msg = msg
		return htmlMsg, errors.New("addSingleHost() Error: failed to begin transaction")
	}

	defer func() {
		if err != nil {
			// 只有失败才回滚
			if rollbackErr := tx.Rollback(); rollbackErr != nil && rollbackErr != sql.ErrTxDone {
				g.Logger.Errorf("addSingleHost() rollback error: %v", rollbackErr)
			}
		}
	}()

	zoneFile := g.Config().ZoneFile
	defaultDns := g.Config().DNS

	var zoneBuf strings.Builder

	for _, domain := range DomainList.Domains {

		// 基础空值校验
		if domain == "" {
			g.Logger.Errorf("domainAdd() Error: domain empty")
			falseAdd += 1
			continue
		}

		if db.IsValidDomain(domain) == false {
			g.Logger.Errorf("domainAdd() Error: %s not valid domain", domain)
			falseAdd += 1
			continue
		}

		subDomainLevel := db.GetDomainReverseLevels(domain)

		for _, subDomain := range subDomainLevel {

			if subDomain == "" {
				continue
			}

			domainDBInfo, err := db.GetDomainsByName(subDomain)
			if domainDBInfo == nil  && err == nil {

				var domainDB db.Domain
				if DomainList.Master == "" {

					domainDB.Type = "MASTER"
					domainDB.Name = subDomain
					domainDB.Master = nil

				} else {

					domainDB.Type = "SLAVE"
					domainDB.Name = subDomain
					domainDB.Master = &DomainList.Master

				}

				domain_id, err := db.InsertDomain(tx, domainDB)

				if err != nil || domain_id == 0 {
					msg := fmt.Sprintf("domainAdd() Error: InsertDomain error %s", err)
					g.Logger.Error(msg)
					falseAdd += 1
					continue
				}

				if domainDB.Type == "MASTER" {
					err = domainMetaDataAdd(tx, domain_id, subDomain)

					if err != nil {
						msg := fmt.Sprintf("metadataba fail with domain %s", subDomain)
						g.Logger.Error(msg)
						falseAdd += 1
						continue
					}
				}

				if g.Config().Named == true {

					forwaroders := fmt.Sprintf("zone \"%s\" IN { type forward; forwarders " +
						"{ %s port %s; }; };\n", subDomain, defaultDns.IP, defaultDns.Port)

					zoneBuf.WriteString(forwaroders)

				}

				successAdd += 1

			}
		}
	}

	if err = tx.Commit(); err != nil {
		msg := fmt.Sprintf("domainAdd() Error: commit failed: %v", err)
		g.Logger.Error(msg)
		htmlMsg.Msg = msg
		return htmlMsg, err
	}


	if zoneBuf.Len() > 0 && g.Config().Named == true {
		f, err := os.OpenFile(zoneFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			msg := fmt.Sprintf("zone file %s open fail", zoneFile)
			g.Logger.Error(msg)
			htmlMsg.Msg = msg
		}

		_, err = f.WriteString(zoneBuf.String())
		_ = f.Close()

		if err != nil {
			msg := fmt.Sprintf("zone file %s write fail", zoneFile)
			g.Logger.Error(msg)
			htmlMsg.Msg = msg
		}


		err = tools.RestartNamed()
		if err != nil {
			msg := fmt.Sprintf("domainAdd() Error: restart named %s", err)
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


func domainMetaDataAdd(tx *sql.Tx, domain_id int64, domainName string) (err error) {

	var domainDBMeta db.DomainMeta
	domainDBMeta.DomainID  = domain_id

	dns_server := g.Config().DnsServer

	for _, server := range dns_server {
		domainDBMeta.Content = server
		domainDBMeta.Kind = "ALLOW-AXFR-IPS"

		_, err := db.InsertDomainMetaData(tx, domainDBMeta)
		if err != nil {
			return err
		}

		remoteContent := server + ":" + g.Config().DNS.Port
		domainDBMeta.Content = remoteContent
		domainDBMeta.Kind    = "ALLOW-AXFR-FROM"

		_, err = db.InsertDomainMetaData(tx, domainDBMeta)
		if err != nil {
			return err
		}

		domainDBMeta.Kind    = "ALSO-NOTIFY"
		_, err = db.InsertDomainMetaData(tx, domainDBMeta)
		if err != nil {
			return err
		}
	}

	return nil
}
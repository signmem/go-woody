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
	_ "errors"
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


	headerContentType := r.Header.Get("Content-Type")
	mediaType, _, err := mime.ParseMediaType(headerContentType)
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
		msg := fmt.Errorf("domainAdd() Error: body json unmarshal error")
		g.Logger.Error(msg)
		htmlMsg.Msg = "domainAdd() Post data not valid, body json unmarshal format error!"
		return htmlMsg, msg
	}

	TrimAllStrings(&DomainList)

	DomainList.Master = strings.TrimSpace(DomainList.Master)

	if len(DomainList.Domains)  == 0 {
		msg := fmt.Errorf("domainAdd() Error: DomainList empty")
		g.Logger.Error(msg)
		htmlMsg.Msg = "domainAdd() Post data not valid, DomainList empty!"
		return htmlMsg, msg
	}

	successAdd := 0
	falseAdd := 0

	if g.Config().Debug == true {
		g.Logger.Debugf("domainAdd() add %s  ", DomainList.String())
	}

	tx, err := db.DB.Begin()
	if err != nil {
		msg := fmt.Sprintf("domainAdd() Error: failed to begin transaction: %w", err)
		g.Logger.Error(msg)
		htmlMsg.Msg = msg
		return htmlMsg, err
	}

	rollbackNeeded := true
	defer func() {
		if rollbackNeeded {
			if rollbackErr := tx.Rollback(); rollbackErr != nil && rollbackErr != sql.ErrTxDone {
				g.Logger.Errorf("addSingleHost() rollback error: %v", rollbackErr)
			}
		}
	}()

	zoneFile := g.Config().ZoneFile
	defaultDns := g.Config().DNS

	var zoneBuf strings.Builder

	for _, domain := range DomainList.Domains {

		domain := strings.TrimSpace(domain)

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

			if err != nil &&  err != sql.ErrNoRows{
				g.Logger.Errorf("domainAdd() %s db query err: %s", subDomain, err)
				falseAdd += 1
				continue
			}

			if domainDBInfo != nil {
				g.Logger.Errorf("domainAdd() %s in db ready", subDomain)
				falseAdd += 1
				continue
			}

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
				_, _, err = DomainMetaDataAdd(tx, domain_id)

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

	if err = tx.Commit(); err != nil {
		msg := fmt.Sprintf("domainAdd() Error: commit failed: %v", err)
		g.Logger.Error(msg)
		htmlMsg.Msg = msg
		return htmlMsg, err
	}

        rollbackNeeded = false

	if zoneBuf.Len() > 0 && g.Config().Named == true {
		f, err := os.OpenFile(zoneFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			msg := fmt.Sprintf("zone file %s open fail", zoneFile)
			g.Logger.Error(msg)
			htmlMsg.Msg = msg
			return htmlMsg, err
		}

		_, err = f.WriteString(zoneBuf.String())
		_ = f.Close()

		if err != nil {
			msg := fmt.Sprintf("zone file %s write fail", zoneFile)
			g.Logger.Error(msg)
			htmlMsg.Msg = msg
			return htmlMsg, err
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


func DomainMetaDataAdd(tx *sql.Tx, domain_id int64) (success int, false int, err error) {

	var domainDBMeta db.DomainMeta
	domainDBMeta.DomainID  = domain_id

	dns_server := g.Config().DnsServer

	for _, server := range dns_server {
		domainDBMeta.Content = server
		domainDBMeta.Kind = "ALLOW-AXFR-IPS"

		idb, err := db.InsertDomainMetaData(tx, domainDBMeta)
		if err != nil {
			return success, false, err
		}

		if idb == 1 {
			success +=1
		} else {
			false += 1
		}

		remoteContent := server + ":" + g.Config().DNS.Port
		domainDBMeta.Content = remoteContent
		domainDBMeta.Kind    = "ALLOW-AXFR-FROM"

		idc, err := db.InsertDomainMetaData(tx, domainDBMeta)
		if err != nil {
			return success, false, err
		}

		if idc == 1 {
			success +=1
		} else {
			false += 1
		}

		domainDBMeta.Kind    = "ALSO-NOTIFY"
		idd, err := db.InsertDomainMetaData(tx, domainDBMeta)
		if err != nil {
			return success, false, err
		}

		if idd == 1 {
			success +=1
		} else {
			false += 1
		}
	}


	return success, false, nil
}

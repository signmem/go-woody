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

func domainAdd(r *http.Request) (htmlMsg ReturnMsg, err error) {

	// 只处理 dns 域名增加功能

	if !isContentTypeJson(r) {
		msg := fmt.Errorf("[v2-domain-add] domainAdd() Error: body not json format")
		g.Logger.Error(msg)
		htmlMsg.Msg = "domainAdd() Post data not valid, body not json format!"
		return htmlMsg, msg
	}

	defer func() {
		_ = r.Body.Close()
	}()

	body, err := io.ReadAll(r.Body)

	if err != nil {
		msg := fmt.Errorf("[v2-domain-add] domainAdd() Error: body read error")
		g.Logger.Error(msg)
		htmlMsg.Msg = "domainAdd() Post data not valid, body read error!"
		return htmlMsg, msg
	}

	var DomainList DomainCreate

	err = json.Unmarshal(body, &DomainList)

	if err != nil {
		msg := fmt.Errorf("[v2-domain-add] domainAdd() Error: body json unmarshal error")
		g.Logger.Error(msg)
		htmlMsg.Msg = "domainAdd() Post data not valid, body json unmarshal format error!"
		return htmlMsg, msg
	}

	TrimAllStrings(&DomainList)

	if len(DomainList.Domains) == 0 {
		msg := fmt.Errorf("[v2-domain-add] domainAdd() Error: DomainList empty")
		g.Logger.Error(msg)
		htmlMsg.Msg = "domainAdd() Post data not valid, DomainList empty!"
		return htmlMsg, msg
	}

	successAdd := 0
	falseAdd := 0

	// 修复: zone 条目统一收集, DB 提交后一次性加锁追加 + rndc reconfig
	addedDomains := make([]string, 0)

	if g.Config().Debug {
		g.Logger.Debugf("[v2-domain-add] domainAdd() add %s", DomainList.DomainString())
	}

	tx, err := db.DB.Begin()
	if err != nil {
		// 修复: 原实现用 fmt.Sprintf("...: %w", err), Sprintf 不支持 %w
		msg := fmt.Errorf("[v2-domain-add] domainAdd() Error: failed to begin transaction: %w", err)
		g.Logger.Error(msg)
		htmlMsg.Msg = msg.Error()
		return htmlMsg, err
	}

	rollbackNeeded := true
	defer func() {
		if rollbackNeeded {
			if rollbackErr := tx.Rollback(); rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
				g.Logger.Errorf("[v2-domain-add] domainAdd() rollback error: %v", rollbackErr)
			}
		}
	}()

	// master 单独取副本, 避免所有 SLAVE 行共享同一指针
	masterAddr := strings.TrimSpace(DomainList.Master)

	for _, domain := range DomainList.Domains {

		domain := strings.TrimSpace(domain)

		// 基础空值校验
		if domain == "" {
			g.Logger.Errorf("[v2-domain-add] domainAdd() Error: domain empty")
			falseAdd += 1
			continue
		}

		if !db.IsValidDomain(domain) {
			g.Logger.Errorf("[v2-domain-add] domainAdd() Error: %s not valid domain", domain)
			falseAdd += 1
			continue
		}

		subDomainLevel := db.GetDomainReverseLevels(domain)

		if !g.Config().AutoParent {
			subDomainLevel = []string{domain}
		}

		for _, subDomain := range subDomainLevel {

			if subDomain == "" {
				continue
			}

			domainDBInfo, err := db.GetDomainsByName(subDomain)

			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				g.Logger.Errorf("[v2-domain-add] domainAdd() %s db query err: %s", subDomain, err)
				falseAdd += 1
				continue
			}

			if domainDBInfo != nil {
				g.Logger.Infof("[v2-domain-add] domainAdd() %s already in db, skip", subDomain)
				falseAdd += 1
				continue
			}

			var domainDB db.Domain
			if masterAddr == "" {
				domainDB.Type = "MASTER"
				domainDB.Name = subDomain
				domainDB.Master = nil
			} else {
				domainDB.Type = "SLAVE"
				domainDB.Name = subDomain
				domainDB.Master = &masterAddr
			}

			domainID, err := db.InsertDomain(tx, domainDB)

			if err != nil || domainID == 0 {
				g.Logger.Errorf("[v2-domain-add] domainAdd() Error: InsertDomain %s error: %v", subDomain, err)
				falseAdd += 1
				continue
			}

			if domainDB.Type == "MASTER" {
				if _, _, err = DomainMetaDataAdd(tx, domainID); err != nil {
					g.Logger.Errorf("[v2-domain-add] domainAdd() metadata fail with domain %s: %v", subDomain, err)
					falseAdd += 1
					continue
				}
			}

			addedDomains = append(addedDomains, subDomain)
			successAdd += 1
		}
	}

	if err = tx.Commit(); err != nil {
		msg := fmt.Errorf("[v2-domain-add] domainAdd() Error: commit failed: %w", err)
		g.Logger.Error(msg)
		htmlMsg.Msg = msg.Error()
		return htmlMsg, err
	}

	rollbackNeeded = false

	var addStatus DnsAddStatus
	addStatus.Success = successAdd
	addStatus.Failure = falseAdd

	htmlMsg.Msg = addStatus.String()

	g.Logger.Debugf("[v2-domain-add] domainAdd() add %s success", DomainList.DomainString())

	// DB 已提交, 同步 named zone; 失败时返回明确的"部分成功"信息
	if zoneErr := appendZoneForwards(addedDomains); zoneErr != nil {
		g.Logger.Errorf("[v2-domain-add] domainAdd() sync named zone failed: %v", zoneErr)
		htmlMsg.Msg = fmt.Sprintf("%s (WARNING: %d domains saved to DB, "+
			"but named zone sync failed: %v)", addStatus.String(), successAdd, zoneErr)
		return htmlMsg, zoneErr
	}

	return htmlMsg, nil
}

// DomainMetaDataAdd 为 MASTER 域名批量写入 AXFR/NOTIFY 元数据。
// 修复: 命名返回值 false 遮蔽内建标识符, 重命名为 succeeded/failed;
// INSERT IGNORE 命中已有记录时计入 failed (skip), 与原语义一致。
func DomainMetaDataAdd(tx *sql.Tx, domainID int64) (succeeded int, failed int, err error) {

	var domainDBMeta db.DomainMeta
	domainDBMeta.DomainID = domainID

	dnsServers := g.Config().DnsServer
	pdnsPort := g.Config().DNS.Port

	insert := func(kind, content string) error {
		domainDBMeta.Kind = kind
		domainDBMeta.Content = content

		affected, err := db.InsertDomainMetaData(tx, domainDBMeta)
		if err != nil {
			return err
		}

		if affected == 1 {
			succeeded++
		} else {
			// INSERT IGNORE 命中已有记录, 视为 skip
			failed++
		}
		return nil
	}

	for _, server := range dnsServers {
		if err := insert("ALLOW-AXFR-IPS", server); err != nil {
			return succeeded, failed, err
		}

		remoteContent := server + ":" + pdnsPort

		if err := insert("ALLOW-AXFR-FROM", remoteContent); err != nil {
			return succeeded, failed, err
		}

		if err := insert("ALSO-NOTIFY", remoteContent); err != nil {
			return succeeded, failed, err
		}
	}

	return succeeded, failed, nil
}

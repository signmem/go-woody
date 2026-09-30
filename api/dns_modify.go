package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/signmem/go-woody/db"
	"github.com/signmem/go-woody/g"
)

func dnsModify(r *http.Request) (record DNSHost, err error) {

	defer func() {
		_ = r.Body.Close()
	}()

	urlPath := strings.TrimPrefix(r.URL.Path, "/api/hosts/")
	pathParts := strings.Split(urlPath, "/")

	if len(pathParts) != 1 || pathParts[0] == "" {
		msg := fmt.Errorf("dnsModify() Error: path invalid")
		g.Logger.Error(msg)
		return record, msg
	}

	domainID, err := strconv.Atoi(pathParts[0])

	if err != nil {
		msg := fmt.Errorf("dnsModify() Error: path %s not valid number", pathParts[0])
		g.Logger.Error(msg)
		return record, msg
	}

	if r.ContentLength == 0 {
		msg := fmt.Errorf("dnsModify() Error: body is blank")
		g.Logger.Error(msg)
		return record, msg
	}

	if !isContentTypeJson(r) {
		msg := fmt.Errorf("dnsModify() Error: body not json format")
		g.Logger.Error(msg)
		return record, msg
	}

	body, err := io.ReadAll(r.Body)

	if err != nil {
		msg := fmt.Errorf("dnsModify() Error: body read error: %v", err)
		g.Logger.Error(msg)
		return record, msg
	}

	var hostDict HostParams
	err = json.Unmarshal(body, &hostDict)

	if err != nil {
		msg := fmt.Errorf("dnsModify() Error: body json unmarshal error: %v", err)
		g.Logger.Error(msg)
		return record, msg
	}

	TrimAllStrings(&hostDict)

	if !isIPv4(hostDict.IP) {
		msg := fmt.Errorf("dnsModify() Error: %s not valid ipaddress", hostDict.IP)
		g.Logger.Error(msg)
		return record, msg
	}

	tx, err := db.DB.Begin()
	if err != nil {
		msg := fmt.Errorf("dnsModify() Error: failed to begin transaction: %w", err)
		g.Logger.Error(msg)
		return record, msg
	}

	rollbackNeeded := true
	defer func() {
		if rollbackNeeded {
			if rollbackErr := tx.Rollback(); rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
				g.Logger.Errorf("dnsModify() transaction rollback error: %v", rollbackErr)
			}
		}
	}()

	var dnsModify db.Record

	dnsModify.DomainID = int64(domainID)
	dnsModify.Name = strings.TrimSpace(hostDict.Hostname)
	dnsModify.Content = strings.TrimSpace(hostDict.IP)

	if dnsModify.Name == "" || dnsModify.Content == "" {
		return record, fmt.Errorf("hostname or ip can not be empty")
	}

	// GetHostRecordsCount 这里查询只验证 domain_id + name 不会校验 content
	count, err := db.GetHostRecordsCount(dnsModify)
	if err != nil {
		g.Logger.Errorf("dnsModify() get host record count error: %v", err)
		return record, err
	}

	if count != 1 {
		msg := fmt.Errorf("dnsModify() Error: id: %d, hostname: %s, not found in DB",
			domainID, hostDict.Hostname)
		g.Logger.Error(msg)
		return record, msg
	}

	hostID, err := db.GetHostRecordsID(dnsModify)

	if err != nil {
		g.Logger.Errorf("dnsModify() get host record id error: %v", err)
		return record, err
	}

	dnsModify.ID = hostID
	dnsModify.TTL = 30
	dnsModify.Type = "A"

	if _, err = db.UpdateRecord(tx, dnsModify); err != nil {
		msg := fmt.Errorf("dnsModify() Error: update record error: %w", err)
		g.Logger.Error(msg)
		return record, msg
	}

	// 这里传入 hostname 属于劫持模式 不需要导入完整 domain
	if err = db.UpdateSOA(tx, dnsModify.Name); err != nil {
		msg := fmt.Errorf("dnsModify() Error: update SOA error: %w", err)
		g.Logger.Error(msg)
		return record, msg
	}

	if err = tx.Commit(); err != nil {
		msg := fmt.Errorf("dnsModify() Error: db commit error: %w", err)
		g.Logger.Error(msg)
		return record, msg
	}
	rollbackNeeded = false

	g.Logger.Infof("dnsModify() update id: %d hostname: %s ipaddr: %s success",
		dnsModify.DomainID, dnsModify.Name, dnsModify.Content)

	dnsARecord, err := db.GetARecordsByDomainID(domainID)
	if err != nil {
		return record, err
	}

	record.IP = dnsARecord.Content
	record.Hostname = dnsARecord.Name
	record.ID = dnsARecord.DomainID

	return record, nil
}

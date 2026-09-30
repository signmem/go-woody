package api

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/signmem/go-woody/db"
	"github.com/signmem/go-woody/g"
)

func dnsDelete(r *http.Request) (record DNSHost, err error) {

	urlPath := strings.TrimPrefix(r.URL.Path, "/api/hosts/")
	pathParts := strings.Split(urlPath, "/")

	if len(pathParts) != 1 || pathParts[0] == "" {
		msg := fmt.Errorf("dnsDelete() Error: path error")
		g.Logger.Error(msg)
		return record, msg
	}

	domainID, err := strconv.Atoi(pathParts[0])

	if err != nil {
		msg := fmt.Errorf("dnsDelete() Error: path param %q not valid number", pathParts[0])
		g.Logger.Error(msg)
		return record, msg
	}

	tx, err := db.DB.Begin()
	if err != nil {
		msg := fmt.Errorf("dnsDelete() Error: failed to begin transaction: %w", err)
		g.Logger.Error(msg)
		return record, msg
	}

	rollbackNeeded := true
	defer func() {
		if rollbackNeeded {
			if rollbackErr := tx.Rollback(); rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
				g.Logger.Errorf("dnsDelete() transaction rollback error: %v", rollbackErr)
			}
		}
	}()

	aRecord, err := db.GetARecordsByDomainID(domainID)

	if err != nil || aRecord.DomainID != int64(domainID) {

		msg := fmt.Errorf("dnsDelete() Error: host %s not found in DB", pathParts[0])
		g.Logger.Error(msg)
		return record, msg
	}

	// 修复: 原错误信息格式化动词错误 (%d 配 string / 无动词多参),
	// 且丢失了底层 err; 现在统一 %w 包装
	if _, err = db.DeleteRecordByDomainID(tx, int64(domainID)); err != nil {
		msg := fmt.Errorf("dnsDelete() Error: host %s delete from DB failed: %w", pathParts[0], err)
		g.Logger.Error(msg)
		return record, msg
	}

	if err = tx.Commit(); err != nil {
		msg := fmt.Errorf("dnsDelete() Error: DB commit failed: %w", err)
		g.Logger.Error(msg)
		return record, msg
	}
	rollbackNeeded = false

	record.IP = aRecord.Content
	record.Hostname = aRecord.Name
	record.ID = aRecord.DomainID

	g.Logger.Infof("dnsDelete() delete hostname %s success", aRecord.Name)

	// 修复: DB 已提交, zone 移除走互斥 + rndc reconfig;
	// zone 条目不存在只记 warning, 不影响删除结果
	if zoneErr := removeZoneForwards([]string{aRecord.Name}); zoneErr != nil {
		g.Logger.Errorf("dnsDelete() remove zone entry for %s failed: %v", aRecord.Name, zoneErr)
		return record, fmt.Errorf("host deleted from DB, but named zone sync failed: %w", zoneErr)
	}

	return record, nil
}

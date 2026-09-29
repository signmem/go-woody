package api

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/signmem/go-woody/db"
	"github.com/signmem/go-woody/g"
)

func domainDelete(r *http.Request) (domaininfo DomainInfo, err error) {

	defer func() {
		_ = r.Body.Close()
	}()

	urlPath := strings.TrimPrefix(r.URL.Path, "/api/v2/domains/")
	pathParts := strings.Split(urlPath, "/")

	if len(pathParts) != 1 || pathParts[0] == "" {
		msg := fmt.Errorf("domainDelete() Error: path error")
		g.Logger.Error(msg)
		return domaininfo, msg
	}

	domainName := pathParts[0]

	domainDetail, err := db.GetDomainsByName(domainName)

	if err != nil || domainDetail == nil || domainDetail.ID == 0 {
		msg := fmt.Errorf("domainDelete() Error: domain %s not found in DB", domainName)
		g.Logger.Error(msg)
		return domaininfo, msg
	}

	// 修复: 先填充返回结构。原实现在删除 named zone 条目时引用
	// domaininfo.DomainName, 而该字段在函数末尾才赋值, 导致永远以空域名
	// 调用 RemoveZoneFromFile, zone 条目无法被删除且错误被吞掉
	domaininfo.DomainID = domainDetail.ID
	domaininfo.DomainName = domainDetail.Name

	tx, err := db.DB.Begin()
	if err != nil {
		msg := fmt.Errorf("domainDelete() Error: failed to begin transaction: %w", err)
		g.Logger.Error(msg)
		return domaininfo, msg
	}

	rollbackNeeded := true
	defer func() {
		if rollbackNeeded {
			if rollbackErr := tx.Rollback(); rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
				g.Logger.Errorf("domainDelete() transaction rollback error: %v", rollbackErr)
			}
		}
	}()

	// DeleteRecordByDomainID 同时删除 records 与 domains 表中的对应行
	if _, err = db.DeleteRecordByDomainID(tx, domainDetail.ID); err != nil {
		// 修复: 原错误信息 %d/%w 动词与参数不匹配且丢失底层 err
		msg := fmt.Errorf("domainDelete() Error: domain id %d delete records failed: %w",
			domainDetail.ID, err)
		g.Logger.Error(msg)
		return domaininfo, msg
	}

	if err = db.DeleteDomainMetaData(tx, domainDetail.ID); err != nil {
		msg := fmt.Errorf("domainDelete() Error: domain id %d delete metadata failed: %w",
			domainDetail.ID, err)
		g.Logger.Error(msg)
		return domaininfo, msg
	}

	if err = tx.Commit(); err != nil {
		msg := fmt.Errorf("domainDelete() Error: DB commit failed: %w", err)
		g.Logger.Error(msg)
		return domaininfo, msg
	}
	rollbackNeeded = false

	g.Logger.Infof("domainDelete() delete domain %s success", domaininfo.DomainName)

	// DB 已提交, 同步移除 named zone 条目 (互斥 + rndc reconfig);
	// 条目不存在只记 warning, 不影响删除结果
	if zoneErr := removeZoneForwards([]string{domainDetail.Name}); zoneErr != nil {
		g.Logger.Errorf("domainDelete() remove zone entry for %s failed: %v",
			domainDetail.Name, zoneErr)
		return domaininfo, fmt.Errorf("domain deleted from DB, but named zone sync failed: %w", zoneErr)
	}

	return domaininfo, nil
}

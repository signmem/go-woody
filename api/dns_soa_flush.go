package api

import (
	"encoding/json"
	"fmt"
	"github.com/signmem/go-woody/db"
	"github.com/signmem/go-woody/g"
	"io"
	"net/http"
)

func domainSOAFlush(r *http.Request) (v interface{}, err error) {

	if !isContentTypeJson(r) {
		msg := fmt.Errorf("domainSOAFlush() Error: body not json format")
		g.Logger.Error(msg)
		return nil, msg
	}

	defer func() {
		_ = r.Body.Close()
	}()

	body, err := io.ReadAll(r.Body)

	if err != nil {
		msg := fmt.Errorf("domainSOAFlush() Error: body read error")
		g.Logger.Error(msg)
		return nil, msg
	}

	var domainInfo DomainInfo
	err = json.Unmarshal(body, &domainInfo)

	if err != nil {
		msg := fmt.Errorf("domainSOAFlush() json format error.")
		g.Logger.Error(msg)
		return nil, msg
	}

	TrimAllStrings(&domainInfo)

	if domainInfo.DomainID == 0 && domainInfo.DomainName == "" {
		msg := fmt.Errorf("domainSOAFlush() Error: domain_id and domain_name is None")
		g.Logger.Error(msg)
		return nil, msg
	}

	var domain *db.Domain

	if domainInfo.DomainID > 0 {
		domain, err = db.GetDomainByID(domainInfo.DomainID)

		if err != nil {
			msg := fmt.Errorf("domainSOAFlush() get soa err:%s", err)
			g.Logger.Error(msg)
			return nil, msg
		}
	}

	if domainInfo.DomainID == 0 && domainInfo.DomainName != "" {
		domain, err = db.GetDomainsByName(domainInfo.DomainName)
		if err != nil {
			msg := fmt.Errorf("domainSOAFlush() get soa err:%s", err)
			g.Logger.Error(msg)
			return nil, msg
		}
	}

	if domain == nil || domain.ID < 1 {
		msg := fmt.Errorf("domainSOAFlush() get domain soa err:")
		g.Logger.Error(msg)
		return nil, msg
	}

	tx, err := db.DB.Begin()
	if err != nil {
		msg := fmt.Errorf("domainSOAFlush() Error: failed to begin transaction")
		g.Logger.Error(msg)
		return nil, msg
	}

	rollbackNeeded := true
	defer func() {
		if rollbackNeeded {
			_ = tx.Rollback()
		}
	}()

	err = db.UpdateSOA(tx, domain.Name)

	if err != nil {
		msg := fmt.Errorf("domainSOAFlush() Error: update soa err: %s", err)
		g.Logger.Error(msg)
		return nil, msg
	}

	if err = tx.Commit(); err != nil {
		msg := fmt.Errorf("domainSOAFlush() Error: db commit error: %s", err)
		g.Logger.Error(msg)
		return nil, msg
	}

	rollbackNeeded = false

	return db.GetDomainSOAByName(domain.Name)
}

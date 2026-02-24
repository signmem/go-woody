package api

import (
	"database/sql"
	"fmt"
	"github.com/signmem/go-woody/db"
	"github.com/signmem/go-woody/g"
	"io"
	"net/http"
	"strconv"
	"strings"
	"encoding/json"
)

func dnsModify(r *http.Request) (record DNSHost, err error) {

	path := strings.TrimPrefix(r.URL.Path, "/api/hosts/")
	pathParts := strings.Split(path, "/")

	if len(pathParts) != 1 || pathParts[0] == "" {
		msg := fmt.Errorf("Error: path invalid")
		g.Logger.Error(msg)
		return record, msg
	}

	domain_id, err :=  strconv.Atoi(pathParts[0])

	if err != nil {
		msg := fmt.Errorf("Error: path %s not valid number.", pathParts[0])
		g.Logger.Error(msg)
		return record, msg
	}

	if r.ContentLength == 0 {
		msg := fmt.Errorf("Error: body is blank")
		g.Logger.Error(msg)
		return record, msg
	}

	headerContentTtype := r.Header.Get("Content-Type")
	if headerContentTtype != "application/json" {
		msg := fmt.Errorf("Error: body not json format")
		g.Logger.Error(msg)
		return record, msg
	}

	body, err := io.ReadAll(r.Body)

	if err != nil {
		msg := fmt.Errorf("Error: body read error")
		g.Logger.Error(msg)
		return record, msg
	}

	var hostDict HostParams
	err = json.Unmarshal(body, &hostDict)

	if err != nil {
		msg := fmt.Errorf("dnsModify() Error: body json unmarshar error")
		g.Logger.Error(msg)
		return record, msg
	}


	if isIPv4(hostDict.IP) == false {
		msg := fmt.Errorf("Error: %s not valid ipaddress", hostDict.IP)
		g.Logger.Error(msg)
		return record, msg
	}

	tx, err := db.DB.Begin()
	if err != nil {
		msg := fmt.Errorf("dnsModify() Error: failed to begin transaction")
		g.Logger.Error(msg)
		return record, err
	}

	defer func() {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			msg := fmt.Sprintf("dnsModify() Error: transaction rollback error")
			g.Logger.Errorf(msg)
		}
	}()

	var dnsModify db.Record

	dnsModify.DomainID  = int64(domain_id)
	dnsModify.Name      = hostDict.Hostname
	dnsModify.Content   = hostDict.IP

	count, _ := db.GetHostRecordsCount(dnsModify)

	if count != 1 {
		msg := fmt.Errorf("Error: id: %d, hostname: %s, " +
			"not found in DB", domain_id,  hostDict.Hostname)
		g.Logger.Error(msg)
		return record, msg
	}

	hostID, _ := db.GetHostRecordsID(dnsModify)

	dnsModify.ID = hostID
	dnsModify.TTL = 30
	dnsModify.Type = "A"

	_, err = db.UpdateRecord(tx, dnsModify)
	if err != nil {
		tx.Rollback()
		msg := fmt.Errorf("Error: update record error: %s", err)
		g.Logger.Error(msg)
		return record, msg
	}

	err = db.UpdateSOA(tx, dnsModify.Name)
	if err != nil {
		tx.Rollback()
		msg := fmt.Errorf("Error: update SOA error: %s", err)
		g.Logger.Error(msg)
		return record, msg
	}

	if err = tx.Commit(); err != nil {
		msg := fmt.Errorf("Error: db commit error: %s", err)
		g.Logger.Error(msg)
		return record, msg
	}

	msg := fmt.Sprintf("Update id: %d hostname: %s " +
		" ipaddr: %s success.", dnsModify.DomainID, dnsModify.Name, dnsModify.Content)

	if g.Config().Debug == true {
		g.Logger.Debug(msg)
	}

	dnsARecord , err := db.GetARecordsByDomainID(domain_id)
	if err != nil {
		return record, err
	}

	record.IP       = dnsARecord.Content
	record.Hostname = dnsARecord.Name
	record.ID       = dnsARecord.DomainID

	return record, nil
}

package api

import (
	"database/sql"
	"fmt"
	"github.com/signmem/go-woody/db"
	"github.com/signmem/go-woody/g"
	"net/http"
	"strconv"
	"strings"
)

func dnsDelete(r *http.Request)  (record DNSHost, err error)  {


	path := strings.TrimPrefix(r.URL.Path, "/api/hosts/")
	pathParts := strings.Split(path, "/")

	if len(pathParts) != 1 || pathParts[0] == "" {
		msg := fmt.Errorf("Error: path error")
		g.Logger.Error(msg)
		return record, msg
	}

	domain_id, err :=  strconv.Atoi(pathParts[0])

	if err != nil {

		msg := fmt.Errorf("Error: path params not valid.")
		g.Logger.Error(msg)
		return record, msg

	}


	tx, err := db.DB.Begin()
	if err != nil {

		msg := fmt.Errorf("Error: failed to begin transaction")
		g.Logger.Error(msg)
		return record, msg

	}

	defer func() {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			msg := fmt.Sprintf("dnsDelete() Error: transaction rollback error")
			g.Logger.Errorf(msg)
		}
	}()

	aRecord, err := db.GetARecordsByDomainID(domain_id)
	if err != nil || aRecord.DomainID != int64(domain_id)  {

		msg := fmt.Errorf("Error: Host %d not found in DB.", pathParts[0])
		g.Logger.Error(msg)
		return record, msg

	}

	_, err = db.DeleteRecordByDomainID(tx, int64(domain_id))

	if err != nil {
		tx.Rollback()

		msg := fmt.Errorf("Error: Host %d delete from DB.", pathParts[0])
		g.Logger.Error(msg)
		return record, msg

	}

	if err = tx.Commit(); err != nil {

		msg := fmt.Errorf("Error: DB commit.", pathParts[0])
		g.Logger.Error(msg)
		return record, msg

	}

	msg := fmt.Sprintf("dnsDelete() delete hostname %s Success", aRecord.Name)

	if g.Config().Debug == true {
		g.Logger.Debug(msg)
	}


	record.IP        = aRecord.Content
	record.Hostname  = aRecord.Name
	record.ID        = aRecord.DomainID
	return record, nil
}


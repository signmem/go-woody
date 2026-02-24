package api

import (
	"database/sql"
	"errors"
	"github.com/signmem/go-woody/db"
)

func dnsHostAdd(tx *sql.Tx, domain_id int64, hostName string,
	ipAddr string)  (id int64, err error) {

	if tx == nil {
		return 0, errors.New("transaction is nil")
	}

	if hostName == "" || ipAddr == "" {
		return 0, errors.New("dnsHostAdd(): hostname, IP address cannot be empty")
	}


	var dnsRecord  db.Record

	dnsRecord.TTL = 30
	dnsRecord.Content = ipAddr
	dnsRecord.Type  = "A"
	dnsRecord.Name  = hostName
	dnsRecord.DomainID = domain_id

	return db.InsertRecord(tx, dnsRecord)
	
}


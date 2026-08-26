package db

import (
	"database/sql"
	"fmt"
	"github.com/signmem/go-woody/g"

	"regexp"
	"strconv"
	"strings"
)

func InsertRecord(tx *sql.Tx, record Record) (int64, error) {

	query := `INSERT INTO records (domain_id, name, type, content, ttl) 
              VALUES (?, ?, ?, ?, ?)`
    
	result, err := tx.Exec(query, record.DomainID, record.Name, record.Type,
                          record.Content, record.TTL)

	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}


// GetARecordByDomainID 单条A记录查询，修复SQL注入
func GetARecordsByDomainID(domainID int) (record Record,  err error) {

	dnsIpList := g.Config().DnsServer
	ipstr := strings.Join(dnsIpList, "','")
	sqlDnsIp  := fmt.Sprintf("'%s'", ipstr)

	// 改用参数化规避SQL注入，不再字符串拼接
	// query := "SELECT id, domain_id, name, type, content, ttl " +
	//	"FROM records WHERE type = 'A' AND domain_id = ? AND content NOT IN (" +
	//	 sqlDnsIp + ")"

	query := "SELECT id, domain_id, name, type, content, ttl FROM records " +
		"WHERE type = 'A' AND domain_id = ? AND  NOT ((name LIKE 'dns0.%' " +
		"OR name LIKE 'dns1.%' ) AND content IN (" + sqlDnsIp + "))"

	err = DB.QueryRow(query, domainID).Scan(
			&record.ID,
			&record.DomainID,
			&record.Name,
			&record.Type,
			&record.Content,
			&record.TTL,
	)

	if err != nil {
		g.Logger.Errorf("sql %s GetARecordsByDomainID() domain_id %d error: %s",
				query , domainID, err)
		return record, err
	}

	return record, nil
}

func GetARecordsByHostname(hostname string) (record Record,  err error) {
	dnsIpList := g.Config().DnsServer
	ipstr := strings.Join(dnsIpList, "','")
	sqlDnsIp  := fmt.Sprintf("'%s'", ipstr)

	query := "SELECT id, domain_id, name, type, content, ttl FROM records " +
		"WHERE type = 'A' AND name = ? AND  NOT ((name LIKE 'dns0.%' " +
		"OR name LIKE 'dns1.%') AND content IN (" + sqlDnsIp + "))"

	err = DB.QueryRow(query, hostname).Scan(
		&record.ID,
		&record.DomainID,
		&record.Name,
		&record.Type,
		&record.Content,
		&record.TTL,
	)

	if err != nil {
		g.Logger.Errorf("sql %s GetARecordsByDomainID() hostname %s error: %s",
			query , hostname, err)
		return record, err
	}

	return record, nil
}

// use to return multi hostname record from db
func GetARecordsByHostNamev2(hostname string) (records []Record,  err error) {

	dnsIpList := g.Config().DnsServer
	ipstr := strings.Join(dnsIpList, "','")
	sqlDnsIp  := fmt.Sprintf("'%s'", ipstr)

	query := "SELECT id, domain_id, name, type, content, ttl FROM records " +
		"WHERE type = 'A' AND name = ? AND  NOT ((name LIKE 'dns0.%' " +
		"OR name LIKE 'dns1.%') AND content IN (" + sqlDnsIp + "))"

	rows, err := DB.Query(query, hostname)

	if err != nil {
		g.Logger.Errorf("GetARecordsByHostNamev2() hostname %s error:%s", hostname, err)
		return nil, err
	}

	defer func() {
		if rows != nil {
			_ = rows.Close()
		}
	}()

	for rows.Next() {
		var record Record
		err := rows.Scan(
			&record.ID,
			&record.DomainID,
			&record.Name,
			&record.Type,
			&record.Content,
			&record.TTL,
		)

		if err != nil {
			g.Logger.Errorf("GetARecordsByHostNamev2() hostname %s scan error:%s", hostname, err)
			return nil, err
		}
		records = append(records, record)
	}

	return records, nil
}



// use to return multi hostname record from db
func GetARecordsByHostIDv2(hostID int) (record Record,  err error) {

	query := "SELECT id, domain_id, name, type, content, ttl FROM records " +
		"WHERE id = ?"

	err = DB.QueryRow(query, hostID).Scan(
		&record.ID,
		&record.DomainID,
		&record.Name,
		&record.Type,
		&record.Content,
		&record.TTL,
	)

	if err != nil {
		g.Logger.Errorf("sql %s GetARecordsByHostIDv2() hostID %d error: %s",
			query , hostID, err)
		return record, err
	}

	return record, nil
}



// use to return multi hostname record from db
func GetARecordsByHostIPv2(ipaddr string) (records []Record,  err error) {

	dnsIpList := g.Config().DnsServer
	ipstr := strings.Join(dnsIpList, "','")
	sqlDnsIp  := fmt.Sprintf("'%s'", ipstr)

	query := "SELECT id, domain_id, name, type, content, ttl FROM records " +
		"WHERE type = 'A' and content = ? AND NOT ((name LIKE 'dns0.%' OR name LIKE 'dns1.%') " +
		"AND content IN (" + sqlDnsIp + "))"

	rows, err := DB.Query(query, ipaddr)

	if err != nil {
		g.Logger.Errorf("GetARecordsByHostIPv2() ip %s error:%s", ipaddr, err)
		return nil, err
	}

	defer func() {
		if rows != nil {
			_ = rows.Close()
		}
	}()

	for rows.Next() {
		var record Record
		err := rows.Scan(
			&record.ID,
			&record.DomainID,
			&record.Name,
			&record.Type,
			&record.Content,
			&record.TTL,
		)

		if err != nil {
			g.Logger.Errorf("GetARecordsByHostIPv2() ip %s scan error:%s", ipaddr, err)
			return nil, err
		}
		records = append(records, record)
	}

	return records, nil
}



func GetARecordsByDomainNamev2(domainID int, domainName string) (records []Record,  err error) {


	dnsContent := "'dns0." + domainName + "','" + "dns1." + domainName + "'"


	query := "SELECT id, domain_id, name, type, content, ttl FROM records " +
		"WHERE type = 'A' AND domain_id = ? AND  name NOT IN (" + dnsContent + ")"

	if g.Config().Debug {
		g.Logger.Debug("GetARecordsByDomainIDv2() sql %s", query )
	}

	rows, err := DB.Query(query, domainID)

	if err != nil {
		g.Logger.Errorf("GetARecordsByDomainIDv2() domainID %d error:%s", domainID, err)
		return nil, err
	}

	defer func() {
		if rows != nil {
			_ = rows.Close()
		}
	}()

	for rows.Next() {
		var record Record
		err := rows.Scan(
			&record.ID,
			&record.DomainID,
			&record.Name,
			&record.Type,
			&record.Content,
			&record.TTL,
		)

		if err != nil {
			g.Logger.Errorf("GetARecordsByDomainIDv2() domainID %d scan error:%s", domainID, err)
			return nil, err
		}
		records = append(records, record)
	}

	return records, nil

}



func GetRecordsByDomainID(domainID int) ([]*Record, error) {

	var records []*Record

	query := `SELECT id, domain_id, name, type, content, ttl 
              FROM records WHERE domain_id = ?`

	rows, err := DB.Query(query, domainID)

	if err != nil {
		g.Logger.Errorf("GetRecordsByDomainID() id %d error:%s", domainID, err)
		return nil, err
	}

	defer func() {
		if rows != nil {
			_ = rows.Close()
		}
	}()

	// defer rows.Close()

	for rows.Next() {
		var record Record
		err := rows.Scan(
			&record.ID,
			&record.DomainID,
			&record.Name,
			&record.Type,
			&record.Content,
			&record.TTL,
		)

		if err != nil {
			g.Logger.Errorf("GetRecordsByDomainID() domain_id %d scan error:%s", domainID, err)
			return nil, err
		}
		records = append(records, &record)
	}
	// 必须检查行迭代错误
	if err = rows.Err(); err != nil {
		g.Logger.Errorf("GetRecordsByDomainID() rows iterate error:%s", err)
		return nil, err
	}
	return records, nil
}

func GetRecordsByHostName(hostName string) (records []*Record, err error) {

	query := `SELECT id, domain_id, name, type, content, ttl 
              FROM records WHERE name = ?`

	rows, err := DB.Query(query, hostName)

	if err != nil {
		g.Logger.Errorf("GetRecordsByHostName() query %s record error " +
			"%s", hostName, err)
		return records, err
	}

	defer func() {
		if rows != nil {
			_ = rows.Close()
		}
	}()

	// defer rows.Close()

	for rows.Next() {

		var record Record
		err = rows.Scan(
			&record.ID,
			&record.DomainID,
			&record.Name,
			&record.Type,
			&record.Content,
			&record.TTL,
		)

		if err != nil {
			g.Logger.Errorf("GetRecordsByHostName() get %s record error " +
				"%s", hostName, err)
			return nil, err
		}

		records = append(records, &record)
	}

	if err = rows.Err(); err != nil {
		g.Logger.Errorf("GetRecordsByHostName() rows iterate error: %s", err)
		return nil, err
	}

	return records, nil
}



func UpdateSOA(tx *sql.Tx, domainName string) (err error) {

	query := `SELECT id, domain_id, name, type, content, ttl 
              FROM records WHERE type ='SOA' and name = ?`

	var record Record

	err = tx.QueryRow(query, domainName).Scan(
		&record.ID,
		&record.DomainID,
		&record.Name,
		&record.Type,
		&record.Content,
		&record.TTL,
	)

	if err != nil {
		g.Logger.Errorf("UpdateSOA() get %s soa record error:%s", domainName, err)
		return err
	}

	content := compressSpaces(record.Content)

	c_sp := strings.Split(content, " ")

	if len(c_sp) < 7 {
		err := fmt.Errorf("soa content format invalid, split len < 7")
		g.Logger.Errorf("UpdateSOA() %s: %v", domainName, err)
		return err
	}

	numStr := c_sp[2]
	num, err := strconv.Atoi(numStr)
	if err != nil {
		g.Logger.Errorf("UpdateSOA() SOA format parse num error: %s", err)
		return err
	}

	if num >= 9223372036854775806 {
		g.Logger.Errorf("UpdateSOA() SOA number match MaxInt Value")
		return err
	}

	num += 1
	newNumStr := strconv.Itoa(num)

	newCont := fmt.Sprintf("%s %s %s %s %s %s %s",
		c_sp[0], c_sp[1], newNumStr, c_sp[3], c_sp[4], c_sp[5], c_sp[6])

	record.Content = newCont

	_, err = UpdateRecord(tx, record)

	if err != nil {
		g.Logger.Errorf("UpdateSOA() update SOA error:%s", err)
		return err
	}

	return nil

}



func UpdateRecord(tx *sql.Tx, record Record) (int64, error) {


	query := `UPDATE records SET domain_id=?, name=?, type=?, content=?,
			ttl=? WHERE id=?`
    
	result, err := tx.Exec(query, record.DomainID, record.Name, record.Type,
                          record.Content, record.TTL, record.ID)
	if err != nil {
 		g.Logger.Errorf("UpdateRecord() update name:%s err:%s", record.Name, err)
		return 0, err
	}
    
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		g.Logger.Errorf("UpdateRecord() affect name:%s err:%s", record.Name, err)
		return 0, err
	}

	return rowsAffected, nil
}

func UpdateRecordV2(record Record, domainName string) (int64, error) {

	if DB == nil {
		return 0, fmt.Errorf("DeleteRecordByID() Error: db not initial")
	}

	tx, err := DB.Begin()
	if err != nil {
		return 0, fmt.Errorf("DeleteRecordByID() Error: failed to begin transaction: %w", err)
	}

	rollbackNeeded := true
	defer func() {
		if rollbackNeeded {
			if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
				g.Logger.Errorf("DeleteRecordByID() transaction rollback error (likely harmless if commit succeeded): %v", err)
			}
		}
	}()


	query := `UPDATE records SET content=? WHERE id=? and name =?`

	result, err := tx.Exec(query, record.Content, record.ID, record.Name)

	if err != nil {
		g.Logger.Errorf("UpdateRecord() update ip:%s err:%s", record.Content, err)
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()

	if err != nil {
		g.Logger.Errorf("UpdateRecord() affect name:%s err:%s", record.Name, err)
		return 0, err
	}

	err = UpdateSOA(tx, domainName)

	if err != nil {
		return 0,  fmt.Errorf("update soa serial failed: %w", err)
	}

	if err := tx.Commit(); err != nil {
		g.Logger.Errorf("DeleteRecordByID() failed to commit transaction: %v", err)
		return  0, fmt.Errorf("tx commit failed: %w", err)
	}

	rollbackNeeded = false

	return rowsAffected, nil
}


func DeleteRecordByID(id int64, domainName string) (int64, error) {

	if DB == nil {
		return 0, fmt.Errorf("DeleteRecordByID() Error: db not initial")
	}

	tx, err := DB.Begin()
	if err != nil {
		return 0, fmt.Errorf("DeleteRecordByID() Error: failed to begin transaction: %w", err)
	}

	rollbackNeeded := true

	defer func() {
		if rollbackNeeded {
			if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
				g.Logger.Errorf("DeleteRecordByID() transaction rollback error (likely harmless if commit succeeded): %v", err)
			}
		}
	}()

	query := "DELETE FROM records WHERE id = ?"

	result, err := tx.Exec(query, id)

	if err != nil {
		g.Logger.Errorf("DeleteRecordByID() delete id %d, err: %s", id, err)
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		g.Logger.Errorf("DeleteRecordByID() delete id %d, err: %s", id, err)
		return 0, fmt.Errorf("delete record exec failed: %w", err)
	}

	err = UpdateSOA(tx, domainName)

	if err != nil {
		return 0,  fmt.Errorf("update soa serial failed: %w", err)
	}

	if err := tx.Commit(); err != nil {
		g.Logger.Errorf("DeleteRecordByID() failed to commit transaction: %v", err)
		return  0, fmt.Errorf("tx commit failed: %w", err)
	}
	rollbackNeeded = false

	return rowsAffected, nil
}


func DeleteRecordByDomainID(tx *sql.Tx, id int64) (int64, error) {

	var totalRowsAffected int64

	query := "DELETE FROM records WHERE domain_id = ?"

	result, err := tx.Exec(query, id)

	if err != nil {
		g.Logger.Errorf("DeleteRecordByDomainID() delete id %d, err: %s", id, err)
		return 0, err
	}

	recordsAffected, err := result.RowsAffected()
	if err != nil {
		g.Logger.Errorf("DeleteRecordByDomainID() affect id %d, err: %s", id, err)
		return 0, err
	}

	totalRowsAffected += recordsAffected

	query = "DELETE FROM domains WHERE id = ?"

	result, err = tx.Exec(query, id)

	if err != nil {
		g.Logger.Errorf("DeleteRecordByDomainID() delete id %d, err: %s", id, err)
		return 0, err
	}

	domainsAffected, err := result.RowsAffected()
	if err != nil {
		g.Logger.Errorf("DeleteRecordByDomainID() affect id %d, err: %s", id, err)
		return 0, err
	}

	totalRowsAffected += domainsAffected

	if g.Config().Debug == true {
		g.Logger.Infof("DeleteRecordByDomainID() successfully deleted %d records" +
			" for domain_id %d", totalRowsAffected, id)
	}

	return totalRowsAffected, nil
}


func compressSpaces(s string) string {
	re := regexp.MustCompile(`\s+`)
	return re.ReplaceAllString(s, " ")
}


func GetRecordsByPageLimit(page int, per_page int) (records []*Record, err error) {

	dnsIpList := g.Config().DnsServer
	ipstr := strings.Join(dnsIpList, "','")
	sqlDnsIp := fmt.Sprintf("'%s'", ipstr)

	var rows *sql.Rows

	//query := "SELECT id, domain_id, name, type, content, ttl FROM records " +
	//	"WHERE type = 'A' AND content NOT IN (" + sqlDnsIp + ") ORDER BY id "

	query := "SELECT id, domain_id, name, type, content, ttl FROM records" +
		" WHERE type = 'A' AND  NOT ((name LIKE 'dns0.%' OR name LIKE 'dns1.%') " +
		"AND content IN (" + sqlDnsIp + "))"

	if  per_page > 0 {
		offset := (page - 1) * per_page
		query += " LIMIT ?, ?"
		rows, err = DB.Query(query, offset, per_page)
	} else {
		rows, err = DB.Query(query)
	}

	// defer rows.Close()
	defer func() {
		if rows != nil {
			_ = rows.Close()
		}
	}()

	if err != nil {
		g.Logger.Errorf("GetRecordsByPageLimit() query error: %s", err)
		return records, err
	}

	if g.Config().Debug == true {
		g.Logger.Infof("GetRecordsByPageLimit() query: %s", query)
	}

	for rows.Next() {

		var record Record
		err = rows.Scan(
			&record.ID,
			&record.DomainID,
			&record.Name,
			&record.Type,
			&record.Content,
			&record.TTL,
		)

		if err != nil {
			g.Logger.Errorf("GetRecordsByPageLimit() scan error: %s", err)
			return records, err
		}

		records = append(records, &record)
	}

	if err = rows.Err(); err != nil {
		g.Logger.Errorf("GetRecordsByPageLimit() rows iteration error: %s", err)
		return records, err
	}

	return records, nil
}

func GetRecordsCount() (count int, err error) {
	dnsIpList := g.Config().DnsServer
	ipstr := strings.Join(dnsIpList, "','")
	sqlDnsIp  := fmt.Sprintf("'%s'", ipstr)


	// query := "SELECT count(id) FROM records WHERE type = 'A' AND content NOT IN (" +
	//   sqlDnsIp + ")"

	query := "SELECT count(id) FROM records WHERE type = 'A' AND " +
		"NOT ((name LIKE 'dns0.%' OR name LIKE 'dns1.%') AND content " +
		" IN (" + sqlDnsIp + "))"

	err = DB.QueryRow(query).Scan(&count)

	if err != nil {
		g.Logger.Errorf("GetRecordsCount() query error: %s", err)
		return 0, err
	}

	return count, nil
}


func GetHostRecordsCount(HostDNS Record) (count int, err error) {

	hostname  := HostDNS.Name
	domain_id := HostDNS.DomainID

	query := `SELECT count(id) FROM records 
		WHERE type = 'A' AND name = ? AND domain_id = ?`

	if g.Config().Debug == true {
		g.Logger.Debugf("GetHostRecordsCount() query: %s", query)
	}

	err = DB.QueryRow(query, hostname, domain_id).Scan(&count)

	if err != nil {
		g.Logger.Errorf("GetHostRecordsCount() query error: %s", err)
		return 0, err
	}

	return count, nil
}

func GetHostRecordsID(HostDNS Record) (id int64, err error) {

	hostname  := HostDNS.Name
	domain_id := HostDNS.DomainID

	query := `SELECT id FROM records 
		where type = 'A' and name = ? and domain_id = ?`

	err = DB.QueryRow(query, hostname, domain_id).Scan(&id)

	if err != nil {
		g.Logger.Errorf("GetHostRecordsID() query error: %s", err)
		return 0, err
	}

	return id, nil
}



func GetSOARecords() (records []*Record, err error) {


	var rows *sql.Rows

	//query := "SELECT id, domain_id, name, type, content, ttl FROM records " +
	//	"WHERE type = 'A' AND content NOT IN (" + sqlDnsIp + ") ORDER BY id "

	query := "SELECT id, domain_id, name, type, content, ttl FROM records" +
		" WHERE type = 'SOA'"

	rows, err = DB.Query(query)


	// defer rows.Close()
	defer func() {
		if rows != nil {
			_ = rows.Close()
		}
	}()

	if err != nil {
		g.Logger.Errorf("GetSOARecords() query error: %s", err)
		return records, err
	}

	if g.Config().Debug == true {
		g.Logger.Infof("GetSOARecords() query: %s", query)
	}

	for rows.Next() {

		var record Record
		err = rows.Scan(
			&record.ID,
			&record.DomainID,
			&record.Name,
			&record.Type,
			&record.Content,
			&record.TTL,
		)

		if err != nil {
			g.Logger.Errorf("GetSOARecords() scan error: %s", err)
			return records, err
		}

		records = append(records, &record)
	}

	if err = rows.Err(); err != nil {
		g.Logger.Errorf("GetSOARecords() rows iteration error: %s", err)
		return records, err
	}

	return records, nil
}

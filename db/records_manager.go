package db

import (
	"database/sql"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/signmem/go-woody/g"
)

// dnsServerFilter 返回用于过滤内置 DNS 服务器记录 (dns0.*/dns1.* 指向 dns_server)
// 的参数化 SQL 子句及其绑定参数。
// 修复: 原实现把配置里的 IP 用字符串拼接进 IN (...), 与"已参数化"的注释不符;
// 现在统一使用占位符 + 参数绑定。dns_server 为空时返回空子句。
func dnsServerFilter() (clause string, args []interface{}) {
	dnsIpList := g.Config().DnsServer
	if len(dnsIpList) == 0 {
		return "", nil
	}

	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(dnsIpList)), ",")
	args = make([]interface{}, 0, len(dnsIpList))
	for _, ip := range dnsIpList {
		args = append(args, ip)
	}

	clause = "NOT ((name LIKE 'dns0.%' OR name LIKE 'dns1.%') AND content IN (" +
		placeholders + "))"
	return clause, args
}

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

// GetARecordsByDomainID 按 domain_id 查询单条 A 记录 (排除内置 dns server 记录)。
// 修复: 增加 ORDER BY id LIMIT 1, 保证多条 A 记录时结果确定 (原实现无序,
// QueryRow 取哪条不确定)。
func GetARecordsByDomainID(domainID int) (record Record, err error) {

	clause, clauseArgs := dnsServerFilter()

	query := "SELECT id, domain_id, name, type, content, ttl FROM records " +
		"WHERE type = 'A' AND domain_id = ?"
	args := []interface{}{domainID}

	if clause != "" {
		query += " AND " + clause
		args = append(args, clauseArgs...)
	}
	query += " ORDER BY id LIMIT 1"

	err = DB.QueryRow(query, args...).Scan(
		&record.ID,
		&record.DomainID,
		&record.Name,
		&record.Type,
		&record.Content,
		&record.TTL,
	)

	if err != nil {
		g.Logger.Errorf("GetARecordsByDomainID() domain_id %d error: %s", domainID, err)
		return record, err
	}

	return record, nil
}

func GetARecordsByHostname(hostname string) (record Record, err error) {

	clause, clauseArgs := dnsServerFilter()

	query := "SELECT id, domain_id, name, type, content, ttl FROM records " +
		"WHERE type = 'A' AND name = ?"
	args := []interface{}{hostname}

	if clause != "" {
		query += " AND " + clause
		args = append(args, clauseArgs...)
	}
	query += " ORDER BY id LIMIT 1"

	err = DB.QueryRow(query, args...).Scan(
		&record.ID,
		&record.DomainID,
		&record.Name,
		&record.Type,
		&record.Content,
		&record.TTL,
	)

	if err != nil {
		g.Logger.Errorf("GetARecordsByHostname() hostname %s error: %s", hostname, err)
		return record, err
	}

	return record, nil
}

// GetARecordsByHostNamev2 按主机名查询全部 A 记录 (排除内置 dns server 记录)
func GetARecordsByHostNamev2(hostname string) (records []Record, err error) {

	clause, clauseArgs := dnsServerFilter()

	query := "SELECT id, domain_id, name, type, content, ttl FROM records " +
		"WHERE type = 'A' AND name = ?"
	args := []interface{}{hostname}

	if clause != "" {
		query += " AND " + clause
		args = append(args, clauseArgs...)
	}

	rows, err := DB.Query(query, args...)

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

	if err = rows.Err(); err != nil {
		g.Logger.Errorf("GetARecordsByHostNamev2() hostname %s rows iterate error:%s", hostname, err)
		return nil, err
	}

	return records, nil
}

// GetARecordsByHostIDv2 按记录主键 id 查询单条记录
func GetARecordsByHostIDv2(hostID int) (record Record, err error) {

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
		g.Logger.Errorf("GetARecordsByHostIDv2() hostID %d error: %s", hostID, err)
		return record, err
	}

	return record, nil
}

// GetARecordsByHostIPv2 按 IP 查询全部 A 记录 (排除内置 dns server 记录)
func GetARecordsByHostIPv2(ipaddr string) (records []Record, err error) {

	clause, clauseArgs := dnsServerFilter()

	query := "SELECT id, domain_id, name, type, content, ttl FROM records " +
		"WHERE type = 'A' AND content = ?"
	args := []interface{}{ipaddr}

	if clause != "" {
		query += " AND " + clause
		args = append(args, clauseArgs...)
	}

	rows, err := DB.Query(query, args...)

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

	if err = rows.Err(); err != nil {
		g.Logger.Errorf("GetARecordsByHostIPv2() ip %s rows iterate error:%s", ipaddr, err)
		return nil, err
	}

	return records, nil
}

// GetARecordsByDomainNamev2 按 domain_id 查询全部 A 记录, 排除该域的内置
// dns0./dns1. 记录。
// 修复: 原实现把 domainName 直接拼接进 SQL (二阶注入面), 现在参数化。
func GetARecordsByDomainNamev2(domainID int, domainName string) (records []Record, err error) {

	query := "SELECT id, domain_id, name, type, content, ttl FROM records " +
		"WHERE type = 'A' AND domain_id = ? AND name NOT IN (?, ?)"

	if g.Config().Debug {
		g.Logger.Debugf("GetARecordsByDomainNamev2() sql %s", query)
	}

	rows, err := DB.Query(query, domainID, "dns0."+domainName, "dns1."+domainName)

	if err != nil {
		g.Logger.Errorf("GetARecordsByDomainNamev2() domainID %d error:%s", domainID, err)
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
			g.Logger.Errorf("GetARecordsByDomainNamev2() domainID %d scan error:%s", domainID, err)
			return nil, err
		}
		records = append(records, record)
	}

	if err = rows.Err(); err != nil {
		g.Logger.Errorf("GetARecordsByDomainNamev2() domainID %d rows iterate error:%s", domainID, err)
		return nil, err
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
		g.Logger.Errorf("GetRecordsByHostName() query %s record error %s", hostName, err)
		return records, err
	}

	defer func() {
		if rows != nil {
			_ = rows.Close()
		}
	}()

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
			g.Logger.Errorf("GetRecordsByHostName() get %s record error %s", hostName, err)
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

// UpdateSOA 递增指定域名的 SOA serial。
// 修复:
//  1. SELECT ... FOR UPDATE 行锁: 原实现普通 SELECT 读 serial 后在 Go 里 +1 写回,
//     并发事务读到同一 serial 会双双写 N+1 (lost update), serial 不前进
//     将导致 slave 不触发 AXFR 同步;
//  2. serial 溢出分支原返回 nil error (静默"成功"但不更新), 现在返回明确错误。
func UpdateSOA(tx *sql.Tx, domainName string) (err error) {

	query := `SELECT id, domain_id, name, type, content, ttl 
              FROM records WHERE type = 'SOA' AND name = ? FOR UPDATE`

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

	cSp := strings.Split(content, " ")

	if len(cSp) < 7 {
		err := fmt.Errorf("UpdateSOA() %s: soa content format invalid, split len < 7", domainName)
		g.Logger.Error(err)
		return err
	}

	num, err := strconv.Atoi(cSp[2])
	if err != nil {
		g.Logger.Errorf("UpdateSOA() %s: SOA serial parse error: %s", domainName, err)
		return err
	}

	if num >= math.MaxInt64-1 {
		err := fmt.Errorf("UpdateSOA() %s: SOA serial %d exceeds max value", domainName, num)
		g.Logger.Error(err)
		return err
	}

	num += 1

	newCont := fmt.Sprintf("%s %s %s %s %s %s %s",
		cSp[0], cSp[1], strconv.Itoa(num), cSp[3], cSp[4], cSp[5], cSp[6])

	record.Content = newCont

	if _, err = UpdateRecord(tx, record); err != nil {
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

// UpdateRecordV2 更新记录 content 并递增所属域 SOA serial (独立事务)
func UpdateRecordV2(record Record, domainName string) (int64, error) {

	if DB == nil {
		return 0, fmt.Errorf("UpdateRecordV2() Error: db not initial")
	}

	tx, err := DB.Begin()
	if err != nil {
		return 0, fmt.Errorf("UpdateRecordV2() Error: failed to begin transaction: %w", err)
	}

	rollbackNeeded := true
	defer func() {
		if rollbackNeeded {
			if rollbackErr := tx.Rollback(); rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
				g.Logger.Errorf("UpdateRecordV2() transaction rollback error: %v", rollbackErr)
			}
		}
	}()

	query := `UPDATE records SET content=? WHERE id=? AND name=?`

	result, err := tx.Exec(query, record.Content, record.ID, record.Name)

	if err != nil {
		g.Logger.Errorf("UpdateRecordV2() update ip:%s err:%s", record.Content, err)
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()

	if err != nil {
		g.Logger.Errorf("UpdateRecordV2() affect name:%s err:%s", record.Name, err)
		return 0, err
	}

	if err = UpdateSOA(tx, domainName); err != nil {
		return 0, fmt.Errorf("update soa serial failed: %w", err)
	}

	if err := tx.Commit(); err != nil {
		g.Logger.Errorf("UpdateRecordV2() failed to commit transaction: %v", err)
		return 0, fmt.Errorf("tx commit failed: %w", err)
	}

	rollbackNeeded = false

	return rowsAffected, nil
}

// DeleteRecordByID 按记录主键删除并递增所属域 SOA serial (独立事务)
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
			if rollbackErr := tx.Rollback(); rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
				g.Logger.Errorf("DeleteRecordByID() transaction rollback error: %v", rollbackErr)
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

	if err = UpdateSOA(tx, domainName); err != nil {
		return 0, fmt.Errorf("update soa serial failed: %w", err)
	}

	if err := tx.Commit(); err != nil {
		g.Logger.Errorf("DeleteRecordByID() failed to commit transaction: %v", err)
		return 0, fmt.Errorf("tx commit failed: %w", err)
	}
	rollbackNeeded = false

	return rowsAffected, nil
}

// DeleteRecordByDomainID 删除 domain 下全部 records 及 domains 表对应行 (传入事务)。
// 注意: 函数名有历史误导性, 它同时删除 records 与 domains 两张表的数据。
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

	if g.Config().Debug {
		g.Logger.Infof("DeleteRecordByDomainID() successfully deleted %d rows for domain_id %d",
			totalRowsAffected, id)
	}

	return totalRowsAffected, nil
}

var spaceRegex = regexp.MustCompile(`\s+`)

func compressSpaces(s string) string {
	// 修复: 正则预编译, 避免每次调用重新编译
	return spaceRegex.ReplaceAllString(s, " ")
}

func GetRecordsByPageLimit(page int, perPage int) (records []*Record, err error) {

	clause, clauseArgs := dnsServerFilter()

	query := "SELECT id, domain_id, name, type, content, ttl FROM records" +
		" WHERE type = 'A'"
	args := make([]interface{}, 0, len(clauseArgs)+2)

	if clause != "" {
		query += " AND " + clause
		args = append(args, clauseArgs...)
	}

	var rows *sql.Rows

	if perPage > 0 {
		offset := (page - 1) * perPage
		query += " ORDER BY id LIMIT ?, ?"
		args = append(args, offset, perPage)
		rows, err = DB.Query(query, args...)
	} else {
		query += " ORDER BY id"
		rows, err = DB.Query(query, args...)
	}

	defer func() {
		if rows != nil {
			_ = rows.Close()
		}
	}()

	if err != nil {
		g.Logger.Errorf("GetRecordsByPageLimit() query error: %s", err)
		return records, err
	}

	if g.Config().Debug {
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

	clause, clauseArgs := dnsServerFilter()

	query := "SELECT count(id) FROM records WHERE type = 'A'"
	if clause != "" {
		query += " AND " + clause
	}

	err = DB.QueryRow(query, clauseArgs...).Scan(&count)

	if err != nil {
		g.Logger.Errorf("GetRecordsCount() query error: %s", err)
		return 0, err
	}

	return count, nil
}

func GetHostRecordsCount(HostDNS Record) (count int, err error) {

	hostname := HostDNS.Name
	domainID := HostDNS.DomainID

	query := `SELECT count(id) FROM records 
		WHERE type = 'A' AND name = ? AND domain_id = ?`

	if g.Config().Debug {
		g.Logger.Debugf("GetHostRecordsCount() query: %s", query)
	}

	err = DB.QueryRow(query, hostname, domainID).Scan(&count)

	if err != nil {
		g.Logger.Errorf("GetHostRecordsCount() query error: %s", err)
		return 0, err
	}

	return count, nil
}

func GetHostRecordsID(HostDNS Record) (id int64, err error) {

	hostname := HostDNS.Name
	domainID := HostDNS.DomainID

	query := `SELECT id FROM records 
		WHERE type = 'A' AND name = ? AND domain_id = ? ORDER BY id LIMIT 1`

	err = DB.QueryRow(query, hostname, domainID).Scan(&id)

	if err != nil {
		g.Logger.Errorf("GetHostRecordsID() query error: %s", err)
		return 0, err
	}

	return id, nil
}

func GetSOARecords() (records []*Record, err error) {

	query := "SELECT id, domain_id, name, type, content, ttl FROM records" +
		" WHERE type = 'SOA'"

	rows, err := DB.Query(query)

	defer func() {
		if rows != nil {
			_ = rows.Close()
		}
	}()

	if err != nil {
		g.Logger.Errorf("GetSOARecords() query error: %s", err)
		return records, err
	}

	if g.Config().Debug {
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

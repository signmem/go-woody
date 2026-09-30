package db

import (
	"database/sql"
	"fmt"
	"math"
	"net"
	"regexp"
	"strconv"
	"strings"

	"github.com/signmem/go-woody/g"
)

// InsertDomain 在事务内新建PDNS域名，同时自动生成NS/A/SOA记录
// !!!重要：本函数内部不会执行tx.Rollback；返回err!=nil时，调用方必须执行tx.Rollback()，否则会产生残缺domain脏数据
// !!!调用前建议先调用 GetDomainsByName 判断域名是否已存在
func InsertDomain(tx *sql.Tx, domain Domain) (int64, error) {

	query := `INSERT INTO domains (name, master, type) VALUES (?, ?, ?)`

	result, err := tx.Exec(query, domain.Name, domain.Master, domain.Type)

	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()

	if err != nil {
		g.Logger.Errorf("InsertDomain() %s, error: %s ", query, err)
		return 0, err
	}

	dnsServer := g.Config().DnsServer

	for num, ip := range dnsServer {

		var dnsNSRecord Record

		numStr := strconv.Itoa(num)
		serverName := "dns" + numStr + "." + domain.Name

		dnsNSRecord.TTL = 30
		dnsNSRecord.Name = domain.Name
		dnsNSRecord.Type = "NS"
		dnsNSRecord.Content = serverName
		dnsNSRecord.DomainID = id

		if _, err = InsertRecord(tx, dnsNSRecord); err != nil {
			g.Logger.Errorf("InsertDomain() failed to add NS record "+
				"for %s: %v", serverName, err)
			return 0, err
		}

		var dnsNameRecord Record
		dnsNameRecord.TTL = 30
		dnsNameRecord.DomainID = id
		dnsNameRecord.Name = serverName
		dnsNameRecord.Content = ip
		dnsNameRecord.Type = "A"

		_, err = InsertRecord(tx, dnsNameRecord)

		if err != nil {
			g.Logger.Errorf("InsertDomain() %s A record "+
				"error: %s", serverName, err)
			return 0, err
		}

	}

	var dnsSOARecord Record
	dnsSOARecord.TTL = 30
	dnsSOARecord.DomainID = id
	dnsSOARecord.Name = domain.Name
	dnsSOARecord.Content = "ns.vip.com ns.vip.com 1 200 200 200 30"
	dnsSOARecord.Type = "SOA"

	_, err = InsertRecord(tx, dnsSOARecord)

	if err != nil {
		g.Logger.Errorf("InsertDomain() %s SOA record "+
			"error: %s", domain.Name, err)
		return 0, err
	}

	return id, nil
}

func GetDomainByID(id int64) (*Domain, error) {

	query := `SELECT id, name, master, type FROM domains WHERE id = ?`

	var domain Domain

	err := DB.QueryRow(query, id).Scan(
		&domain.ID,
		&domain.Name,
		&domain.Master,
		&domain.Type,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			g.Logger.Errorf("GetDomainByID() rows error %s", err)
			return nil, fmt.Errorf("domain id %d not found", id)
		}
		g.Logger.Errorf("GetDomainByID() error %s", err)
		return nil, err
	}
	return &domain, nil
}

func GetDomainsByName(name string) (*Domain, error) {

	if name == "" {
		return nil, fmt.Errorf("domain name cannot be empty")
	}

	query := `SELECT id, name, master, type FROM domains WHERE name = ?`

	var domain Domain

	err := DB.QueryRow(query, name).Scan(
		&domain.ID,
		&domain.Name,
		&domain.Master,
		&domain.Type,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			// Info
			g.Logger.Infof("GetDomainsByName(): domain '%s' not found", name)
			return nil, err
		}

		g.Logger.Errorf("GetDomainsByName(): database query failed for domain '%s'. Error: %v", name, err)
		return nil, err
	}

	g.Logger.Debugf("GetDomainsByName(): successfully retrieved domain '%s' (ID: %d)", name, domain.ID)
	return &domain, nil
}

func GetDomainSOAByID(domainID int64) (domain_soa DomainSOA, err error) {
	query := `SELECT id, domain_id, name, type, content, ttl FROM records where type='SOA' and domain_id = ?`

	var record Record

	err = DB.QueryRow(query, domainID).Scan(
		&record.ID,
		&record.DomainID,
		&record.Name,
		&record.Type,
		&record.Content,
		&record.TTL,
	)

	if err != nil {
		g.Logger.Errorf("GetDomainSOAByID() get %d soa record error:%s", domainID, err)
		return domain_soa, err
	}

	content := compressSpaces(record.Content)
	c_sp := strings.Split(content, " ")
	if len(c_sp) < 7 {
		err := fmt.Errorf("soa content format invalid, split len < 7")
		g.Logger.Errorf("GetDomainSOAByID() domainID%d: %v", domainID, err)
		return domain_soa, err
	}

	numStr := c_sp[2]
	num, err := strconv.Atoi(numStr)
	if err != nil {
		g.Logger.Errorf("GetDomainSOAByID() SOA format parse num error: %s", err)
		return domain_soa, err
	}

	if num >= math.MaxInt64-1 {
		// 修复: 原实现此分支 return 的 err 为 nil, 静默返回零值"成功"
		err = fmt.Errorf("GetDomainSOAByID() domain id %d: SOA serial %d exceeds max value", domainID, num)
		g.Logger.Error(err)
		return domain_soa, err
	}

	domain_soa.DomainID = domainID
	domain_soa.DomainName = record.Name
	domain_soa.DomainSOA = int64(num)

	return domain_soa, nil
}

func GetDomainSOAByName(domainName string) (domain_soa DomainSOA, err error) {
	query := `SELECT id, domain_id, name, type, content, ttl FROM records where type='SOA' and name = ?`

	var record Record

	err = DB.QueryRow(query, domainName).Scan(
		&record.ID,
		&record.DomainID,
		&record.Name,
		&record.Type,
		&record.Content,
		&record.TTL,
	)

	if err != nil {
		g.Logger.Errorf("GetDomainSOAByName() get %s soa record error:%s", domainName, err)
		return domain_soa, err
	}

	content := compressSpaces(record.Content)
	c_sp := strings.Split(content, " ")
	if len(c_sp) < 7 {
		err := fmt.Errorf("soa content format invalid, split len < 7")
		g.Logger.Errorf("GetDomainSOAByName() domain %s: %v", domainName, err)
		return domain_soa, err
	}

	numStr := c_sp[2]
	num, err := strconv.Atoi(numStr)
	if err != nil {
		g.Logger.Errorf("GetDomainSOAByName() SOA format parse num error: %s", err)
		return domain_soa, err
	}

	if num >= math.MaxInt64-1 {
		// 修复: 原实现此分支 return 的 err 为 nil, 静默返回零值"成功"
		err = fmt.Errorf("GetDomainSOAByName() domain %s: SOA serial %d exceeds max value", domainName, num)
		g.Logger.Error(err)
		return domain_soa, err
	}

	domain_soa.DomainID = record.DomainID
	domain_soa.DomainName = record.Name
	domain_soa.DomainSOA = int64(num)

	return domain_soa, nil
}

func GetDomainSOA() (domain_soas []DomainSOA, err error) {

	all_db_records, err := GetSOARecords()

	if err != nil {
		return domain_soas, err
	}

	if len(all_db_records) == 0 {
		err := fmt.Errorf("GetDomainSOA() found none records in db")
		g.Logger.Error(err)
		return domain_soas, err
	}

	for _, record := range all_db_records {

		var domain_soa DomainSOA

		if record.Type != "SOA" {
			continue
		}

		domain_soa.DomainID = record.DomainID
		domain_soa.DomainName = record.Name

		content := compressSpaces(record.Content)
		c_sp := strings.Split(content, " ")

		if len(c_sp) < 7 {
			err := fmt.Errorf("GetDomainSOA() domain %s: split len <7", record.Name)
			g.Logger.Error(err)
			return domain_soas, err
		}

		numStr := c_sp[2]
		num, err := strconv.Atoi(numStr)

		if err != nil {
			inErr := fmt.Errorf("GetDomainSOAByID()  %s SOA format parse num error: %s", record.Name, err)
			g.Logger.Error(inErr)
			return domain_soas, inErr
		}

		if num >= 9223372036854775806 {
			inErr := fmt.Errorf("GetDomainSOAByID() %s SOA number match MaxInt Value", record.Name)
			g.Logger.Error(inErr)
			return domain_soas, inErr
		}

		domain_soa.DomainSOA = int64(num)

		domain_soas = append(domain_soas, domain_soa)
	}

	return domain_soas, nil
}

func HostCheck(hostname string) (host string, domains []string, fullDomain string,
	err error) {

	if !IsValidDomain(hostname) {
		return "", nil, "",
			fmt.Errorf("无效的域名格式: %s", hostname)
	}

	// 分割域名部分
	parts := strings.Split(hostname, ".")

	if len(parts) < 3 {
		// 如果是二级域名，没有host部分，返回错误
		return "", nil, "",
			fmt.Errorf("这个可能是域名，不是一个合法的主机名: %s", hostname)
	}

	// 第一个部分是host
	host = parts[0]
	fullDomain = strings.Join(parts[1:], ".")

	// 生成所有层次的域名：从三级域名到二级域名
	domains = generateDomainLevels(parts[1:])
	domainReverse := ListReverse(domains)

	return host, domainReverse, fullDomain, nil
}

func generateDomainLevels(domainParts []string) []string {
	var domains []string

	// 从当前部分开始，逐步减少层级，直到只剩下二级域名
	for i := 0; i <= len(domainParts)-2; i++ {
		// 组合从第i个部分开始的所有部分
		currentDomain := strings.Join(domainParts[i:], ".")
		domains = append(domains, currentDomain)
	}

	return domains
}

var (
	domainRegex        = regexp.MustCompile(`^([a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$`)
	hostnameLabelRegex = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9_-]{0,61}[a-zA-Z0-9])?$`)
)

func IsValidDomain(hostname string) bool {

	// remove domain.  (last .)
	hostname = strings.TrimSuffix(hostname, ".")

	if len(hostname) == 0 || len(hostname) > 253 {
		return false
	}

	// remove hostname:port  usage
	if strings.Contains(hostname, ":") {
		var err error
		hostname, _, err = net.SplitHostPort(hostname)
		if err != nil {
			return false
		}
	}

	labels := strings.Split(hostname, ".")

	// support 162.com 2 level domain
	if len(labels) < 2 {
		return false
	}

	// 修复: 正则预编译 (包级变量), 且简化冗余的 if/return
	return domainRegex.MatchString(hostname)
}

func ListReverse(list []string) []string {
	out := make([]string, len(list))
	copy(out, list)
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

func IsValidHostname(hostname string) bool {
	if len(hostname) == 0 || len(hostname) > 253 {
		return false
	}

	if !strings.Contains(hostname, ".") {
		return false
	}

	parts := strings.Split(hostname, ".")
	if len(parts) < 3 {
		return false
	}

	for _, part := range parts {
		if len(part) == 0 || len(part) > 63 {
			return false
		}

		// 支持含 "_" 的 label (预编译正则)
		if !hostnameLabelRegex.MatchString(part) {
			return false
		}
	}

	if len(parts[len(parts)-1]) < 2 {
		return false
	}

	return true
}

func GetDomainReverseLevels(domain string) []string {
	domain = strings.Trim(domain, ".")
	parts := strings.Split(domain, ".")

	if len(parts) < 2 {
		return []string{domain}
	}

	levels := make([]string, 0, len(parts)-1)

	for i := 2; i <= len(parts); i++ {
		levels = append(levels, strings.Join(parts[len(parts)-i:], "."))
	}

	return levels
}

func GetDomainsByPageLimit(page int, per_page int) (records []*Domain, err error) {

	var rows *sql.Rows

	query := "SELECT id, name, master, type FROM domains Order By id"

	if per_page > 0 {
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
		g.Logger.Errorf("GetDomainsByPageLimit() query error: %s", err)
		return records, err
	}

	if g.Config().Debug == true {
		g.Logger.Infof("GetDomainsByPageLimit() query: %s", query)
	}

	for rows.Next() {

		var record Domain
		err = rows.Scan(
			&record.ID,
			&record.Name,
			&record.Master,
			&record.Type,
		)

		if err != nil {
			g.Logger.Errorf("GetDomainsByPageLimit() scan error: %s", err)
			return records, err
		}

		records = append(records, &record)
	}

	if err = rows.Err(); err != nil {
		g.Logger.Errorf("GetDomainsByPageLimit() rows iteration error: %s", err)
		return records, err
	}

	return records, nil
}

func GetDomainCount() (count int, err error) {

	query := "SELECT count(id) FROM domains"

	err = DB.QueryRow(query).Scan(&count)

	if err != nil {
		g.Logger.Errorf("GetDomainCount() query error: %s", err)
		return 0, err
	}

	return count, nil
}

func GetAllPDNSDomain(domainType string) (records []*Domain, err error) {

	var query string
	var rows *sql.Rows

	if domainType != "" {
		// 修复: 原条件 "type != 'NATIVE' and type = ?" 中前半段冗余
		query = "SELECT id, name, master, type FROM domains WHERE type = ?"
		rows, err = DB.Query(query, domainType)
	} else {
		query = "SELECT id, name, master, type From domains where type != 'NATIVE'"
		rows, err = DB.Query(query)
	}

	if err != nil {
		g.Logger.Errorf("GetAllPDNSDomain() query error: %s", err)
		return records, err
	}

	defer rows.Close()
	for rows.Next() {

		var record Domain
		err = rows.Scan(
			&record.ID,
			&record.Name,
			&record.Master,
			&record.Type,
		)

		if err != nil {
			g.Logger.Errorf("GetAllPDNSDomain() scan error: %s", err)
			return records, err
		}

		records = append(records, &record)
	}

	if err = rows.Err(); err != nil {
		g.Logger.Errorf("GetAllPDNSDomain() rows iteration error: %s", err)
		return records, err
	}

	return records, nil

}

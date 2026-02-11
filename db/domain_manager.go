package db

import (
	"database/sql"
	"fmt"
	"github.com/signmem/go-woody/g"
	"regexp"
	"strconv"
	"strings"
)

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
        serverName := "dns" + numStr + "." +  domain.Name

        dnsNSRecord.TTL      = 30
        dnsNSRecord.Name    = domain.Name
        dnsNSRecord.Type    = "NS"
        dnsNSRecord.Content = serverName
        dnsNSRecord.DomainID = id

		if _, err = InsertRecord(tx, dnsNSRecord); err != nil {
			g.Logger.Errorf("InsertDomain() failed to add NS record " +
				"for %s: %v", serverName, err)
			return 0, err
		}

		var dnsNameRecord Record
		dnsNameRecord.TTL       = 30
		dnsNameRecord.DomainID  = id
		dnsNameRecord.Name      = serverName
		dnsNameRecord.Content   = ip
		dnsNameRecord.Type        = "A"

		_, err = InsertRecord(tx, dnsNameRecord)

		if err != nil {
			g.Logger.Errorf("InsertDomain() %s A record " +
				"error: %s", serverName, err)
			return 0, err
		}

    }

	var dnsSOARecord Record
	dnsSOARecord.TTL       = 30
	dnsSOARecord.DomainID  = id
	dnsSOARecord.Name      = domain.Name
	dnsSOARecord.Content   = "ns.vip.com ns.vip.com 1 200 200 200 30"
	dnsSOARecord.Type      = "SOA"

	_, err = InsertRecord(tx, dnsSOARecord)

	if err != nil {
		g.Logger.Errorf("InsertDomain() %s SOA record " +
			"error: %s", domain.Name, err)
		return 0, err
	}

	return id, nil
}



func GetDomainByID(id int64) (*Domain, error) {

    query := `SELECT id, name, master, type FROM domains WHERE id = ?`
    
    var domain Domain

    err := DB.QueryRow(query, id).Scan (
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
			return nil, fmt.Errorf("domain '%s' not found", name)
		}

		g.Logger.Errorf("GetDomainsByName(): database query failed for domain '%s'. Error: %v", name, err)
		return nil, fmt.Errorf("database query failed for domain '%s': %w", name, err)
	}

    /*
    if err != nil {
        if err == sql.ErrNoRows {
			g.Logger.Errorf("GetDomainsByName() query %s error %s", name, err)
            return nil, nil
        }
		g.Logger.Errorf("GetDomainsByName() query error %s", err)
        return nil, err
    }
    */

	g.Logger.Debugf("GetDomainsByName(): successfully retrieved domain '%s' (ID: %d)", name, domain.ID)
    return &domain, nil
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
	fullDomain =  strings.Join(parts[1:], ".")

	// 生成所有层次的域名：从三级域名到二级域名
	domains = generateDomainLevels(parts[1:])
	domainReverse := ListReverse(domains)

	return host, domainReverse, fullDomain,nil
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


func IsValidDomain(hostname string) bool {
	if len(hostname) == 0 || len(hostname) > 253 {
		return false
	}
	domainRegex := `^([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])\.([a-zA-Z]{2,}|xn--[a-zA-Z0-9]+)$`
	matched, _ := regexp.MatchString(domainRegex, hostname)
	if matched {
		return true
	}

	multiLevelRegex := `^([a-zA-Z0-9][-a-zA-Z0-9]{0,61}[a-zA-Z0-9]\.)+([a-zA-Z]{2,})$`
	matched, _ = regexp.MatchString(multiLevelRegex, hostname)

	return matched
}

func ListReverse(list []string) ([]string) {

	for i, j := 0, len(list)-1; i < j; i, j = i+1, j-1 {
		list[i], list[j] = list[j], list[i]
	}
	return list
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

		matched, _ := regexp.MatchString(`^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?$`, part)
		if !matched {
			return false
		}
	}

	if len(parts[len(parts)-1]) < 2 {
		return false
	}

	return true
}

package api

import (
	"fmt"
	"github.com/signmem/go-woody/db"
	"net/http"
	"path"
	"strconv"
	"strings"
	"github.com/signmem/go-woody/g"
)

func domainGet(r *http.Request) (v interface{}, err error) {


	cleanPath := path.Clean(r.URL.Path)
	pathSegments := strings.Split(cleanPath, "/")
	segments := make([]string, 0)

	for _, seg := range pathSegments {
		if seg != "" {
			segments = append(segments, seg)
		}
	}

	if len(segments) == 4 && segments[0] == "api" && segments[1] == "v2" && segments[2] == "domains" {
		id := segments[3]
		idInt, err := strconv.Atoi(id)

		if err != nil {
			// 不是数字，按域名名称查询
			return dnsGetSingleDomainName(id)
		}
		// dns 没有纯数字， 因此这里可以安全使用
		return dnsGetSingleDomainID(int64(idInt))
	}


	if len(segments) == 3 && segments[0] == "api" && segments[1] == "v2" && segments[2] == "domains" {

		queryParams := r.URL.Query()
		page, perPage, err := validatePaginationParams(queryParams)

		if err != nil {
			g.Logger.Error(err)
			return nil, err
		}
		return dnsGetMultiDomain( page, perPage)
	}

	msg := fmt.Errorf("domainGet() Error: params error.")
	g.Logger.Error(msg)

	return nil, msg
}

func dnsGetSingleDomainID(domain_id int64) ( domainInfo *db.Domain, err error) {
	return  db.GetDomainByID(domain_id)
}

func dnsGetSingleDomainName(domain_name string) (domainInfo *db.Domain, err error) {
	return  db.GetDomainsByName(domain_name)
}


func dnsGetMultiDomain(m_page int, m_per_page int) (dnsDBRecord DomainRecord, err error) {

	if g.Config().Debug == true {
		g.Logger.Infof("domainGet() page is %d, per_page is %d", m_page, m_per_page)
	}

	records, err := db.GetDomainsByPageLimit(m_page, m_per_page)

	if err != nil {
		g.Logger.Errorf("domainGet() GetRecordsByPageLimit() error: %s", err)
		return dnsDBRecord, fmt.Errorf("Error: failed to get records: %w", err)
	}

	count, err := db.GetDomainCount()

	if err != nil {
		g.Logger.Errorf("domainGet() GetDomainCount() error: %s", err)
		return dnsDBRecord, fmt.Errorf("Error: get domain count failed: %w", err)
	}

	for _, record := range records {
		var DomainInfo db.Domain
		DomainInfo.ID      =  record.ID
		DomainInfo.Name    =  record.Name
		DomainInfo.Master  =  record.Master
		DomainInfo.Type    =  record.Type

		dnsDBRecord.Domains = append(dnsDBRecord.Domains, DomainInfo)
	}

	dnsDBRecord.Page = m_page
	dnsDBRecord.PerPage = m_per_page
	dnsDBRecord.Total = count

	return dnsDBRecord, nil

}

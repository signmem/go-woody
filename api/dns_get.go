package api

import (
	"fmt"
	"github.com/signmem/go-woody/db"
	"github.com/signmem/go-woody/g"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
)

func dnsGet(r *http.Request) (v interface{}, err error) {

	cleanPath := path.Clean(r.URL.Path)

	pathSegments := strings.Split(cleanPath, "/")

	segments := make([]string, 0)
	for _, seg := range pathSegments {
		if seg != "" {
			segments = append(segments, seg)
		}
	}

	if len(segments) == 3 && segments[0] == "api" && segments[1] == "hosts" {
		id := segments[2]
		idInt, err := strconv.Atoi(id)

		if err != nil {
			msg := fmt.Errorf("Error: %s not digital", id)
			g.Logger.Error(msg)
			return nil, msg
		}

		return dnsGetSignleHost(idInt)
	}

	if len(segments) == 2 && segments[0] == "api" && segments[1] == "hosts" {

		queryParams := r.URL.Query()
		page, perPage, err := validatePaginationParams(queryParams)

		if err != nil {
			g.Logger.Error(err)
			return nil, err
		}
		return dnsGetMultiHost( page, perPage)
	}

	msg := fmt.Errorf("Error: params error.")
	g.Logger.Error(msg)

	return nil, msg
}


func validatePaginationParams(params url.Values) (int, int, error) {
	pageStr := params.Get("page")
	page := 1
	if pageStr != "" {
		pageNum, err := strconv.Atoi(pageStr)
		if err != nil || pageNum < 1 {
			return 0, 0, strconv.ErrSyntax
		}
		page = pageNum
	}

	perPageStr := params.Get("per_page")
	perPage := 100
	if perPageStr != "" {
		perPageNum, err := strconv.Atoi(perPageStr)

		if err != nil || perPageNum < 1 {
			return page, 0, strconv.ErrSyntax
		}

		if perPageNum > 1000 {
			perPageNum = 1000
		}

		perPage = perPageNum
	}

	for key := range params {
		if key != "page" && key != "per_page" {
			return 0, 0, fmt.Errorf("not supoort %s", key)
		}
	}

	return page, perPage, nil
}


func dnsGetMultiHost(m_page int, m_per_page int) (dnsDBRecord DNSRecord, err error) {

	if g.Config().Debug == true {
		g.Logger.Infof("dnsGet() page is %d, per_page is %d", m_page, m_per_page)
	}

	records, err := db.GetRecordsByPageLimit(m_page, m_per_page)

	if err != nil {
		g.Logger.Errorf("dnsGet() GetRecordsByPageLimit() error: %s", err)
		return dnsDBRecord, fmt.Errorf("Error: failed to get records: %w", err)
	}

	if len(records) == 0 {
		return dnsDBRecord, fmt.Errorf("Error: not records found.")
	}

	count, err := db.GetRecordsCount()

	if err != nil {
		g.Logger.Errorf("dnsGet() GetRecordsCount() error: %s", err)
	}


	for _, record := range records {
		var hostInfo DNSHost
		hostInfo.ID        =  record.DomainID
		hostInfo.IP        =  record.Content
		hostInfo.Hostname  =  record.Name

		dnsDBRecord.Hosts = append(dnsDBRecord.Hosts, hostInfo)
	}

	dnsDBRecord.Page = m_page
	dnsDBRecord.PerPage = m_per_page
	dnsDBRecord.Total = count

	return dnsDBRecord, nil

}

func dnsGetSignleHost(domain_id int) (dnshost DNSHost, err error) {

	dnsARecord, err := db.GetARecordsByDomainID(domain_id)

	if err != nil {
		g.Logger.Errorf("domain id: %d, error: %s", domain_id, err)
		return dnshost, fmt.Errorf("Host id %d not found.", domain_id )
	}

	dnshost.ID       = dnsARecord.DomainID
	dnshost.Hostname = dnsARecord.Name
	dnshost.IP       = dnsARecord.Content

	return dnshost, nil
}
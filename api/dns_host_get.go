package api

import (
	"errors"
	"fmt"
	"github.com/signmem/go-woody/db"
	"github.com/signmem/go-woody/g"
	"net/http"
	"path"
	"strconv"
	"strings"
)

func hostGet(r *http.Request) (v interface{}, err error) {

	cleanPath := path.Clean(r.URL.Path)
	pathSegments := strings.Split(cleanPath, "/")
	segments := make([]string, 0)

	for _, seg := range pathSegments {
		if seg != "" {
			segments = append(segments, seg)
		}
	}

	if len(segments) == 5 && segments[0] == "api" && segments[1] == "v2" &&
		segments[2] == "hosts" && segments[3] == "domain_id" {
		id := segments[4]
		idInt, err := strconv.Atoi(id)

		if err != nil {
			msg := fmt.Errorf("domain_id must be digital")
			return nil, msg
		}

		if idInt <= 0 {
			return nil, fmt.Errorf("domain_id must greater than zero")
		}

		dnshosts, err := dnsGetSingleHostByDomainIDv2(idInt)

		if err != nil {
			return nil, err
		}

		if len(dnshosts) == 0 {
			msg := fmt.Errorf("domain_id %d now row found in db", idInt)
			return nil, msg
		}

		return dnshosts, nil
	}

	if len(segments) == 5 && segments[0] == "api" && segments[1] == "v2" &&
		segments[2] == "hosts" && segments[3] == "host_id" {
		id := segments[4]
		idInt, err := strconv.Atoi(id)

		if err != nil {
			msg := fmt.Errorf("host_id must be digital")
			return nil, msg
		}

		if idInt <= 0 {
			return nil, errors.New("host_id must greater than zero")
		}

		return dnsGetSingleHostByIDv2(idInt)
	}

	//  /api/v2/hosts/host_name/{hostname}
	if len(segments) == 5 && segments[0] == "api" && segments[1] == "v2" &&
		segments[2] == "hosts" && segments[3] == "host_name" {
		hostname := segments[4]

		if db.IsValidHostname(hostname) == false {
			msg := fmt.Errorf("%s not valid hostname", hostname)
			return nil, msg
		}

		records, err := dnsGetHostByHostNamev2(hostname)

		if err != nil {
			return nil, err
		}

		if len(records) == 0 {
			msg := fmt.Errorf("%s not found in db.", hostname)
			return nil, msg
		}

		return records, nil
	}
	//  api/v2/hosts/{host_ip}
	if len(segments) == 5 && segments[0] == "api" && segments[1] == "v2" &&
		segments[2] == "hosts" && segments[3] == "host_ip" {
		ipaddr := segments[4]

		records, err := dnsGetSingleHostByIPv2(ipaddr)

		if err != nil {
			return nil, err
		}

		if len(records) == 0 {
			msg := fmt.Errorf("%s not found in db.", ipaddr)
			return nil, msg
		}

		return records, nil
	}

	if len(segments) == 3 && segments[0] == "api" && segments[1] == "v2" && segments[2] == "hosts" {

		queryParams := r.URL.Query()
		page, perPage, err := validatePaginationParams(queryParams)

		if err != nil {
			g.Logger.Error(err)
			return nil, err
		}

		dnsInfo, err := dnsGetMultiHostv2(page, perPage)
		if err != nil {
			return nil, err
		}

		if len(dnsInfo.Hosts) == 0 {
			msg := errors.New("none record found in db.")
			return nil, msg
		}

		return dnsInfo, nil
	}

	msg := fmt.Errorf("hostGet() Error: params error.")
	g.Logger.Error(msg)

	return nil, msg
}

func dnsGetHostByHostNamev2(hostname string) (dnshost []DNSHostv2, err error) {

	dnsARecord, err := db.GetARecordsByHostNamev2(hostname)

	if err != nil {
		msg := fmt.Errorf("dnsGetHostByDomainNamev2() error hostname %s error: %s", hostname, err)
		g.Logger.Error(msg)

		return nil, msg
	}

	for _, aRecord := range dnsARecord {

		var host DNSHostv2
		host.ID = aRecord.ID
		host.DomainID = aRecord.DomainID
		host.Hostname = aRecord.Name
		host.IP = aRecord.Content

		dnshost = append(dnshost, host)
	}

	return dnshost, nil
}

func dnsGetMultiHostv2(m_page int, m_per_page int) (dnsDBRecord DNSRecordv2, err error) {

	if g.Config().Debug == true {
		g.Logger.Infof("page is %d, per_page is %d", m_page, m_per_page)
	}

	records, err := db.GetRecordsByPageLimit(m_page, m_per_page)

	if err != nil {
		g.Logger.Errorf("GetRecordsByPageLimit() error: %s", err)
		return dnsDBRecord, fmt.Errorf("Error: failed to get records: %w", err)
	}

	count, err := db.GetRecordsCount()

	if err != nil {
		g.Logger.Errorf("GetRecordsCount() error: %s", err)
		return dnsDBRecord, err
	}

	for _, record := range records {
		var hostInfo DNSHostv2
		hostInfo.ID = record.ID
		hostInfo.IP = record.Content
		hostInfo.Hostname = record.Name
		hostInfo.DomainID = record.DomainID

		dnsDBRecord.Hosts = append(dnsDBRecord.Hosts, hostInfo)
	}

	dnsDBRecord.Page = m_page
	dnsDBRecord.PerPage = m_per_page
	dnsDBRecord.Total = count

	return dnsDBRecord, nil

}

func dnsGetSingleHostByDomainIDv2(domainID int) (dnshosts []DNSHostv2, err error) {

	domainInfo, err := db.GetDomainByID(int64(domainID))

	if err != nil {
		msg := fmt.Errorf("domain %d not in domsins table", domainID)
		g.Logger.Debugf("GetARecordsByDomainIDv2() errror: %s", msg)
		return nil, msg
	}

	domainName := domainInfo.Name

	dnsARecord, err := db.GetARecordsByDomainNamev2(domainID, domainName)

	if err != nil {
		g.Logger.Errorf("domain id: %d, error: %s", domainID, err)
		return nil, fmt.Errorf("Host id %d not found.", domainID)
	}

	for _, aRecord := range dnsARecord {
		var dnshost DNSHostv2
		dnshost.ID = aRecord.ID
		dnshost.Hostname = aRecord.Name
		dnshost.IP = aRecord.Content
		dnshost.DomainID = aRecord.DomainID
		dnshosts = append(dnshosts, dnshost)
	}

	return dnshosts, nil
}

func dnsGetSingleHostByIDv2(hostID int) (dnshosts DNSHostv2, err error) {

	records, err := db.GetARecordsByHostIDv2(hostID)

	if err != nil {
		return dnshosts, err
	}

	dnshosts.ID = records.ID
	dnshosts.DomainID = records.DomainID
	dnshosts.Hostname = records.Name
	dnshosts.IP = records.Content

	return dnshosts, nil
}

func dnsGetSingleHostByIPv2(ipaddr string) (dnshosts []DNSHostv2, err error) {

	records, err := db.GetARecordsByHostIPv2(ipaddr)

	if err != nil {
		return dnshosts, err
	}

	for _, host := range records {
		var dnshost DNSHostv2
		dnshost.ID = host.ID
		dnshost.DomainID = host.DomainID
		dnshost.Hostname = host.Name
		dnshost.IP = host.Content
		dnshosts = append(dnshosts, dnshost)
	}

	return dnshosts, nil
}

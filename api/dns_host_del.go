package api

import (
	"errors"
	"fmt"
	"net/http"
	"path"
	"strconv"
	"strings"

	"github.com/signmem/go-woody/db"
	"github.com/signmem/go-woody/g"
)

func hostDelete(r *http.Request) (v interface{}, err error) {

	cleanPath := path.Clean(r.URL.Path)
	pathSegments := strings.Split(cleanPath, "/")
	segments := make([]string, 0)

	for _, seg := range pathSegments {
		if seg != "" {
			segments = append(segments, seg)
		}
	}

	// do delete from /api/v2/hosts/host_id/ID
	if len(segments) == 5 && segments[0] == "api" && segments[1] == "v2" &&
		segments[2] == "hosts" && segments[3] == "host_id" {

		id := segments[4]
		idInt, err := strconv.Atoi(id)
		if err != nil {
			msg := errors.New("host_id is not digital")
			return nil, msg
		}

		dnshosts, err := dnsGetSingleHostByIDv2(idInt)

		if err != nil {
			return nil, err
		}

		_, err = dnsDeleteSingleHostByDomainIDv2(dnshosts)

		if err != nil {
			return nil, err
		}

		return dnshosts, nil
	}

	// do delete from /api/v2/hosts/host_name/hostName
	if len(segments) == 5 && segments[0] == "api" && segments[1] == "v2" &&
		segments[2] == "hosts" && segments[3] == "host_name" {

		hostname := strings.TrimSpace(segments[4])
		if !db.IsValidHostname(hostname) {
			msg := fmt.Errorf("%s not valid hostname", hostname)
			return nil, msg
		}

		dnsARecords, err := db.GetARecordsByHostNamev2(hostname)

		if err != nil {
			return nil, err
		}

		if len(dnsARecords) != 1 {
			msg := fmt.Errorf("%s total count is %d not unique data", hostname, len(dnsARecords))
			return nil, msg
		}

		var host DNSHostv2
		host.ID = dnsARecords[0].ID
		host.Hostname = dnsARecords[0].Name
		host.DomainID = dnsARecords[0].DomainID
		host.IP = dnsARecords[0].Content

		_, err = dnsDeleteSingleHostByDomainIDv2(host)

		if err != nil {
			return nil, err
		}

		return host, nil
	}

	msg := fmt.Errorf("not valid parameters.")

	return nil, msg
}

// dnsDeleteSingleHostByDomainIDv2 删除单条 host 记录并递增所属 domain 的 SOA serial。
// 修复: 原实现用 GetParentDomain(hostname) 猜测域名, 当主机挂在更高层域名下时
// (如 a.b.163.com 挂在 163.com) 会猜错, 导致 UpdateSOA 找不到 SOA 记录而删除失败;
// 现在直接按记录的 domain_id 反查 domains 表取真实域名。
func dnsDeleteSingleHostByDomainIDv2(host DNSHostv2) (int64, error) {
	domainInfo, err := db.GetDomainByID(host.DomainID)
	if err != nil {
		return 0, fmt.Errorf("dnsDeleteSingleHostByDomainIDv2() lookup domain (id=%d) for host %s failed: %w",
			host.DomainID, host.Hostname, err)
	}

	affected, err := db.DeleteRecordByID(host.ID, domainInfo.Name)
	if err != nil {
		g.Logger.Errorf("[v2-host-delete] delete host %s (id=%d) failed: %v", host.Hostname, host.ID, err)
		return affected, err
	}

	g.Logger.Infof("[v2-host-delete] delete host %s (id=%d, ip=%s, domain=%s) success",
		host.Hostname, host.ID, host.IP, domainInfo.Name)

	return affected, nil
}

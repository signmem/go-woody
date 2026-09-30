package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/signmem/go-woody/db"
	"github.com/signmem/go-woody/g"
)

func hostModify(r *http.Request) (v interface{}, err error) {

	defer func() {
		_ = r.Body.Close()
	}()

	urlPath := strings.TrimPrefix(r.URL.Path, "/api/v2/hosts/")
	pathParts := strings.Split(urlPath, "/")

	if len(pathParts) != 1 || pathParts[0] == "" {
		msg := fmt.Errorf("hostModify() Error: path invalid")
		g.Logger.Error(msg)
		return nil, msg
	}

	hostID, err := strconv.Atoi(pathParts[0])

	if err != nil {
		msg := fmt.Errorf("hostModify() Error: path %s not valid number", pathParts[0])
		g.Logger.Error(msg)
		return nil, msg
	}

	if r.ContentLength == 0 {
		msg := fmt.Errorf("hostModify() Error: body is blank")
		g.Logger.Error(msg)
		return nil, msg
	}

	if !isContentTypeJson(r) {
		msg := fmt.Errorf("hostModify() Error: body not json format")
		g.Logger.Error(msg)
		return nil, msg
	}

	body, err := io.ReadAll(r.Body)

	if err != nil {
		msg := fmt.Errorf("hostModify() Error: body read error: %v", err)
		g.Logger.Error(msg)
		return nil, msg
	}

	var hostDict HostParams
	err = json.Unmarshal(body, &hostDict)

	if err != nil {
		msg := fmt.Errorf("hostModify() Error: body json unmarshal error: %v", err)
		g.Logger.Error(msg)
		return nil, msg
	}

	TrimAllStrings(&hostDict)

	if !isIPv4(hostDict.IP) {
		msg := fmt.Errorf("hostModify() Error: %s not valid ipaddress", hostDict.IP)
		g.Logger.Error(msg)
		return nil, msg
	}

	hostInfo, err := dnsGetSingleHostByIDv2(hostID)

	if err != nil {
		msg := fmt.Errorf("hostModify() Error: %s", err)
		g.Logger.Error(msg)
		return nil, msg
	}

	if hostInfo.Hostname != hostDict.Hostname {
		msg := fmt.Errorf("hostModify() Error: hostname %s not match %s in db",
			hostDict.Hostname, hostInfo.Hostname)
		g.Logger.Error(msg)
		return nil, msg
	}

	if hostInfo.IP == hostDict.IP {
		msg := fmt.Errorf("hostModify() Error: IP %s has not change", hostDict.IP)
		g.Logger.Error(msg)
		return nil, msg
	}

	var updateInfo db.Record
	updateInfo.Content = hostDict.IP
	updateInfo.Name = hostDict.Hostname
	updateInfo.DomainID = hostInfo.DomainID
	updateInfo.ID = hostInfo.ID

	// 修复: 原实现用 GetParentDomain(hostname) 猜测域名, 主机挂在更高层域名下时
	// 会猜错导致 UpdateSOA 失败; 现在按 domain_id 反查真实域名
	domainInfo, err := db.GetDomainByID(hostInfo.DomainID)
	if err != nil {
		msg := fmt.Errorf("hostModify() Error: lookup domain (id=%d) failed: %w",
			hostInfo.DomainID, err)
		g.Logger.Error(msg)
		return nil, msg
	}

	if err = updateRecordV2(updateInfo, domainInfo.Name); err != nil {
		g.Logger.Errorf("hostModify() Error: updateRecordV2() error %s", err)
		return nil, err
	}

	// 修复: 原实现直接返回内部 db.Record (无 json tag, 字段名与其他接口不一致),
	// 现在统一返回 DNSHostv2
	result := DNSHostv2{
		Hostname: updateInfo.Name,
		ID:       updateInfo.ID,
		DomainID: updateInfo.DomainID,
		IP:       updateInfo.Content,
	}

	return result, nil
}

func updateRecordV2(host db.Record, domainName string) (err error) {
	_, err = db.UpdateRecordV2(host, domainName)
	return err
}

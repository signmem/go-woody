package api

import (
	"encoding/json"
	"fmt"
	"github.com/signmem/go-woody/db"
	"github.com/signmem/go-woody/g"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"
)


func hostModify(r *http.Request) (v interface{}, err error) {

	defer func() {
		_ = r.Body.Close()
	}()

	path := strings.TrimPrefix(r.URL.Path, "/api/v2/hosts/")
	pathParts := strings.Split(path, "/")

	if len(pathParts) != 1 || pathParts[0] == "" {
		msg := fmt.Errorf("Error: path invalid")
		g.Logger.Error(msg)
		return nil, msg
	}

	host_id, err :=  strconv.Atoi(pathParts[0])

	if err != nil {
		msg := fmt.Errorf("Error: path %s not valid number.", pathParts[0])
		g.Logger.Error(msg)
		return nil, msg
	}

	if r.ContentLength == 0 {
		msg := fmt.Errorf("Error: body is blank")
		g.Logger.Error(msg)
		return nil, msg
	}

	contentType := r.Header.Get("Content-Type")
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil || mediaType != "application/json" {
		msg := fmt.Errorf("body not json format")
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
		msg := fmt.Errorf("hostModify() Error: body json unmarshal error")
		g.Logger.Error(msg)
		return nil, msg
	}

	if isIPv4(hostDict.IP) == false {
		msg := fmt.Errorf("hostModify() Error: %s not valid ipaddress", hostDict.IP)
		g.Logger.Error(msg)
		return nil, msg
	}

	hostInfo, err := dnsGetSingleHostByIDv2(host_id)

	if err != nil {
		msg := fmt.Errorf("hostModify()  Error: %s", err)
		g.Logger.Error(msg)
		return nil, msg
	}

	if  hostInfo.Hostname != hostDict.Hostname {
		msg := fmt.Errorf("Error: hostname %s not match %s in db", hostDict.Hostname, hostInfo.Hostname)
		g.Logger.Error(msg)
		return nil, msg
	}

	if hostInfo.IP == hostDict.IP {
		msg := fmt.Errorf("Error: IP %s has not change.", hostDict.IP)
		g.Logger.Error(msg)
		return nil, msg
	}

	var updateInfo  db.Record
	updateInfo.Content   = hostDict.IP
	updateInfo.Name      = hostDict.Hostname
	updateInfo.DomainID  = hostInfo.DomainID
	updateInfo.ID        = hostInfo.ID

	domainName := GetParentDomain(updateInfo.Name)
	err = updateRecordV2(updateInfo, domainName)

	if err != nil {
		msg := fmt.Sprintf("hostModify() Error: UpdateRecordV2() error %s", err)
		g.Logger.Error(msg)
		return nil, err
	}

	return updateInfo, nil

}

func updateRecordV2(host db.Record, domainName string) (err error) {
	_, err = db.UpdateRecordV2(host, domainName)
	return err
}

package api

import (
	"fmt"
	"strings"

	"github.com/signmem/go-woody/db"
)

type DomainHostCreate struct {
	Domain string `json:"domain"`
	// 修复: 原 tag 拼写为 "hsots"
	Hosts []HostParams `json:"hosts"`
}

type HostCreate struct {
	Hosts []HostParams `json:"hosts"`
}

type SyncSlaveDomain struct {
	Sync bool `json:"sync"`
}

type DomainCreate struct {
	// 修复: 原 tag 为大写 "Domains", 序列化输出与 README 不一致
	Domains []string `json:"domains"`
	// 修复: 原 tag 为 "master, omitempty" (逗号后带空格), omitempty 静默失效
	Master string `json:"master,omitempty"`
}

type HostParams struct {
	Hostname string `json:"hostname"`
	IP       string `json:"ip"`
}

type DnsAddStatus struct {
	Success int `json:"success"`
	Failure int `json:"failure"`
}

type ReturnMsg struct {
	Msg string `json:"message"`
}

func (this *DnsAddStatus) String() string {
	return fmt.Sprintf("Save %d hosts. Error %d hosts.", this.Success, this.Failure)
}

func (this *HostCreate) String() string {
	var builder strings.Builder
	for _, host := range this.Hosts {
		builder.WriteString(fmt.Sprintf("Host: %s, ipaddr: %s ",
			host.Hostname, host.IP))
	}
	return builder.String()
}

func (this *DomainCreate) String() string {
	var builder strings.Builder
	for _, domain := range this.Domains {
		builder.WriteString(fmt.Sprintf("domain: %s", domain))
	}
	return builder.String()
}

type DNSRecord struct {
	Hosts   []DNSHost `json:"hosts"`
	Page    int       `json:"page"`
	PerPage int       `json:"per_page"`
	Total   int       `json:"total"`
}

type DNSRecordv2 struct {
	Hosts   []DNSHostv2 `json:"hosts"`
	Page    int         `json:"page"`
	PerPage int         `json:"per_page"`
	Total   int         `json:"total"`
}

type DomainRecord struct {
	Domains []db.Domain `json:"domains"`
	Page    int         `json:"page"`
	PerPage int         `json:"per_page"`
	Total   int         `json:"total"`
}

type DNSHost struct {
	Hostname string `json:"hostname"`
	ID       int64  `json:"id"`
	IP       string `json:"ip"`
}

type DNSHostv2 struct {
	Hostname string `json:"hostname"`
	ID       int64  `json:"id"`
	// 注意: 保持历史字段名 domain_ID 不变, 避免破坏既有客户端
	DomainID int64  `json:"domain_ID"`
	IP       string `json:"ip"`
}

type DomainInfo struct {
	DomainName string `json:"domain_name,omitempty"`
	DomainID   int64  `json:"domain_id,omitempty"`
}

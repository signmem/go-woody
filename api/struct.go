package api

import (
	"fmt"
	"github.com/signmem/go-woody/db"
	"strings"
)

type DomainHostCreate struct {
	Domain 		string				`json:"domain"`
	Hosts 		[]HostParams		`json:"hsots"`
}

type HostCreate struct {
	Hosts		[]HostParams		`json:"hosts"`
}

type SyncSlaveDomain struct {
	Sync 		bool 				`json:"sync"`
}

type DomainCreate struct {
	Domains 	[]string			`json:"Domains"`
	Master 		string 				`json:"master, omitempty"`
}

type HostParams struct {
	Hostname	string		`json:"hostname"`
	IP			string		`json:"ip"`
}

type DnsAddStatus struct {
	Success 		int 		`json:"success"`
	Failure			int 		`json:"failure"`
}

type ReturnMsg struct {
	Msg 		string		`json:"message"`
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
	Hosts		[]DNSHost		`json:"hosts"`
	Page		int				`json:"page"`
	PerPage		int				`json:"per_page"`
	Total		int				`json:"total"`
}

type DNSRecordv2 struct {
	Hosts		[]DNSHostv2		`json:"hosts"`
	Page		int				`json:"page"`
	PerPage		int				`json:"per_page"`
	Total		int				`json:"total"`
}

type DomainRecord struct {
	Domains 	[]db.Domain		`json:"domains"`
	Page		int				`json:"page"`
	PerPage		int				`json:"per_page"`
	Total		int				`json:"total"`
}

type DNSHost struct {
	Hostname		string		`json:"hostname"`
	ID				int64		`json:"id"`
	IP				string		`json:"ip"`
}

type DNSHostv2 struct {
	Hostname		string		`json:"hostname"`
	ID				int64		`json:"id"`
	DomainID		int64		`json:"domain_ID"`
	IP				string		`json:"ip"`
}


type DomainInfo struct {
	DomainName 		string		`json:"domain_name,omitempty"`
	DomainID 		int64		`json:"domain_id,omitempty"`
}


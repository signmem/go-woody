package api

import (
	"fmt"
	"strings"
)

type HostCreate struct {
	Hosts		[]HostParams		`json:"hosts"`
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


type DNSRecord struct {
	Hosts		[]DNSHost		`json:"hosts"`
	Page		int				`json:"page"`
	PerPage		int				`json:"per_page"`
	Total		int				`json:"total"`
}

type DNSHost struct {
	Hostname		string		`json:"hostname"`
	ID				int64		`json:"id"`
	IP				string		`json:"ip"`
}
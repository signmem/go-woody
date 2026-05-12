package api

import (
	"database/sql"
	"fmt"
	"github.com/pkg/errors"
	"github.com/signmem/go-woody/db"
	"github.com/signmem/go-woody/g"
	"net/netip"
)



func isIPv4(ipaddr string) bool {
    _, err := netip.ParseAddr(ipaddr)
    if err != nil {
            return false
    }
    return true
}


func dnsDomainAdd(tx *sql.Tx, domainStr string) (id int64, err error) {


	// 域名添加

	domainInfo, err := db.GetDomainsByName(domainStr)

	// ======================================
	// [终极修复]不管错误是什么，只要没查到数据，就继续插入
	// ======================================

	if domainInfo != nil && domainInfo.ID != 0  {
		msg := fmt.Sprintf("domain %s records exists," +
			" id: %d", domainStr, domainInfo.ID)
		g.Logger.Info( msg )
		return 0, errors.New(msg)
	}
	// [正常] 域名不存在 -> 执行插入
	g.Logger.Infof("dnsDomainAdd() domain %s not found, prepare insert", domainStr)
	var domainDB db.Domain
	domainDB.Type = "NATIVE"
	domainDB.Name  = domainStr
	domainDB.Master = ""

	if g.Config().Debug == true {
		g.Logger.Debugf("dnsDomainAdd() add domain %v", domainDB)
	}

	return db.InsertDomain(tx, domainDB)

}

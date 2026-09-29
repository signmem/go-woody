package api

import (
	"database/sql"
	"errors"
	"fmt"
	"net/netip"

	"github.com/go-sql-driver/mysql"
	"github.com/signmem/go-woody/db"
	"github.com/signmem/go-woody/g"
)

// errRecordExists 记录已存在 (唯一索引冲突 1062 或查重命中),
// 调用方可用 errors.Is 判定并按"重复记录"计数而非未知错误
var errRecordExists = errors.New("record already exists")

// isMySQLDuplicate 判断是否为 MySQL 唯一键冲突错误 (errno 1062)。
// 配合建议的唯一索引:
//
//	ALTER TABLE records ADD UNIQUE KEY uk_records_name_type_content (name, type, content);
func isMySQLDuplicate(err error) bool {
	var me *mysql.MySQLError
	return errors.As(err, &me) && me.Number == 1062
}

// isIPv4 严格校验 IPv4 地址。
// 修复: 原实现 netip.ParseAddr 对 IPv6 也返回成功, IPv6 地址会被写成
// 非法的 A 记录; 现在要求 addr.Is4()
func isIPv4(ipaddr string) bool {
	addr, err := netip.ParseAddr(ipaddr)
	if err != nil {
		return false
	}
	return addr.Is4()
}

// dnsDomainAdd 在事务内为劫持主机名创建独立 domain (NATIVE)。
// 修复: 原实现("[终极修复]不管错误是什么")把 sql.ErrNoRows 与真实 DB 故障
// (连接断开/超时) 混为一谈, DB 故障时仍继续 INSERT, 产生误导性错误甚至脏数据;
// 现在显式区分两类错误。
func dnsDomainAdd(tx *sql.Tx, domainStr string) (id int64, err error) {

	domainInfo, err := db.GetDomainsByName(domainStr)

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		// 真实 DB 故障, 必须终止, 不能当作"域名不存在"继续插入
		return 0, fmt.Errorf("dnsDomainAdd() query domain %s failed: %w", domainStr, err)
	}

	if domainInfo != nil && domainInfo.ID != 0 {
		return 0, fmt.Errorf("domain %s records exists, id: %d: %w",
			domainStr, domainInfo.ID, errRecordExists)
	}

	// 域名不存在 -> 执行插入
	g.Logger.Infof("dnsDomainAdd() domain %s not found, prepare insert", domainStr)

	var domainDB db.Domain
	domainDB.Type = "NATIVE"
	domainDB.Name = domainStr
	domainDB.Master = nil

	if g.Config().Debug {
		g.Logger.Debugf("dnsDomainAdd() add domain %v", domainDB)
	}

	return db.InsertDomain(tx, domainDB)
}

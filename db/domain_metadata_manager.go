package db

import (
	"database/sql"
	"github.com/signmem/go-woody/g"
)

func InsertDomainMetaData(tx *sql.Tx, domainInfo DomainMeta) (int64, error) {

	query := `INSERT IGNORE INTO domainmetadata (domain_id, kind, content) VALUES (?, ?, ?)`

	result, err := tx.Exec(query, domainInfo.DomainID, domainInfo.Kind, domainInfo.Content)

	if err != nil {
		g.Logger.Errorf("InsertDomainMetaData() exec error: query=%s err=%v", query, err)
		return 0, err
	}

	id, err := result.RowsAffected()

	if err != nil {
		g.Logger.Errorf("InsertDomainMetaData() RowsAffected error: query=%s err=%v", query, err)
		return 0, err
	}

	return id, nil
}

// DeleteDomainMetaData 删除指定domain_id全部domainmetadata元数据，传入事务
func DeleteDomainMetaData(tx *sql.Tx, domain_id int64) (err error) {
	query := `delete from domainmetadata where domain_id = ?`

	_, err = tx.Exec(query, domain_id)

	if err != nil {
		g.Logger.Errorf("DeleteDomainMetaData() delete domain_id=%d err=%v", domain_id, err)
		return err
	}

	return nil
}


func GetDomainMetaData() (domainMeta []DomainMeta, err error) {
	query := "SELECT id, domain_id, content, kind FROM domainmetadata"
	rows, err := DB.Query(query)

	if err != nil {
		g.Logger.Errorf("GetDomainMetaData() query error:%s", err)
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var record DomainMeta
		err := rows.Scan(
			&record.ID,
			&record.DomainID,
			&record.Content,
			&record.Kind,
		)

		if err != nil {
			g.Logger.Errorf("GetDomainMetaData() scan error:%s", err)
			return nil, err
		}
		domainMeta = append(domainMeta, record)
	}

	if err = rows.Err(); err != nil {
		g.Logger.Errorf("GetDomainMetaData() rows iteration error:%v", err)
		return nil, err
	}

	return domainMeta, nil
}


// GetDomainMetaDataByDomainID 根据domain_id查询元数据，业务优先使用此接口
func GetDomainMetaDataByDomainID(domainID int64) (domainMeta []DomainMeta, err error) {
	query := "SELECT id, domain_id, content, kind FROM domainmetadata WHERE domain_id = ?"
	rows, err := DB.Query(query, domainID)
	if err != nil {
		g.Logger.Errorf("GetDomainMetaDataByDomainID() query domain_id=%d err:%v", domainID, err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var record DomainMeta
		err := rows.Scan(
			&record.ID,
			&record.DomainID,
			&record.Content,
			&record.Kind,
		)
		if err != nil {
			g.Logger.Errorf("GetDomainMetaDataByDomainID() scan error:%v", err)
			return nil, err
		}
		domainMeta = append(domainMeta, record)
	}

	if err = rows.Err(); err != nil {
		g.Logger.Errorf("GetDomainMetaDataByDomainID() rows iteration error:%v", err)
		return nil, err
	}

	return domainMeta, nil
}

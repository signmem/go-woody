package db

import (
	"database/sql"
	"github.com/signmem/go-woody/g"
)

func InsertDomainMetaData(tx *sql.Tx, domainInfo DomainMeta) (int64, error) {

	query := `INSERT INTO domainmetadata (domain_id, kind, content) VALUES (?, ?, ?)`

	result, err := tx.Exec(query, domainInfo.DomainID, domainInfo.Kind, domainInfo.Content)

	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()

	if err != nil {
		g.Logger.Errorf("InsertDomainMetaData() %s, error: %s ", query, err)
		return 0, err
	}

	return id, nil
}

func DeleteDomainMetaData(tx *sql.Tx, domain_id int64) (err error) {
	query := `delete from domainmetadata where domain_id = ?`

	_, err = tx.Exec(query, domain_id)

	if err != nil {
		return err
	}

	return nil
}
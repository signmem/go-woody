package db

import (
    "database/sql"
    _  "github.com/go-sql-driver/mysql"
)

type Domain struct {
    ID             int64            `db:"id"`
    Name           string           `db:"name"`
    Master         *string          `db:"master"`
    Type           string           `db:"type"`
}


type Record struct {
    ID       int64          `db:"id"`
    DomainID int64          `db:"domain_id"`
    Name     string         `db:"name"`
    Type     string         `db:"type"`
    Content  string         `db:"content"`
    TTL      int            `db:"ttl"`
}

type DomainMeta struct {
    ID          int64           `db:"id"`
    DomainID    int64           `db:"domain_id"`
    Kind        string          `db:"kind"`
    Content     string          `db:"content"`
}


var DB *sql.DB

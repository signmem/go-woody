package db

import (
    "github.com/signmem/go-woody/g"
    _ "github.com/go-sql-driver/mysql"
    "database/sql"
    "time"
)

func InitDB() error {

	user := g.Config().MySQL.UserName
	pass := g.Config().MySQL.PassWord
	host := g.Config().MySQL.DBHost
	port := g.Config().MySQL.DBPort
	dbName := g.Config().MySQL.DBName

	dataSourceName := user + ":" + pass + "@tcp(" + host + ":" +
		port + ")/" + dbName + "?charset=utf8mb4&parseTime=True"

	maxConnection := g.Config().MySQL.MaxConnection
	maxIdle       := g.Config().MySQL.MaxIdel

    var err error

	DB, err = sql.Open("mysql", dataSourceName)

	if err != nil {
    	g.Logger.Errorf("InitDB() Open error: %s", err)
        return err
    }
    
	err = DB.Ping()
	if err != nil {
		g.Logger.Errorf("InitDB() ping error: %s", err)
		return err
	}
    
	DB.SetMaxOpenConns(maxConnection)
	DB.SetMaxIdleConns(maxIdle)

	DB.SetConnMaxLifetime(1 * time.Hour)
	DB.SetConnMaxIdleTime(30 * time.Minute)

	g.Logger.Infof("InitDB() MySQL connect success, maxOpen: %d, maxIdle: %d",
	maxConnection, maxIdle)
    
    return nil
}


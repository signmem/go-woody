package db

import (
    "github.com/signmem/go-woody/g"
    "database/sql"
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
    maxIdel       := g.Config().MySQL.MaxIdel

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
    DB.SetMaxIdleConns(maxIdel)
    
    return nil
}


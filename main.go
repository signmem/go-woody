package main

import (
	"flag"
	"fmt"
	"github.com/signmem/go-woody/api"
	"github.com/signmem/go-woody/db"
	"github.com/signmem/go-woody/g"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	cfgFile := flag.String("c", "cfg.json", "configuration file")
	version := flag.Bool("v", false, "show version")

	flag.Parse()

	if *version {
		fmt.Printf("go-woody version: %s\n", g.Version)
		os.Exit(0)
	}

	g.ParseConfig(*cfgFile)
	g.Logger = g.InitLog()


	g.Logger.Infof("load config from file: %s", *cfgFile)
	g.Logger.Infof("go-woody version: %s started", g.Version)

	if err := db.InitDB(); err != nil {
		g.Logger.Fatalf("initialize database failed: %v", err)
	}

	defer func() {
		if err := db.DB.Close(); err != nil {
			g.Logger.Errorf("close database connection failed: %v", err)
		} else {
			g.Logger.Info("database connection closed successfully")
		}
	}()
	g.Logger.Info("db initial success.")

	go func() {
		if err := api.Start(); err != nil {
			g.Logger.Fatalf("start api server failed: %v", err)
		}
	}()
	g.Logger.Info("api server is running...")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	// 退出日志
	g.Logger.Info("received exit signal, program shutting down...")

}

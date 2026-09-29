package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/signmem/go-woody/api"
	"github.com/signmem/go-woody/db"
	"github.com/signmem/go-woody/g"
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
		if db.DB != nil {
			if err := db.DB.Close(); err != nil {
				g.Logger.Errorf("close database connection failed: %v", err)
			} else {
				g.Logger.Info("database connection closed successfully")
			}
		}
	}()
	g.Logger.Info("db initial success.")

	srv := api.NewServer()

	// 修复: 原实现在 goroutine 里 Fatalf, 会跳过所有 defer 且无法优雅退出;
	// 现在收到信号或致命错误后先 srv.Shutdown 排空 in-flight 请求,
	// 再由 defer 关闭 DB 连接池
	serverErrChan := make(chan error, 1)
	go func() {
		g.Logger.Infof("api server listening on: %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrChan <- err
		}
	}()
	g.Logger.Info("api server is running...")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErrChan:
		g.Logger.Errorf("api server failed: %v, shutting down...", err)
	case sig := <-sigChan:
		g.Logger.Infof("received signal %s, program shutting down...", sig)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		g.Logger.Errorf("graceful shutdown failed: %v", err)
	}

	g.Logger.Info("go-woody exited.")
}

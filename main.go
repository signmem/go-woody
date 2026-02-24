package main

import (
	"flag"
	"fmt"
	"github.com/signmem/go-woody/api"
	"github.com/signmem/go-woody/db"
	"github.com/signmem/go-woody/g"
	"log"
	"os"
)

func main() {
	cfg := flag.String("c", "cfg.json", "configuration file")
	version := flag.Bool("v", false, "show version")

	flag.Parse()

	if *version {
		version := g.Version
		fmt.Printf("%s", version)
		os.Exit(0)
	}

	g.ParseConfig(*cfg)
	g.Logger = g.InitLog()

	err := db.InitDB()

	if err != nil {
		log.Fatal( err )
	}

	defer db.DB.Close()

	g.Logger.Info("db initial success.")

	go api.Start()
	select {}


}

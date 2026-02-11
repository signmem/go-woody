package g

import (
	"errors"
	"log"
	"os"
	"sync"
	"encoding/json"
	"github.com/signmem/go-woody/tools"
)

var (
	ConfigFile string
	config     *GlobalConfig
	lock       = new(sync.RWMutex)
)


func ParseConfig(cfg string) {
	if cfg == "" {
		log.Fatalln("use -c to specify configuration file")
	}

	if _, err := os.Stat(cfg); errors.Is(err, os.ErrNotExist) {
		// file does not exist
		log.Fatalln("config file:", cfg, "is not existent. maybe you need `mv cfg.example.json cfg.json`")
	}

	ConfigFile = cfg

	configContent, err := tools.ToTrimString(cfg)
	if err != nil {
		log.Fatalln("read config file:", cfg, "fail:", err)
	}

	var c GlobalConfig
	err = json.Unmarshal([]byte(configContent), &c)
	if err != nil {
		log.Fatalln("parse config file:", cfg, "fail:", err)
	}

	lock.Lock()
	defer lock.Unlock()

	config = &c

	log.Println("read config file:", cfg, "successfully")
}

func Config() *GlobalConfig {
	lock.RLock()
	defer lock.RUnlock()
	return config
}

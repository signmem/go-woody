package g

type GlobalConfig struct {
	Debug        bool        `json:"debug"`
	LogFile      string      `json:"log_file"`
	LogMaxAge    int         `json:"log_maxage"`
	LogRotateAge int         `json:"log_rotateage"`
	DnsServer    []string    `json:"dns_server"`
	Http         *HttpConfig `json:"http"`
	ZoneFile     string      `json:"zone_file"`
	Named        bool        `json:"named"`
	DNS          *DNSConfig  `json:"pdns"`
	AutoParent   bool        `json:"auto_parent"`
	// AuthToken API 鉴权 token; 为空时不启用鉴权。
	// 请求头携带 X-Auth-Token: *** 或 Authorization: Bearer <token>
	AuthToken string    `json:"auth_token"`
	MySQL     *DBConfig `json:"mysql"`
}

type DNSConfig struct {
	IP   string `json:"ip"`
	Port string `json:"port"`
}

type HttpConfig struct {
	// 修复: 原 tag 为 "listen", 与 cfg.json/cfg.json.default 中的 "address" 键不一致,
	// 导致监听地址永远解析为空串
	Address string `json:"address"`
	Port    string `json:"port"`
}

type DBConfig struct {
	MaxConnection int `json:"max_connection"`
	// MaxIdle 配置文件键名历史拼写为 max_idel, 为兼容存量 cfg.json 保留原 tag
	MaxIdle  int    `json:"max_idel"`
	UserName string `json:"db_user"`
	PassWord string `json:"db_pass"`
	DBHost   string `json:"db_host"`
	DBPort   string `json:"db_port"`
	DBName   string `json:"db_name"`
}

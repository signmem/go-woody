package g

type GlobalConfig struct {
	Debug			bool		`json:"debug"`
	LogFile			string		`json:"log_file"`
	LogMaxAge		int			`json:"log_maxage"`
	LogRotateAge	int			`json:"log_rotateage"`
	DnsServer		[]string	`json:"dns_server"`
	Http			*HttpConfig	`json:"http"`
	ZoneFile 		string		`json:"zone_file"`
	MySQL			*DBConfig	`json:"mysql"`
}

type HttpConfig struct {
	Address			string		`json:"listen"`
	Port			string		`json:"port"`
}

type DBConfig struct {
	MaxConnection		int		`json:"max_connection"`
	MaxIdel				int		`json:"max_idel"`
	UserName			string	`json:"db_user"`
	PassWord 			string	`json:"db_pass"`
	DBHost				string	`json:"db_host"`
	DBPort				string	`json:"db_port"`
	DBName				string	`json:"db_name"`
}

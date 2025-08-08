package config

const (
	AppName = "aichat-proxy"

	EnvPrefix = "AICHAT_PROXY_"

	ServerAddress = ":9540"
	ProxyAddress  = "127.0.0.1:9543"
	CamoufoxPort  = 9550

	AppRootPath   = "/app"
	UserdataPath  = AppRootPath + "/userdata"
	Userdata0Path = UserdataPath + "/user0"
	DownloadsPath = AppRootPath + "/downloads"
	CertsPath     = AppRootPath + "/certs"
)

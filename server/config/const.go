package config

const (
	AppName = "aichat-proxy"

	EnvPrefix = "AICHAT_PROXY_"

	ServerAddress = ":9540"

	AppRootPath           = "/app"
	AddonPath             = AppRootPath + "/addons"
	AddonTamperMonkeyPath = AddonPath + "/tampermonkey"
	AddonTamperMonkeyName = "firefox@tampermonkey.net"
	UserdataPath          = AppRootPath + "/userdata"
	Userdata0Path         = UserdataPath + "/user0"
	DownloadsPath         = AppRootPath + "/downloads"
)

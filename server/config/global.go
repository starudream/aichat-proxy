package config

type Config struct {
	LogLevel   string `config:"log.level"`
	LogNoColor bool   `config:"log.nocolor"`

	ServerAddr          string `config:"server.addr"`
	ServerFGProfEnabled bool   `config:"server.fgprof.enabled"`
}

var g = &Config{
	LogLevel:   "INFO",
	LogNoColor: false,

	ServerAddr:          ServerAddress,
	ServerFGProfEnabled: false,
}

func G() *Config {
	return g
}

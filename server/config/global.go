package config

type Config struct {
	LogLevel   string `config:"log.level"`
	LogNoColor bool   `config:"log.nocolor"`

	ServerAddr string `config:"server.addr"`

	ApiKeys Array[string] `config:"api.keys"`
}

var g = &Config{
	LogLevel:   "INFO",
	LogNoColor: false,

	ServerAddr: ServerAddress,
}

func G() *Config {
	return g
}

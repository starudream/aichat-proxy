package config

type Config struct {
	LogLevel   string `config:"log.level"`
	LogNoColor bool   `config:"log.nocolor"`

	RedisURL string `config:"redis.url"`

	ServerAddr          string `config:"server.addr"`
	ServerFGProfEnabled bool   `config:"server.fgprof.enabled"`

	ApiKeys Array[string] `config:"api.keys"`
}

var g = &Config{
	LogLevel:   "INFO",
	LogNoColor: false,

	RedisURL: "redis://@redis:6379/0",

	ServerAddr:          ServerAddress,
	ServerFGProfEnabled: false,
}

func G() *Config {
	return g
}

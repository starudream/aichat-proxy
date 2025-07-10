package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/knadh/koanf/parsers/dotenv"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"

	"github.com/starudream/aichat-proxy/server/internal/json"
	"github.com/starudream/aichat-proxy/server/internal/osx"
)

var loaders = []func() (koanf.Provider, koanf.Parser){
	fileDotenvLoader,
	envLoader,
}

func init() {
	k = koanf.New(".")
	for _, loader := range loaders {
		p, pa := loader()
		if p == nil {
			continue
		}
		err := k.Load(p, pa)
		if err != nil {
			name := filepath.Base(osx.FuncName(loader))
			_, _ = fmt.Fprintf(os.Stderr, "init config with %s error: %v\n", name, err)
			os.Exit(1)
		}
	}
	err := k.UnmarshalWithConf("", g, koanf.UnmarshalConf{Tag: "config", FlatPaths: true})
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "unmarshal config error: %v\n", err)
		os.Exit(1)
	}
	if DEBUG("CONFIG") {
		_, _ = fmt.Fprintf(os.Stdout, "config loaded: %v\n", json.MustMarshalToString(g))
	}
}

func envLoader() (koanf.Provider, koanf.Parser) {
	return env.ProviderWithValue(EnvPrefix, ".", envCB), nil
}

func fileDotenvLoader() (koanf.Provider, koanf.Parser) {
	path := strings.TrimSpace(os.Getenv(EnvPrefix + "DOTENV_PATH"))
	if path == "" {
		path = ".env"
		if fi, err := os.Stat(path); err != nil || fi.IsDir() {
			return nil, nil
		}
	}
	return file.Provider(path), dotenv.ParserEnvWithValue("", ".", envCB)
}

func envCB(k, v string) (string, any) {
	k = strings.ToLower(strings.TrimPrefix(k, EnvPrefix))
	k = strings.ReplaceAll(k, "_", ".")
	return k, v
}

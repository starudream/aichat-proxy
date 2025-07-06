package config

import (
	"os"
	"strings"

	"github.com/knadh/koanf/v2"
	"github.com/spf13/cast"
)

var k *koanf.Koanf

func C() *koanf.Koanf {
	return k
}

func To[T cast.Basic](path string) T {
	return cast.To[T](C().Get(path))
}

func DEBUG(names ...string) bool {
	// env MODULE_DEBUG
	for _, name := range names {
		if name == "" {
			continue
		}
		if s := os.Getenv(strings.ToUpper(name) + "_DEBUG"); s != "" {
			return cast.To[bool](s)
		}
	}
	// env DEBUG
	if cast.To[bool](os.Getenv("DEBUG")) {
		return true
	}
	// go test
	for _, arg := range os.Args {
		if strings.HasPrefix(arg, "-test.") {
			return true
		}
	}
	return false
}

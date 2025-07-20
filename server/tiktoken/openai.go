package tiktoken

import (
	_ "embed"
	"encoding/base64"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkoukk/tiktoken-go"

	"github.com/starudream/aichat-proxy/server/logger"
)

var (
	encoding *tiktoken.Tiktoken

	//go:embed o200k_base.tiktoken
	o200k []byte
)

func init() {
	tiktoken.SetBpeLoader(&loader{})

	var err error
	encoding, err = tiktoken.GetEncoding(tiktoken.MODEL_O200K_BASE)
	if err != nil {
		logger.Fatal().Err(err).Msg("tiktoken get encoding error")
	}
}

type loader struct{}

func (l *loader) LoadTiktokenBpe(file string) (map[string]int, error) {
	base := filepath.Base(file)
	switch base {
	case "o200k_base.tiktoken":
		return l.load(o200k)
	default:
		return nil, fmt.Errorf("tiktoken: unsupported file: %s", base)
	}
}

func (l *loader) load(contents []byte) (map[string]int, error) {
	bpeRanks := make(map[string]int)
	for _, line := range strings.Split(string(contents), "\n") {
		if line == "" {
			continue
		}
		parts := strings.Split(line, " ")
		token, err := base64.StdEncoding.DecodeString(parts[0])
		if err != nil {
			return nil, err
		}
		rank, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, err
		}
		bpeRanks[string(token)] = rank
	}
	return bpeRanks, nil
}

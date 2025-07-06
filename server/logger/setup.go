package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	slogzerolog "github.com/samber/slog-zerolog/v2"

	"github.com/starudream/aichat-proxy/server/config"
	"github.com/starudream/aichat-proxy/server/internal/json"
)

func init() {
	zerolog.TimeFieldFormat = "2006-01-02T15:04:05.000Z07:00"
	zerolog.ErrorStackMarshaler = func(e error) any { return strings.TrimSpace(eris.ToString(e, true)) }
	zerolog.ErrorMarshalFunc = func(e error) any { return strings.TrimSpace(eris.ToString(e, true)) }
	zerolog.ErrorHandler = func(err error) { _, _ = fmt.Fprintf(os.Stderr, "non-expected logger error: %v", err) }
	zerolog.InterfaceMarshalFunc = json.Marshal

	lv, err := zerolog.ParseLevel(config.To[string]("log.level"))
	if err != nil || lv == zerolog.NoLevel {
		lv = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(lv)

	zl := zerolog.New(loggerWriters()).With().Timestamp()
	if config.DEBUG() || lv <= zerolog.DebugLevel {
		zl = zl.Caller()
	}
	Logger = zl.Logger()
	log.Logger = Logger
	zerolog.DefaultContextLogger = &Logger

	slog.SetDefault(slog.New(slogzerolog.Option{Level: slog.LevelInfo, Logger: &Logger}.NewZerologHandler()))
}

func loggerWriters() io.Writer {
	return zerolog.MultiLevelWriter(
		zerolog.NewConsoleWriter(func(w *zerolog.ConsoleWriter) {
			w.Out = os.Stdout
			w.NoColor = config.To[bool]("log.no_color")
			w.TimeFormat = zerolog.TimeFieldFormat
			w.TimeLocation = time.Local
		}),
	)
}

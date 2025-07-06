package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/starudream/aichat-proxy/server/browser"
	"github.com/starudream/aichat-proxy/server/config"
	"github.com/starudream/aichat-proxy/server/internal/json"
	"github.com/starudream/aichat-proxy/server/logger"
	"github.com/starudream/aichat-proxy/server/router"
)

func main() {
	if config.DEBUG("CONFIG") {
		logger.Debug().Msgf("loaded config: %s", json.MustMarshal(config.C().Raw()))
	}

	slog.Default().Info("app starting")

	ctx, cancel := context.WithCancel(context.Background())

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	if ctx.Err() == nil {
		go func() {
			select {
			case <-ctx.Done():
			case sig := <-ch:
				println()
				logger.Info().Msgf("received signal: %s", sig.String())
				cancel()
			}
		}()
	}

	logger.Info().Msg("app starting")

	browser.Run(ctx)
	router.Start(ctx)

	<-ctx.Done()

	logger.Info().Msg("app stopped")
}

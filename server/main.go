package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/starudream/aichat-proxy/server/browser"
	"github.com/starudream/aichat-proxy/server/config"
	"github.com/starudream/aichat-proxy/server/logger"
	"github.com/starudream/aichat-proxy/server/router"
)

var (
	pHelp    bool
	pVersion bool
)

func init() {
	flag.BoolVar(&pHelp, "h", false, "show help")
	flag.BoolVar(&pVersion, "v", false, "show version")
	flag.Parse()

	if pHelp {
		flag.Usage()
		os.Exit(0)
	}
	if pVersion {
		fmt.Print(config.GetVersion().String())
		os.Exit(0)
	}
}

func main() {
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

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/starudream/aichat-proxy/server/api"
	"github.com/starudream/aichat-proxy/server/browser"
	"github.com/starudream/aichat-proxy/server/config"
	"github.com/starudream/aichat-proxy/server/logger"
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
	go func() {
		sig := <-ch
		fmt.Println()
		logger.Info().Msgf("received signal: %s", sig.String())
		cancel()
	}()

	logger.Info().Msg("app starting")

	wg := &sync.WaitGroup{}

	browser.Start(ctx, wg)
	api.Start(ctx, wg)

	wg.Wait()

	logger.Info().Msg("app stopped")
}

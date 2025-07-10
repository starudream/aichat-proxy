package redis

import (
	"context"
	"fmt"
	"sync"

	"github.com/redis/rueidis"

	"github.com/starudream/aichat-proxy/server/config"
	"github.com/starudream/aichat-proxy/server/logger"
)

var c rueidis.Client

func Connect(ctx context.Context, wg *sync.WaitGroup) {
	opt, err := rueidis.ParseURL(config.G().RedisURL)
	if err != nil {
		logger.Fatal().Err(err).Msg("redis parse url error")
	}

	c, err = rueidis.NewClient(opt)
	if err != nil {
		logger.Fatal().Err(err).Msg("redis new client error")
	}

	err = c.Do(context.Background(), c.B().Ping().Build()).Error()
	if err != nil {
		logger.Fatal().Err(err).Msg("redis ping error")
	}

	logger.Info().Msg("redis client connected")

	wg.Add(1)

	go func() {
		defer wg.Done()
		<-ctx.Done()
		logger.Warn().Msg("redis client stopping")
		c.Close()
		logger.Info().Msg("redis client stopped")
	}()
}

func C() rueidis.Client {
	return c
}

func B() rueidis.Builder {
	return c.B()
}

func Do(cmd rueidis.Completed) rueidis.RedisResult {
	return c.Do(context.Background(), cmd)
}

func Key(f string, a ...any) string {
	if len(a) > 0 {
		f = fmt.Sprintf(f, a...)
	}
	return config.AppName + ":" + f
}

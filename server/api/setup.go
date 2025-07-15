package api

import (
	"context"
	"errors"
	"fmt"
	stdlog "log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/ziflex/lecho/v3"

	"github.com/starudream/aichat-proxy/server/config"
	"github.com/starudream/aichat-proxy/server/internal/echox"
	"github.com/starudream/aichat-proxy/server/internal/osx"
	"github.com/starudream/aichat-proxy/server/logger"
)

type Ctx = echo.Context

func Start(ctx context.Context, wg *sync.WaitGroup) {
	app := echo.New()
	app.HideBanner = true
	app.HidePort = true
	app.StdLogger = stdlog.Default()
	app.JSONSerializer = echox.JSONSerializer{}
	app.Validator = echox.Validator{}
	app.Logger = lecho.From(logger.Logger)
	app.Debug = config.DEBUG("SERVER")
	app.HTTPErrorHandler = echox.ErrorHandler(app)

	mds := []func() echo.MiddlewareFunc{
		mdRequestId,
		mdRecover,
	}
	for i := range mds {
		md := mds[i]()
		if md == nil {
			continue
		}
		app.Use(md)
	}

	setupRoutes(app)
	setupSwagger(app)

	ln, err := net.Listen("tcp", config.G().ServerAddr)
	if err != nil {
		logger.Fatal().Err(err).Msg("http server listen error")
	}
	app.Listener = ln

	go func() {
		logger.Info().Str("addr", ln.Addr().String()).Msg("http server starting")
		err = app.Start(config.G().ServerAddr)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal().Err(err).Msg("http server run error")
		}
	}()

	wg.Add(1)

	go func() {
		defer wg.Done()
		<-ctx.Done()
		logger.Warn().Msg("http server stopping")
		_ctx, _cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer _cancel()
		_ = app.Shutdown(_ctx)
		logger.Info().Msg("http server stopped")
	}()
}

func mdAuth() echo.MiddlewareFunc {
	keys := map[string]struct{}{}
	for _, v := range config.G().ApiKeys {
		keys[v] = struct{}{}
	}
	if len(keys) == 0 {
		logger.Warn().Msg("api key auth disabled")
		return func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error { return next(c) }
		}
	}
	return middleware.KeyAuthWithConfig(middleware.KeyAuthConfig{
		Validator: func(key string, c echo.Context) (bool, error) {
			_, ok := keys[key]
			if !ok {
				return false, fmt.Errorf("invalid api key")
			}
			return true, nil
		},
	})
}

func mdRequestId() echo.MiddlewareFunc {
	return middleware.RequestIDWithConfig(middleware.RequestIDConfig{
		Generator:    uuid.Must(uuid.NewV7()).String,
		TargetHeader: echo.HeaderXRequestID,
		RequestIDHandler: func(c echo.Context, tid string) {
			// c.Set("requestId", tid)
			c.Request().WithContext(logger.With().Str("requestId", tid).Logger().WithContext(c.Request().Context()))
		},
	})
}

func mdRecover() echo.MiddlewareFunc {
	return middleware.RecoverWithConfig(middleware.RecoverConfig{
		DisablePrintStack: true,
		LogLevel:          4,
		LogErrorFunc: func(c echo.Context, err error, _ []byte) error {
			logger.Ctx(c.Request().Context()).Err(err).Msgf("recover from panic:\n%s", osx.Stack(3))
			return err
		},
	})
}

func mdLogger() echo.MiddlewareFunc {
	return middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogValuesFunc: func(c echo.Context, vs middleware.RequestLoggerValues) error {
			logger.Ctx(c.Request().Context()).
				Info().
				Err(vs.Error).
				Dur("latency", vs.Latency).
				Str("protocol", vs.Protocol).
				Str("ip", vs.RemoteIP).
				Str("method", vs.Method).
				Str("route", vs.RoutePath).
				Str("requestId", vs.RequestID).
				Str("ua", vs.UserAgent).
				Int("status", vs.Status).
				Msg("new request")
			return nil
		},
		LogLatency:   true,
		LogProtocol:  true,
		LogRemoteIP:  true,
		LogMethod:    true,
		LogRoutePath: true,
		LogRequestID: true,
		LogUserAgent: true,
		LogStatus:    true,
		LogError:     true,
	})
}

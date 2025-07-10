package api

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/gofiber/contrib/fgprof"
	"github.com/gofiber/contrib/fiberzerolog"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/keyauth"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/google/uuid"
	"github.com/rotisserie/eris"
	"github.com/spf13/cast"

	"github.com/starudream/aichat-proxy/server/config"
	"github.com/starudream/aichat-proxy/server/internal/json"
	"github.com/starudream/aichat-proxy/server/internal/osx"
	"github.com/starudream/aichat-proxy/server/logger"
)

type Ctx = fiber.Ctx

func Start(ctx context.Context, wg *sync.WaitGroup) {
	// debug := config.DEBUG("SERVER")

	app := fiber.New(fiber.Config{
		AppName:               config.AppName,
		ServerHeader:          config.AppName,
		JSONEncoder:           json.Marshal,
		JSONDecoder:           json.Unmarshal,
		ErrorHandler:          hdrError,
		DisableStartupMessage: true,

		// EnablePrintRoutes:     debug,
		// DisableStartupMessage: !debug,
	})

	mds := []func() fiber.Handler{
		mdRequestId,
		mdRequestIdCtx,
		mdRecover,
		// mdLogger,
		mdFGProf,
	}
	for i := range mds {
		md := mds[i]()
		if md == nil {
			continue
		}
		app.Use(md)
	}

	setupSwagger(app)
	setupRoutes(app)

	ln, err := net.Listen("tcp", config.G().ServerAddr)
	if err != nil {
		logger.Fatal().Err(err).Msg("http server listen error")
	}

	go func() {
		logger.Info().Str("addr", ln.Addr().String()).Msg("http server starting")
		err = app.Listener(ln)
		if err != nil {
			logger.Fatal().Err(err).Msg("http server run error")
		}
	}()

	wg.Add(1)

	go func() {
		defer wg.Done()
		<-ctx.Done()
		logger.Warn().Msg("http server stopping")
		_ = app.ShutdownWithTimeout(3 * time.Second)
		logger.Info().Msg("http server stopped")
	}()
}

func mdAuth() fiber.Handler {
	keys := map[string]struct{}{}
	for _, v := range config.G().ApiKeys {
		keys[v] = struct{}{}
	}
	if len(keys) == 0 {
		logger.Warn().Msg("api key auth disabled")
		return nil
	}
	return keyauth.New(keyauth.Config{
		Validator: func(ctx *fiber.Ctx, s string) (bool, error) {
			_, ok := keys[s]
			if !ok {
				return false, keyauth.ErrMissingOrMalformedAPIKey
			}
			return true, nil
		},
	})
}

func mdRequestId() fiber.Handler {
	return requestid.New(requestid.Config{
		Generator:  uuid.Must(uuid.NewV7()).String,
		ContextKey: fiberzerolog.FieldRequestID,
	})
}

func mdRequestIdCtx() fiber.Handler {
	return func(c *fiber.Ctx) error {
		log := logger.Ctx(c.UserContext()).With().Str(fiberzerolog.FieldRequestID, cast.To[string](c.Locals(fiberzerolog.FieldRequestID)))
		c.SetUserContext(log.Logger().WithContext(c.UserContext()))
		return c.Next()
	}
}

func mdRecover() fiber.Handler {
	return recover.New(recover.Config{
		EnableStackTrace: true,
		StackTraceHandler: func(c *fiber.Ctx, v any) {
			log := logger.Ctx(c.UserContext()).Error()
			if e, ok := v.(error); ok && e != nil {
				log.Err(e).Msgf("recover from panic:\n%s", osx.Stack())
			} else if e != nil {
				log.Msgf("recover from panic: %v", v)
			}
		},
	})
}

func mdLogger() fiber.Handler {
	return fiberzerolog.New(fiberzerolog.Config{
		GetLogger:   func(c *fiber.Ctx) logger.ZLogger { return *logger.Ctx(c.UserContext()) },
		WrapHeaders: true,
		Fields: []string{
			fiberzerolog.FieldProtocol,
			fiberzerolog.FieldIP,
			fiberzerolog.FieldHost,
			fiberzerolog.FieldPath,
			fiberzerolog.FieldURL,
			fiberzerolog.FieldUserAgent,
			fiberzerolog.FieldLatency,
			fiberzerolog.FieldStatus,
			fiberzerolog.FieldRoute,
			fiberzerolog.FieldMethod,
			fiberzerolog.FieldRequestID,
			fiberzerolog.FieldError,
		},
	})
}

func mdFGProf() fiber.Handler {
	if !config.G().ServerFGProfEnabled {
		return nil
	}
	return fgprof.New(fgprof.Config{})
}

func hdrError(c *fiber.Ctx, err error) error {
	var e *fiber.Error
	if !eris.As(err, &e) {
		e = fiber.ErrInternalServerError
	}
	return c.Status(e.Code).JSON(e)
}

func NewError(code int, args ...any) *fiber.Error {
	if len(args) == 0 {
		return fiber.NewError(code)
	}
	switch len(args) {
	case 0:
		return fiber.NewError(code)
	case 1:
		return fiber.NewError(code, cast.To[string](args[0]))
	default:
		return fiber.NewError(code, fmt.Sprintf(cast.To[string](args[0]), args[1:]...))
	}
}

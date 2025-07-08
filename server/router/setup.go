package router

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/gofiber/contrib/fgprof"
	"github.com/gofiber/contrib/fiberzerolog"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/google/uuid"
	"github.com/rotisserie/eris"
	"github.com/spf13/cast"

	"github.com/starudream/aichat-proxy/server/config"
	"github.com/starudream/aichat-proxy/server/internal/json"
	"github.com/starudream/aichat-proxy/server/logger"
)

type Ctx = fiber.Ctx

func Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)

	debug := config.DEBUG("SERVER")

	app := fiber.New(fiber.Config{
		AppName:               config.AppName,
		ServerHeader:          config.AppName,
		JSONEncoder:           json.Marshal,
		JSONDecoder:           json.Unmarshal,
		ErrorHandler:          hdrError,
		EnablePrintRoutes:     debug,
		DisableStartupMessage: !debug,
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

	go func() {
		defer cancel()
		addr := config.G().ServerAddr
		tcpAddr, _ := net.ResolveTCPAddr(app.Config().Network, addr)
		if tcpAddr != nil {
			logger.Info().Str("addr", addr).Msg("http server starting")
		}
		err := app.Listen(addr)
		if err != nil {
			logger.Fatal().Err(err).Msg("http server run error")
		}
	}()

	go func() {
		<-ctx.Done()
		logger.Warn().Msg("http server stopping")
		_ = app.ShutdownWithTimeout(3 * time.Second)
		logger.Info().Msg("http server stopped")
	}()
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
			if e, ok := v.(error); ok {
				log = log.Err(e)
			}
			log.Msgf("recover from panic: %s", cast.To[string](v))
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

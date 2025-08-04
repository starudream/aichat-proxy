package echox

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/starudream/aichat-proxy/server/internal/conv"
	"github.com/starudream/aichat-proxy/server/internal/json"
	"github.com/starudream/aichat-proxy/server/logger"
)

func MiddlewareLogger() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			startTime := time.Now()

			// Common
			log := logger.Ctx(c.Request().Context()).With().
				Str("method", c.Request().Method).
				Str("route", c.Path()).
				Logger()

			// Request
			var reqBody []byte
			if c.Request().Body != nil { // Read
				reqBody, _ = io.ReadAll(c.Request().Body)
			}
			c.Request().Body = io.NopCloser(bytes.NewBuffer(reqBody)) // Reset

			// Callback request
			var reqMsg []byte
			reqType := getContentType(c.Request().Header)
			switch reqType {
			case echo.MIMEApplicationJSON:
				buf := &bytes.Buffer{}
				if json.Compact(buf, reqBody) == nil {
					reqMsg = buf.Bytes()
				}
			}
			reqLog := log.Info().
				Str("ip", c.RealIP()).
				Str("protocol", c.Request().Proto).
				Str("contentType", reqType).
				Str("userAgent", c.Request().UserAgent())
			if len(reqMsg) > 0 {
				reqLog.Msgf("req=%s", conv.BytesToString(reqMsg))
			} else {
				reqLog.Msg("req")
			}

			// Response
			resBody := &bytes.Buffer{}
			mw := io.MultiWriter(c.Response().Writer, resBody)
			writer := &bodyDumpResponseWriter{Writer: mw, ResponseWriter: c.Response().Writer}
			c.Response().Writer = writer

			// Next
			err := next(c)
			if err != nil {
				c.Error(err)
			}

			// Callback response
			var resMsg []byte
			resType := getContentType(c.Response().Header())
			switch resType {
			case echo.MIMEApplicationJSON:
				buf := &bytes.Buffer{}
				if json.Compact(buf, resBody.Bytes()) == nil {
					resMsg = buf.Bytes()
				}
			}
			resLog := log.Err(err).
				Int("status", c.Response().Status).
				Dur("took", time.Since(startTime)).
				Str("contentType", resType)
			if len(resMsg) > 0 {
				resLog.Msgf("resp=%s", conv.BytesToString(resMsg))
			} else {
				resLog.Msg("resp")
			}

			return nil
		}
	}
}

func getContentType(headers http.Header) string {
	base, _, _ := strings.Cut(headers.Get(echo.HeaderContentType), ";")
	return strings.TrimSpace(base)
}

type bodyDumpResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

func (w *bodyDumpResponseWriter) WriteHeader(code int) {
	w.ResponseWriter.WriteHeader(code)
}

func (w *bodyDumpResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func (w *bodyDumpResponseWriter) Flush() {
	err := http.NewResponseController(w.ResponseWriter).Flush()
	if err != nil && errors.Is(err, http.ErrNotSupported) {
		panic(errors.New("response writer flushing is not supported"))
	}
}

func (w *bodyDumpResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return http.NewResponseController(w.ResponseWriter).Hijack()
}

func (w *bodyDumpResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

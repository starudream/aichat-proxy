package browser

import (
	"bufio"
	"compress/gzip"
	"context"
	stdjson "encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/andybalholm/brotli"
	"github.com/elazarl/goproxy"

	"github.com/starudream/aichat-proxy/server/config"
	"github.com/starudream/aichat-proxy/server/internal/conv"
	"github.com/starudream/aichat-proxy/server/internal/json"
	"github.com/starudream/aichat-proxy/server/internal/writer"
	"github.com/starudream/aichat-proxy/server/logger"
)

func startProxy(ctx context.Context, wg *sync.WaitGroup) {
	proxy := goproxy.NewProxyHttpServer()
	proxy.Verbose = config.DEBUG("PROXY")
	proxy.Logger = writer.NewPrefixWriter("proxy")
	proxy.KeepDestinationHeaders = true
	proxy.KeepHeader = true
	proxy.OnRequest(goproxy.ReqConditionFunc(onRequest)).HandleConnectFunc(handleConnect)
	proxy.OnResponse().DoFunc(doResponse)

	srv := &http.Server{Addr: config.ProxyAddress, Handler: proxy}
	ln, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		logger.Fatal().Err(err).Msg("proxy server listen error")
	}

	go func() {
		logger.Info().Str("addr", ln.Addr().String()).Msg("proxy server starting")
		err = srv.Serve(ln)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal().Err(err).Msg("proxy server run error")
		}
	}()

	proxyChs = map[string]chan any{}
	for _, m := range Models() {
		proxyChs[m] = make(chan any, 4096)
	}

	wg.Add(1)

	go func() {
		defer wg.Done()
		<-ctx.Done()
		logger.Warn().Msg("proxy server stopping")
		_ctx, _cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer _cancel()
		_ = srv.Shutdown(_ctx)
		logger.Info().Msg("proxy server stopped")
	}()
}

type mitmModule struct {
	Name         string
	TypePrefix   string
	PathContains string
}

var mitmHosts = map[string]*mitmModule{
	"www.doubao.com:443": {
		Name:         "doubao",
		TypePrefix:   "text/event-stream",
		PathContains: "/chat/completion",
	},
	"chat.qwen.ai:443": {
		Name:         "qwen",
		TypePrefix:   "text/event-stream",
		PathContains: "/chat/completions",
	},
	"yuanbao.tencent.com:443": {
		Name:         "yuanbao",
		TypePrefix:   "text/event-stream",
		PathContains: "/api/chat/",
	},
	"chat.deepseek.com:443": {
		Name:         "deepseek",
		TypePrefix:   "text/event-stream",
		PathContains: "/chat/completion",
	},
	"www.kimi.com:443": {
		Name:         "kimi",
		TypePrefix:   "text/event-stream",
		PathContains: "/completion/stream",
	},
	"alkalimakersuite-pa.clients6.google.com:443": {
		Name:         "google",
		TypePrefix:   "application/json+protobuf",
		PathContains: "GenerateContent",
	},
}

func onRequest(req *http.Request, _ *goproxy.ProxyCtx) bool {
	_, ok := mitmHosts[req.URL.Host]
	if ok {
		logger.Debug().Str("host", req.URL.Host).Msg("proxy request detected")
	}
	return ok
}

func handleConnect(host string, _ *goproxy.ProxyCtx) (*goproxy.ConnectAction, string) {
	return &goproxy.ConnectAction{Action: goproxy.ConnectMitm, TLSConfig: newTLSConfig}, host
}

var proxyChs map[string]chan any

func doResponse(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
	if resp == nil {
		return resp
	}
	module, ok := mitmHosts[ctx.Req.URL.Host]
	if !ok {
		return resp
	}
	contentType := strings.ToLower(resp.Header.Get("Content-Type"))
	contentEncoding := strings.ToLower(resp.Header.Get("Content-Encoding"))
	if strings.HasPrefix(contentType, module.TypePrefix) && strings.Contains(ctx.Req.URL.Path, module.PathContains) {
		logger.Debug().Str("host", ctx.Req.URL.Host).Str("path", ctx.Req.URL.Path).Msg("proxy response detected")
		pr, pw := io.Pipe()
		resp.Body = newTeeReader(resp.Body, pw)
		go func() {
			defer func() { _ = pr.Close() }()
			proxyChs[module.Name] <- true
			logger.Debug().Msg("proxy handle stream start")
			var rr io.Reader = pr
			switch contentEncoding {
			case "gzip":
				rr, _ = gzip.NewReader(pr)
			case "br":
				rr = brotli.NewReader(pr)
			}
			if module.Name == "google" {
				handleStreamGoogle(module, rr)
			} else {
				handleStreamLine(module, rr)
			}
			logger.Debug().Msg("proxy handle stream finish")
			proxyChs[module.Name] <- false
		}()
	}
	return resp
}

func handleStreamGoogle(module *mitmModule, rr io.Reader) {
	dec := stdjson.NewDecoder(rr)
	for i := 1; i <= 2; i++ {
		delim, err := dec.Token()
		if err != nil {
			logger.Error().Err(err).Msgf("proxy read delim %d error", i)
			return
		}
		if r, ok := delim.(stdjson.Delim); !ok || r != '[' {
			logger.Error().Msgf("proxy read delim %d want [, got: %s", i, delim)
			return
		}
	}
	for dec.More() {
		var v []any
		err := dec.Decode(&v)
		if err != nil {
			if !errors.Is(err, io.EOF) {
				logger.Error().Err(err).Msg("proxy decode stream error")
			}
			return
		}
		text := json.MustMarshalToString(v)
		logger.Debug().Msgf("proxy stream raw: %s", text)
		pushStreamEvent(module, text)
	}
	suffix, err := io.ReadAll(dec.Buffered())
	if err != nil {
		logger.Error().Err(err).Msg("proxy read suffix error")
		return
	}
	if conv.BytesToString(suffix) != "]]" {
		logger.Error().Msgf("proxy read suffix want ]], got: %s", suffix)
		return
	}
}

func handleStreamLine(module *mitmModule, rr io.Reader) {
	for rd := bufio.NewReader(rr); ; {
		text, err := rd.ReadString('\n')
		if err != nil {
			if !errors.Is(err, io.EOF) {
				logger.Error().Err(err).Msg("proxy read stream error")
			}
			return
		}
		text = strings.TrimRight(text, "\r\n")
		if text == "" {
			continue
		}
		logger.Debug().Msgf("proxy stream raw: %s", text)
		pushStreamEvent(module, text)
	}
}

func pushStreamEvent(module *mitmModule, text string) {
	select {
	case proxyChs[module.Name] <- text:
		// normal
	default:
		logger.Warn().Msg("proxy stream channel full, drop all messages")
		count := 0
		for {
			out := false
			select {
			case <-proxyChs[module.Name]:
				count++
			default:
				out = true
			}
			if out {
				break
			}
		}
		logger.Info().Int("count", count).Msg("proxy stream channel clear")
	}
}

type teeReader struct {
	io.Reader
	body io.Closer
	pw   io.Closer
}

func newTeeReader(body io.ReadCloser, pw io.WriteCloser) io.ReadCloser {
	return &teeReader{Reader: io.TeeReader(body, pw), body: body, pw: pw}
}

var _ io.ReadCloser = (*teeReader)(nil)

func (t *teeReader) Close() error {
	pe := t.pw.Close()
	be := t.body.Close()
	if pe != nil {
		return pe
	}
	return be
}

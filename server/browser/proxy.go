package browser

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/elazarl/goproxy"
	lru "github.com/hashicorp/golang-lru/v2"

	"github.com/starudream/aichat-proxy/server/config"
	"github.com/starudream/aichat-proxy/server/internal/writer"
	"github.com/starudream/aichat-proxy/server/logger"
)

func startProxy(ctx context.Context, wg *sync.WaitGroup) {
	proxy := goproxy.NewProxyHttpServer()
	proxy.Verbose = config.DEBUG("PROXY")
	proxy.Logger = writer.NewPrefixWriter("proxy")
	proxy.CertStore = newLRUStorage()
	proxy.KeepDestinationHeaders = true
	proxy.KeepHeader = true
	proxy.OnRequest(goproxy.ReqConditionFunc(onRequest)).HandleConnectFunc(goproxy.AlwaysMitm)
	proxy.OnResponse().DoFunc(doResponse)

	srv := &http.Server{Addr: config.ProxyAddress, Handler: proxy}
	ln, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		logger.Fatal().Err(err).Msg("proxy server listen error")
	}

	go func() {
		logger.Info().Int("port", ln.Addr().(*net.TCPAddr).Port).Msg("proxy server starting")
		err = srv.Serve(ln)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal().Err(err).Msg("proxy server run error")
		}
	}()

	proxyCh = make(chan any)

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

var mitmHosts = map[string]struct{}{
	"www.doubao.com:443": {},
}

func onRequest(req *http.Request, _ *goproxy.ProxyCtx) bool {
	_, ok := mitmHosts[req.URL.Host]
	if ok {
		logger.Debug().Str("host", req.URL.Host).Msg("proxy request detected")
	}
	return ok
}

var proxyCh chan any

func doResponse(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
	if resp == nil {
		return resp
	}
	if _, ok := mitmHosts[ctx.Req.URL.Host]; !ok {
		return resp
	}
	contentType := strings.ToLower(resp.Header.Get("Content-Type"))
	if strings.HasSuffix(ctx.Req.URL.Path, "/chat/completion") && strings.HasPrefix(contentType, "text/event-stream") {
		pr, pw := io.Pipe()
		resp.Body = newTeeReader(resp.Body, pw)
		go func() {
			defer func() { _ = pr.Close() }()
			proxyCh <- true
			logger.Debug().Msg("proxy listen sse start")
			sc := bufio.NewScanner(pr)
			for sc.Scan() {
				text := sc.Text()
				if text == "" {
					continue
				}
				logger.Debug().Msgf("proxy sse raw: %s", text)
				proxyCh <- text
			}
			if err := sc.Err(); err != nil {
				logger.Error().Err(err).Msg("proxy listen sse error")
			}
			logger.Debug().Msg("proxy listen sse finish")
			proxyCh <- false
		}()
	}
	return resp
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

type lruStorage struct {
	certs *lru.Cache[string, *tls.Certificate]
}

func newLRUStorage() goproxy.CertStorage {
	certs, _ := lru.NewWithEvict[string, *tls.Certificate](4096, onLRUEvicted)
	return &lruStorage{certs: certs}
}

func onLRUEvicted(hostname string, _ *tls.Certificate) {
	logger.Debug().Str("hostname", hostname).Msg("cert storage evicted")
}

func (s *lruStorage) Fetch(hostname string, gen func() (*tls.Certificate, error)) (cert *tls.Certificate, err error) {
	var ok bool
	cert, ok = s.certs.Get(hostname)
	if !ok {
		cert, err = gen()
		if err != nil {
			return
		}
		s.certs.Add(hostname, cert)
	}
	return
}

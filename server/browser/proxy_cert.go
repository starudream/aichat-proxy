package browser

import (
	"crypto/tls"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/elazarl/goproxy"
	lru "github.com/hashicorp/golang-lru/v2"

	"github.com/starudream/aichat-proxy/server/config"
	"github.com/starudream/aichat-proxy/server/internal/signer"
)

func newTLSConfig(host string, ctx *goproxy.ProxyCtx) (*tls.Config, error) {
	hostname := stripPort(host)
	ctx.Logf("signing for %s", hostname)
	cert, err := newCert(hostname)
	if err != nil {
		ctx.Warnf("Cannot sign host certificate with provided CA: %s", err)
		return nil, err
	}
	return &tls.Config{Certificates: []tls.Certificate{*cert}, InsecureSkipVerify: true}, nil
}

var (
	ca     *tls.Certificate
	caOnce sync.Once

	certs, _ = lru.New[string, *tls.Certificate](64)
)

func newCert(hostname string) (_ *tls.Certificate, err error) {
	caOnce.Do(func() {
		ca, err = signer.ReadCert(filepath.Join(config.CertsPath, config.AppName+".pem"))
		if err != nil {
			ca, err = signer.CreateCA(config.AppName)
			if err == nil {
				_ = signer.SaveCert(ca, filepath.Join(config.CertsPath, config.AppName+".pem"))
			}
		}
	})
	if err != nil {
		return nil, err
	}
	cert, ok := certs.Get(hostname)
	if !ok {
		cert, err = signer.ReadCert(filepath.Join(config.CertsPath, hostname+".pem"))
	}
	if cert != nil && cert.Leaf.NotAfter.After(time.Now().AddDate(0, 0, 7)) {
		certs.Add(hostname, cert)
		return cert, nil
	}
	cert, err = signer.CreateDV(ca, hostname)
	if err != nil {
		return nil, err
	}
	_ = signer.SaveCert(cert, filepath.Join(config.CertsPath, hostname+".pem"))
	certs.Add(hostname, cert)
	return cert, nil
}

func stripPort(s string) string {
	var ix int
	if strings.Contains(s, "[") && strings.Contains(s, "]") {
		// ipv6 address example: [2606:4700:4700::1111]:443
		// strip '[' and ']'
		s = strings.ReplaceAll(s, "[", "")
		s = strings.ReplaceAll(s, "]", "")

		ix = strings.LastIndexAny(s, ":")
		if ix == -1 {
			return s
		}
	} else {
		// ipv4
		ix = strings.IndexRune(s, ':')
		if ix == -1 {
			return s
		}
	}
	return s[:ix]
}

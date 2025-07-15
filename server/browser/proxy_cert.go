package browser

import (
	"crypto/tls"

	"github.com/elazarl/goproxy"
	lru "github.com/hashicorp/golang-lru/v2"

	"github.com/starudream/aichat-proxy/server/logger"
)

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

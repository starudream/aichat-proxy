package signer

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/starudream/aichat-proxy/server/logger"
)

var (
	now   = time.Now()
	today = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	caNotBefore = today.AddDate(-5, 0, 0)
	caNotAfter  = today.AddDate(15, 0, 0)

	caKeyUsage    = x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign
	caExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth}

	dvNotBefore = today.AddDate(0, -1, 0)
	dvNotAfter  = today.AddDate(4, 11, 0)

	dvKeyUsage    = x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment
	dvExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth}
)

func CreateCert(current, parent *x509.Certificate, parentKey *rsa.PrivateKey) (*tls.Certificate, error) {
	defer func(start time.Time) { logger.Debug().Dur("took", time.Since(start)).Msgf("signer: create cert for %s", current.Subject.CommonName) }(time.Now())

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("signer: rsa generate key error: %w", err)
	}
	if parentKey == nil {
		parentKey = key
	}
	crt, err := x509.CreateCertificate(rand.Reader, current, parent, &key.PublicKey, parentKey)
	if err != nil {
		return nil, fmt.Errorf("signer: x509 create certificate error: %w", err)
	}
	cert := &tls.Certificate{
		Certificate: [][]byte{crt},
		PrivateKey:  key,
	}
	cert.Leaf, err = x509.ParseCertificate(crt)
	if err != nil {
		return nil, fmt.Errorf("signer: x509 parse certificate error: %w", err)
	}
	return cert, nil
}

func CreateCA(name string) (*tls.Certificate, error) {
	crt := &x509.Certificate{
		SerialNumber:          genSerialNumber(),
		Subject:               pkix.Name{CommonName: name},
		NotBefore:             caNotBefore,
		NotAfter:              caNotAfter,
		KeyUsage:              caKeyUsage,
		ExtKeyUsage:           caExtKeyUsage,
		IsCA:                  true,
		BasicConstraintsValid: true,
	}
	return CreateCert(crt, crt, nil)
}

func CreateDV(ca *tls.Certificate, names ...string) (*tls.Certificate, error) {
	crt := &x509.Certificate{
		SerialNumber:          genSerialNumber(),
		Issuer:                ca.Leaf.Subject,
		Subject:               pkix.Name{CommonName: ""},
		DNSNames:              []string{},
		IPAddresses:           []net.IP{},
		NotBefore:             dvNotBefore,
		NotAfter:              dvNotAfter,
		KeyUsage:              dvKeyUsage,
		ExtKeyUsage:           dvExtKeyUsage,
		IsCA:                  false,
		BasicConstraintsValid: true,
	}
	for _, name := range names {
		ip := net.ParseIP(name)
		if ip == nil {
			crt.DNSNames = append(crt.DNSNames, name)
			if crt.Subject.CommonName == "" {
				crt.Subject.CommonName = name
			}
		} else {
			crt.IPAddresses = append(crt.IPAddresses, ip)
		}
	}
	return CreateCert(crt, ca.Leaf, ca.PrivateKey.(*rsa.PrivateKey))
}

func SaveCert(cert *tls.Certificate, path string) error {
	defer func() { logger.Debug().Msgf("signer: save cert pem to %s", path) }()
	blocks := make([][]byte, len(cert.Certificate)+1)
	for i, crt := range cert.Certificate {
		blocks[i] = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: crt})
	}
	blocks[len(blocks)-1] = pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(cert.PrivateKey.(*rsa.PrivateKey))})
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("signer: mkdir error: %w", err)
	}
	if err := os.WriteFile(path, bytes.Join(blocks, []byte{'\n'}), 0600); err != nil {
		return fmt.Errorf("signer: write file error: %w", err)
	}
	return nil
}

var serialNumberLimit *big.Int

func genSerialNumber() *big.Int {
	if serialNumberLimit == nil {
		serialNumberLimit = new(big.Int).Lsh(big.NewInt(1), 128)
	}
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		panic(err)
	}
	return serialNumber
}

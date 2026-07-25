package cert

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

func configDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "netcli"), nil
}

func ConfigDir() (string, error) {
	return configDir()
}

func CAPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "ca.crt"), nil
}

func KeyPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "ca.key"), nil
}

func CAExists() (bool, error) {
	caPath, err := CAPath()
	if err != nil {
		return false, err
	}
	keyPath, err := KeyPath()
	if err != nil {
		return false, err
	}
	_, errCa := os.Stat(caPath)
	_, errKey := os.Stat(keyPath)
	return errCa == nil && errKey == nil, nil
}

func EnsureCA() (string, bool, error) {
	caPath, err := CAPath()
	if err != nil {
		return "", false, err
	}

	exists, err := CAExists()
	if err != nil {
		return "", false, err
	}
	if exists {
		return caPath, false, nil
	}

	dir, err := configDir()
	if err != nil {
		return "", false, err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", false, fmt.Errorf("create config dir: %w", err)
	}

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return "", false, fmt.Errorf("generate key: %w", err)
	}

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return "", false, fmt.Errorf("generate serial: %w", err)
	}

	template := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName:   "netcli Local CA",
			Organization: []string{"netcli"},
		},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(10 * 365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            1,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return "", false, fmt.Errorf("create certificate: %w", err)
	}

	keyPath, err := KeyPath()
	if err != nil {
		return "", false, err
	}

	keyFile, err := os.OpenFile(keyPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return "", false, fmt.Errorf("open key file: %w", err)
	}
	defer keyFile.Close()

	keyBytes, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return "", false, fmt.Errorf("marshal key: %w", err)
	}
	if err := pem.Encode(keyFile, &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes}); err != nil {
		return "", false, fmt.Errorf("write key: %w", err)
	}

	certFile, err := os.OpenFile(caPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return "", false, fmt.Errorf("open cert file: %w", err)
	}
	defer certFile.Close()

	if err := pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: certDER}); err != nil {
		return "", false, fmt.Errorf("write cert: %w", err)
	}

	return caPath, true, nil
}

func LoadCA() (*x509.Certificate, *ecdsa.PrivateKey, error) {
	caPath, err := CAPath()
	if err != nil {
		return nil, nil, err
	}
	keyPath, err := KeyPath()
	if err != nil {
		return nil, nil, err
	}

	certPEM, err := os.ReadFile(caPath)
	if err != nil {
		return nil, nil, fmt.Errorf("read CA cert: %w", err)
	}
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, nil, fmt.Errorf("invalid CA cert PEM")
	}
	caCert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("parse CA cert: %w", err)
	}

	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, nil, fmt.Errorf("read CA key: %w", err)
	}
	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil {
		return nil, nil, fmt.Errorf("invalid CA key PEM")
	}
	caKey, err := x509.ParseECPrivateKey(keyBlock.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("parse CA key: %w", err)
	}

	return caCert, caKey, nil
}

package proxy

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"

	"github.com/elazarl/goproxy"
)

const (
	certDir  = ".codex-proxy"
	certFile = "ca-cert.pem"
	keyFile  = "ca-key.pem"
)

// GenerateCACertificate generates a CA certificate for HTTPS interception
func GenerateCACertificate() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	certPath := filepath.Join(homeDir, certDir)
	if err := os.MkdirAll(certPath, 0700); err != nil {
		return fmt.Errorf("failed to create cert directory: %w", err)
	}

	certFilePath := filepath.Join(certPath, certFile)
	keyFilePath := filepath.Join(certPath, keyFile)

	// Check if certificate already exists
	if _, err := os.Stat(certFilePath); err == nil {
		fmt.Printf("Certificate already exists at %s\n", certFilePath)
		return nil
	}

	// Generate RSA key
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("failed to generate RSA key: %w", err)
	}

	// Create certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization:  []string{"Codex Proxy"},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{""},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	// Create certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return fmt.Errorf("failed to create certificate: %w", err)
	}

	// Save certificate
	certOut, err := os.Create(certFilePath)
	if err != nil {
		return fmt.Errorf("failed to create cert file: %w", err)
	}
	defer certOut.Close()

	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certDER}); err != nil {
		return fmt.Errorf("failed to write certificate: %w", err)
	}

	// Save private key
	keyOut, err := os.Create(keyFilePath)
	if err != nil {
		return fmt.Errorf("failed to create key file: %w", err)
	}
	defer keyOut.Close()

	privKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(priv),
	}

	if err := pem.Encode(keyOut, privKeyPEM); err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}

	fmt.Printf("CA certificate generated successfully!\n")
	fmt.Printf("Certificate: %s\n", certFilePath)
	fmt.Printf("Private key: %s\n", keyFilePath)
	fmt.Printf("\nTo trust this certificate:\n")
	fmt.Printf("macOS: security add-trusted-cert -d -r trustRoot -k ~/Library/Keychains/login.keychain %s\n", certFilePath)
	fmt.Printf("Linux: sudo cp %s /usr/local/share/ca-certificates/codex-proxy.crt && sudo update-ca-certificates\n", certFilePath)
	fmt.Printf("Windows: certutil -addstore -f \"ROOT\" %s\n", certFilePath)

	return nil
}

// certExists checks if CA certificate exists
func certExists() bool {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return false
	}

	certPath := filepath.Join(homeDir, certDir, certFile)
	keyPath := filepath.Join(homeDir, certDir, keyFile)

	_, certErr := os.Stat(certPath)
	_, keyErr := os.Stat(keyPath)

	return certErr == nil && keyErr == nil
}

// setupHTTPS configures HTTPS interception
func setupHTTPS(proxy *goproxy.ProxyHttpServer) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	certPath := filepath.Join(homeDir, certDir, certFile)
	keyPath := filepath.Join(homeDir, certDir, keyFile)

	// Load certificate and key
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return fmt.Errorf("failed to load certificate: %w", err)
	}

	// Configure TLS
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	// Set up MITM
	goproxy.GoproxyCa = cert
	goproxy.OkConnect = &goproxy.ConnectAction{
		Action:    goproxy.ConnectAccept,
		TLSConfig: goproxy.TLSConfigFromCA(&cert),
	}
	goproxy.MitmConnect = &goproxy.ConnectAction{
		Action:    goproxy.ConnectMitm,
		TLSConfig: goproxy.TLSConfigFromCA(&cert),
	}
	goproxy.HTTPMitmConnect = &goproxy.ConnectAction{
		Action:    goproxy.ConnectHTTPMitm,
		TLSConfig: goproxy.TLSConfigFromCA(&cert),
	}

	_ = tlsConfig // Use tlsConfig if needed for custom configuration

	return nil
}
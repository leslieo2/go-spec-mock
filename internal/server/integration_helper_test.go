package server

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/leslieo2/go-spec-mock/internal/config"
)

// testServer holds information about a running test server.
type testServer struct {
	httpServer *http.Server
	baseURL    string
}

// startTestServer starts a new server (HTTP or HTTPS) for integration tests.
// It listens on a dynamic port and returns a testServer instance and a cleanup function.
func startTestServer(t *testing.T, cfg *config.Config) (*testServer, func()) {
	t.Helper()

	if cfg.TLS.Enabled {
		if cfg.TLS.CertFile == "" || cfg.TLS.KeyFile == "" {
			tmpDir := t.TempDir()
			certFile, keyFile, err := generateTestCertificates(tmpDir)
			if err != nil {
				t.Fatalf("Failed to generate test certificates: %v", err)
			}
			cfg.TLS.CertFile = certFile
			cfg.TLS.KeyFile = keyFile
		}
	}

	// Use a dynamic port
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("Failed to listen on a dynamic port: %v", err)
	}

	port := listener.Addr().(*net.TCPAddr).Port
	address := fmt.Sprintf("localhost:%d", port)
	cfg.Server.Port = fmt.Sprintf("%d", port) // Update config with the dynamic port

	appServer, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	httpServer := &http.Server{
		Addr:    address,
		Handler: appServer.buildHandler(),
	}
	appServer.server = httpServer // Keep a reference for shutdown

	protocol := "http"
	if cfg.TLS.Enabled {
		protocol = "https"
	}
	baseURL := fmt.Sprintf("%s://%s", protocol, address)

	errCh := make(chan error, 1)
	go func() {
		if cfg.TLS.Enabled {
			errCh <- httpServer.ServeTLS(listener, cfg.TLS.CertFile, cfg.TLS.KeyFile)
		} else {
			errCh <- httpServer.Serve(listener)
		}
	}()

	// Wait for server to be ready
	waitForServerReady(t, baseURL, cfg.TLS.Enabled)

	ts := &testServer{
		httpServer: httpServer,
		baseURL:    baseURL,
	}

	cleanup := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(ctx); err != nil {
			t.Logf("Error shutting down test server: %v", err)
		}

		select {
		case err := <-errCh:
			if err != nil && err != http.ErrServerClosed {
				t.Errorf("Test server returned an error: %v", err)
			}
		default:
		}
	}

	return ts, cleanup
}

func waitForServerReady(t *testing.T, baseURL string, tlsEnabled bool) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)

	client := &http.Client{Timeout: 1 * time.Second}
	if tlsEnabled {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	healthURL := baseURL + "/health"

	for time.Now().Before(deadline) {
		resp, err := client.Get(healthURL)
		if err == nil {
			_ = resp.Body.Close()
			// Any response from the server means it's up.
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("Server at %s failed to start within timeout", baseURL)
}

func generateTestCertificates(tmpDir string) (string, string, error) {
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", err
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "localhost"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{"localhost"},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privKey.PublicKey, privKey)
	if err != nil {
		return "", "", err
	}

	certFile := filepath.Join(tmpDir, "test-cert.pem")
	certOut, _ := os.Create(certFile)
	defer certOut.Close()
	if err = pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certDER}); err != nil {
		return "", "", err
	}

	keyFile := filepath.Join(tmpDir, "test-key.pem")
	keyOut, _ := os.Create(keyFile)
	defer keyOut.Close()

	privKeyBytes, err := x509.MarshalPKCS8PrivateKey(privKey)
	if err != nil {
		return "", "", err
	}
	if err = pem.Encode(keyOut, &pem.Block{Type: "PRIVATE KEY", Bytes: privKeyBytes}); err != nil {
		return "", "", err
	}

	return certFile, keyFile, nil
}

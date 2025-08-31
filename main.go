package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/leslieo2/go-spec-mock/internal/config"
	"github.com/leslieo2/go-spec-mock/internal/hotreload"
	"github.com/leslieo2/go-spec-mock/internal/server"
	"github.com/spf13/pflag"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	// No longer require positional arguments

	// Parse CLI flags
	configFile := pflag.String("config", "", "Path to configuration file (YAML or JSON)")
	specFile := pflag.String("spec-file", "", "Path to OpenAPI specification file")
	port := pflag.String("port", "8080", "Port to run the mock server on")
	host := pflag.String("host", "localhost", "Host to run the mock server on")

	// Server configuration (timeouts and limits moved to config file/env vars only)

	// Hot reload flags
	hotReload := pflag.Bool("hot-reload", true, "Enable hot reload for specification file")

	// Proxy flags
	proxyEnabled := pflag.Bool("proxy-enabled", false, "Enable proxy mode for undefined endpoints")
	proxyTarget := pflag.String("proxy-target", "", "Target server URL for proxy mode")

	// TLS flags
	tlsEnabled := pflag.Bool("tls-enabled", false, "Enable HTTPS/TLS")
	tlsCertFile := pflag.String("tls-cert-file", "", "Path to TLS certificate file")
	tlsKeyFile := pflag.String("tls-key-file", "", "Path to TLS private key file")

	pflag.Parse()

	// Create signal-aware context for graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Create CLI flags struct for configuration loading
	cliFlags := &config.CLIFlags{
		Host:         host,
		Port:         port,
		SpecFile:     specFile,
		HotReload:    hotReload,
		ProxyEnabled: proxyEnabled,
		ProxyTarget:  proxyTarget,
		TLSEnabled:   tlsEnabled,
		TLSCertFile:  tlsCertFile,
		TLSKeyFile:   tlsKeyFile,
	}

	// Load configuration with precedence (CLI > Env > File > Defaults)
	cfg, err := config.LoadConfig(*configFile, cliFlags)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Validate configuration and spec file
	if cfg.SpecFile == "" {
		fmt.Fprintf(os.Stderr, "Error: OpenAPI spec file is required\n\n")
		printUsage()
		os.Exit(1)
	}
	if _, err := os.Stat(cfg.SpecFile); os.IsNotExist(err) {
		return fmt.Errorf("OpenAPI spec file not found: %s", cfg.SpecFile)
	}
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Create mock server with configuration
	mockServer, err := server.New(cfg)
	if err != nil {
		return fmt.Errorf("failed to create mock server: %w", err)
	}

	// Initialize hot reload if enabled
	var hotReloadManager *hotreload.Manager
	if cfg.HotReload.Enabled {
		hotReloadManager, err = hotreload.NewManager()
		if err != nil {
			return fmt.Errorf("failed to create hot reload manager: %w", err)
		}

		// Set debounce time from config
		hotReloadManager.SetDebounceTime(cfg.HotReload.Debounce)

		// Watch the spec file
		if err := hotReloadManager.AddWatch(cfg.SpecFile); err != nil {
			return fmt.Errorf("failed to watch spec file: %w", err)
		}

		// Register the server as a reloadable component
		if err := hotReloadManager.RegisterReloadable(mockServer); err != nil {
			return fmt.Errorf("failed to register server for hot reload: %w", err)
		}

		// Start hot reload manager
		if err := hotReloadManager.Start(); err != nil {
			return fmt.Errorf("failed to start hot reload: %w", err)
		}

		log.Printf("Hot reload enabled for %s", cfg.SpecFile)
	}

	log.Printf("Starting mock server for %s on %s:%s", cfg.SpecFile, cfg.Server.Host, cfg.Server.Port)
	if err := mockServer.Start(); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	// Wait for shutdown signal
	<-ctx.Done()

	// Shutdown hot reload manager if enabled with timeout context
	if hotReloadManager != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := hotReloadManager.Shutdown(shutdownCtx); err != nil {
			log.Printf("Failed to shutdown hot reload manager: %v", err)
		}
	}

	return nil
}

// printUsage prints the usage information
func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [flags]\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "\nRequired:\n")
	fmt.Fprintf(os.Stderr, "  -spec-file\t\tPath to OpenAPI specification file\n")
	fmt.Fprintf(os.Stderr, "\nConfiguration options:\n")
	fmt.Fprintf(os.Stderr, "  -config\t\tPath to configuration file (YAML or JSON)\n")
	fmt.Fprintf(os.Stderr, "\nServer configuration:\n")
	fmt.Fprintf(os.Stderr, "  -host\t\t\tHost to run the mock server on (default: localhost)\n")
	fmt.Fprintf(os.Stderr, "  -port\t\t\tPort to run the mock server on (default: 8080)\n")
	fmt.Fprintf(os.Stderr, "\nProxy flags:\n")
	fmt.Fprintf(os.Stderr, "  -proxy-enabled\t\tEnable proxy mode for undefined endpoints (default: false)\n")
	fmt.Fprintf(os.Stderr, "  -proxy-target\t\tTarget server URL for proxy mode\n")
	fmt.Fprintf(os.Stderr, "\nTLS flags:\n")
	fmt.Fprintf(os.Stderr, "  -tls-enabled\t\tEnable HTTPS/TLS (default: false)\n")
	fmt.Fprintf(os.Stderr, "  -tls-cert-file\t\tPath to TLS certificate file\n")
	fmt.Fprintf(os.Stderr, "  -tls-key-file\t\tPath to TLS private key file\n")
	fmt.Fprintf(os.Stderr, "\nHot reload flags:\n")
	fmt.Fprintf(os.Stderr, "  -hot-reload\t\tEnable hot reload for specification file (default: true)\n")
	fmt.Fprintf(os.Stderr, "\nEnvironment variables:\n")
	fmt.Fprintf(os.Stderr, "  GO_SPEC_MOCK_HOST, GO_SPEC_MOCK_PORT\n")
	fmt.Fprintf(os.Stderr, "  GO_SPEC_MOCK_READ_TIMEOUT, GO_SPEC_MOCK_WRITE_TIMEOUT, GO_SPEC_MOCK_IDLE_TIMEOUT\n")
	fmt.Fprintf(os.Stderr, "  GO_SPEC_MOCK_MAX_REQUEST_SIZE, GO_SPEC_MOCK_SHUTDOWN_TIMEOUT, GO_SPEC_MOCK_SPEC_FILE\n")
	fmt.Fprintf(os.Stderr, "  GO_SPEC_MOCK_HOT_RELOAD, GO_SPEC_MOCK_HOT_RELOAD_DEBOUNCE\n")
	fmt.Fprintf(os.Stderr, "\nConfiguration file:\n")
	fmt.Fprintf(os.Stderr, "  go-spec-mock.yaml (default configuration file)\n")
	fmt.Fprintf(os.Stderr, "\nExample usage:\n")
	fmt.Fprintf(os.Stderr, "  %s -spec-file ./examples/petstore.yaml -proxy-enabled\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s -spec-file ./examples/petstore.yaml -config ./config.yaml\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s -spec-file ./examples/petstore.yaml -port 8081\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  GO_SPEC_MOCK_PORT=8081 %s -spec-file ./examples/petstore.yaml\n", os.Args[0])
}

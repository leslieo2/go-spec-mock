package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/leslieo2/go-spec-mock/internal/config"
	"github.com/leslieo2/go-spec-mock/internal/constants"
	"github.com/leslieo2/go-spec-mock/internal/hotreload"
	"github.com/leslieo2/go-spec-mock/internal/server"
	"github.com/spf13/pflag"
)

func main() {
	// No longer require positional arguments

	// Parse CLI flags
	configFile := pflag.String("config", "", "Path to configuration file (YAML or JSON)")
	specFile := pflag.String("spec-file", "", "Path to OpenAPI specification file")
	port := pflag.String("port", "8080", "Port to run the mock server on")
	host := pflag.String("host", "localhost", "Host to run the mock server on")
	metricsPort := pflag.String("metrics-port", "9090", "Port to run the metrics server on")

	// Server configuration (timeouts and limits moved to config file/env vars only)

	// Security flags
	rateLimitEnabled := pflag.Bool("rate-limit-enabled", false, "Enable rate limiting")
	rateLimitStrategy := pflag.String("rate-limit-strategy", constants.RateLimitStrategyIP, "Rate limiting strategy: ip, api_key")

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

	// Create CLI flags struct for configuration loading
	cliFlags := &config.CLIFlags{
		Host:              host,
		Port:              port,
		MetricsPort:       metricsPort,
		SpecFile:          specFile,
		RateLimitEnabled:  rateLimitEnabled,
		RateLimitStrategy: rateLimitStrategy,
		HotReload:         hotReload,
		ProxyEnabled:      proxyEnabled,
		ProxyTarget:       proxyTarget,
		TLSEnabled:        tlsEnabled,
		TLSCertFile:       tlsCertFile,
		TLSKeyFile:        tlsKeyFile,
	}

	// Load configuration with precedence (CLI > Env > File > Defaults)
	cfg, err := config.LoadConfig(*configFile, cliFlags)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Validate configuration and spec file
	if cfg.SpecFile == "" {
		fmt.Fprintf(os.Stderr, "Error: OpenAPI spec file is required\n\n")
		printUsage()
		os.Exit(1)
	}
	if _, err := os.Stat(cfg.SpecFile); os.IsNotExist(err) {
		log.Fatalf("OpenAPI spec file not found: %s", cfg.SpecFile)
	}
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	// Create mock server with configuration
	mockServer, err := server.New(cfg)
	if err != nil {
		log.Fatalf("Failed to create mock server: %v", err)
	}

	// Initialize hot reload if enabled
	var hotReloadManager *hotreload.Manager
	if cfg.HotReload.Enabled {
		hotReloadManager, err = hotreload.NewManager()
		if err != nil {
			log.Fatalf("Failed to create hot reload manager: %v", err)
		}

		// Set debounce time from config
		hotReloadManager.SetDebounceTime(cfg.HotReload.Debounce)

		// Watch the spec file
		if err := hotReloadManager.AddWatch(cfg.SpecFile); err != nil {
			log.Fatalf("Failed to watch spec file: %v", err)
		}

		// Register the server as a reloadable component
		if err := hotReloadManager.RegisterReloadable(mockServer); err != nil {
			log.Fatalf("Failed to register server for hot reload: %v", err)
		}

		// Start hot reload manager
		if err := hotReloadManager.Start(); err != nil {
			log.Fatalf("Failed to start hot reload: %v", err)
		}

		log.Printf("Hot reload enabled for %s", cfg.SpecFile)
	}

	log.Printf("Starting mock server for %s on %s:%s", cfg.SpecFile, cfg.Server.Host, cfg.Server.Port)
	if cfg.Security.RateLimit.Enabled {
		log.Printf("Rate limiting enabled (strategy: %s)",
			cfg.Security.RateLimit.Strategy)
	}

	if err := mockServer.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	// Shutdown hot reload manager if enabled
	if hotReloadManager != nil {
		if err := hotReloadManager.Shutdown(context.Background()); err != nil {
			log.Printf("Failed to shutdown hot reload manager: %v", err)
		}
	}
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
	fmt.Fprintf(os.Stderr, "  -metrics-port\t\tPort to run the metrics server on (default: 9090)\n")
	fmt.Fprintf(os.Stderr, "\nSecurity flags:\n")
	fmt.Fprintf(os.Stderr, "  -rate-limit-enabled\tEnable rate limiting (default: false)\n")
	fmt.Fprintf(os.Stderr, "  -rate-limit-strategy\tRate limiting strategy: ip (default: ip)\n")
	fmt.Fprintf(os.Stderr, "\nHot reload flags:\n")
	fmt.Fprintf(os.Stderr, "  -hot-reload\t\tEnable hot reload for specification file (default: true)\n")
	fmt.Fprintf(os.Stderr, "\nEnvironment variables:\n")
	fmt.Fprintf(os.Stderr, "  GO_SPEC_MOCK_HOST, GO_SPEC_MOCK_PORT, GO_SPEC_MOCK_METRICS_PORT\n")
	fmt.Fprintf(os.Stderr, "  GO_SPEC_MOCK_READ_TIMEOUT, GO_SPEC_MOCK_WRITE_TIMEOUT, GO_SPEC_MOCK_IDLE_TIMEOUT\n")
	fmt.Fprintf(os.Stderr, "  GO_SPEC_MOCK_MAX_REQUEST_SIZE, GO_SPEC_MOCK_SHUTDOWN_TIMEOUT, GO_SPEC_MOCK_SPEC_FILE\n")
	fmt.Fprintf(os.Stderr, "  GO_SPEC_MOCK_HOT_RELOAD, GO_SPEC_MOCK_HOT_RELOAD_DEBOUNCE\n")
	fmt.Fprintf(os.Stderr, "\nConfiguration file:\n")
	fmt.Fprintf(os.Stderr, "  go-spec-mock.yaml (default configuration file)\n")
	fmt.Fprintf(os.Stderr, "\nExample usage:\n")
	fmt.Fprintf(os.Stderr, "  %s -spec-file ./examples/petstore.yaml -rate-limit-enabled\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s -spec-file ./examples/petstore.yaml -config ./config.yaml\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s -spec-file ./examples/petstore.yaml -port 8081 -metrics-port 9091\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  GO_SPEC_MOCK_PORT=8081 %s -spec-file ./examples/petstore.yaml\n", os.Args[0])
}

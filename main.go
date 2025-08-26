package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/leslieo2/go-spec-mock/internal/config"
	"github.com/leslieo2/go-spec-mock/internal/constants"
	"github.com/leslieo2/go-spec-mock/internal/hotreload"
	"github.com/leslieo2/go-spec-mock/internal/security"
	"github.com/leslieo2/go-spec-mock/internal/server"
)

func main() {
	// No longer require positional arguments

	// Parse CLI flags
	configFile := flag.String("config", "", "Path to configuration file (YAML or JSON)")
	specFile := flag.String("spec-file", "", "Path to OpenAPI specification file")
	port := flag.String("port", "8080", "Port to run the mock server on")
	host := flag.String("host", "localhost", "Host to run the mock server on")
	metricsPort := flag.String("metrics-port", "9090", "Port to run the metrics server on")

	// Server configuration
	readTimeout := flag.Duration("read-timeout", 15*time.Second, "HTTP server read timeout")
	writeTimeout := flag.Duration("write-timeout", 15*time.Second, "HTTP server write timeout")
	idleTimeout := flag.Duration("idle-timeout", 60*time.Second, "HTTP server idle timeout")
	maxRequestSize := flag.Int64("max-request-size", 10*1024*1024, "Maximum request size in bytes")
	shutdownTimeout := flag.Duration("shutdown-timeout", 30*time.Second, "Graceful shutdown timeout")

	// Security flags
	authEnabled := flag.Bool("auth-enabled", false, "Enable API key authentication")
	rateLimitEnabled := flag.Bool("rate-limit-enabled", false, "Enable rate limiting")
	rateLimitStrategy := flag.String("rate-limit-strategy", constants.RateLimitStrategyIP, "Rate limiting strategy: ip, api_key, both")
	rateLimitRPS := flag.Int("rate-limit-rps", 100, "Global rate limit requests per second")
	generateKey := flag.String("generate-key", "", "Generate a new API key with given name")

	// Hot reload flags
	hotReload := flag.Bool("hot-reload", true, "Enable hot reload for specification file")
	hotReloadDebounce := flag.Duration("hot-reload-debounce", 500*time.Millisecond, "Debounce time for hot reload events")

	// Proxy flags
	proxyEnabled := flag.Bool("proxy-enabled", false, "Enable proxy mode for undefined endpoints")
	proxyTarget := flag.String("proxy-target", "", "Target server URL for proxy mode")
	proxyTimeout := flag.Duration("proxy-timeout", 30*time.Second, "Timeout for proxy requests")

	if err := flag.CommandLine.Parse(os.Args[1:]); err != nil {
		log.Fatalf("Failed to parse flags: %v", err)
	}

	// Create CLI flags struct for configuration loading
	cliFlags := &config.CLIFlags{
		Host:              host,
		Port:              port,
		MetricsPort:       metricsPort,
		SpecFile:          specFile,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
		MaxRequestSize:    maxRequestSize,
		ShutdownTimeout:   shutdownTimeout,
		AuthEnabled:       authEnabled,
		RateLimitEnabled:  rateLimitEnabled,
		RateLimitStrategy: rateLimitStrategy,
		RateLimitRPS:      rateLimitRPS,
		GenerateKey:       generateKey,
		HotReload:         hotReload,
		HotReloadDebounce: hotReloadDebounce,
		ProxyEnabled:      proxyEnabled,
		ProxyTarget:       proxyTarget,
		ProxyTimeout:      proxyTimeout,
	}

	// Load configuration with precedence (CLI > Env > File > Defaults)
	cfg, err := config.LoadConfig(*configFile, cliFlags)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Skip validation for help and version flags
	if *generateKey != "" {
		// Handle key generation without validation
	} else {
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
	}

	// Handle key generation
	if *generateKey != "" {
		authManager := security.NewAuthManager(&cfg.Security.Auth)
		apiKey, err := authManager.GenerateAPIKey(*generateKey)
		if err != nil {
			log.Fatalf("Failed to generate API key: %v", err)
		}

		fmt.Printf("Generated API key for '%s':\n", *generateKey)
		fmt.Printf("Key: %s\n", apiKey.Key)
		fmt.Printf("Created: %s\n", apiKey.CreatedAt.Format(time.RFC3339))
		fmt.Printf("\nAdd this to your security configuration:\n")
		fmt.Printf("keys:\n")
		fmt.Printf("  - key: %s\n", apiKey.Key)
		fmt.Printf("    name: %s\n", apiKey.Name)
		fmt.Printf("    enabled: true\n")
		os.Exit(0)
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
	if cfg.Security.Auth.Enabled {
		log.Printf("API key authentication enabled")
	}
	if cfg.Security.RateLimit.Enabled {
		log.Printf("Rate limiting enabled (strategy: %s, rps: %d)",
			cfg.Security.RateLimit.Strategy,
			cfg.Security.RateLimit.Global.RequestsPerSecond)
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
	fmt.Fprintf(os.Stderr, "  -read-timeout\t\tHTTP server read timeout (default: 15s)\n")
	fmt.Fprintf(os.Stderr, "  -write-timeout\tHTTP server write timeout (default: 15s)\n")
	fmt.Fprintf(os.Stderr, "  -idle-timeout\t\tHTTP server idle timeout (default: 60s)\n")
	fmt.Fprintf(os.Stderr, "  -max-request-size\tMaximum request size in bytes (default: 10485760)\n")
	fmt.Fprintf(os.Stderr, "  -shutdown-timeout\tGraceful shutdown timeout (default: 30s)\n")
	fmt.Fprintf(os.Stderr, "\nSecurity flags:\n")
	fmt.Fprintf(os.Stderr, "  -auth-enabled\t\tEnable API key authentication (default: false)\n")
	fmt.Fprintf(os.Stderr, "  -rate-limit-enabled\tEnable rate limiting (default: false)\n")
	fmt.Fprintf(os.Stderr, "  -rate-limit-strategy\tRate limiting strategy: ip, api_key, both (default: ip)\n")
	fmt.Fprintf(os.Stderr, "  -rate-limit-rps\t\tGlobal rate limit requests per second (default: 100)\n")
	fmt.Fprintf(os.Stderr, "  -generate-key\t\tGenerate a new API key with given name\n")
	fmt.Fprintf(os.Stderr, "\nHot reload flags:\n")
	fmt.Fprintf(os.Stderr, "  -hot-reload\t\tEnable hot reload for specification file (default: true)\n")
	fmt.Fprintf(os.Stderr, "  -hot-reload-debounce\tDebounce time for hot reload events (default: 500ms)\n")
	fmt.Fprintf(os.Stderr, "\nEnvironment variables:\n")
	fmt.Fprintf(os.Stderr, "  GO_SPEC_MOCK_HOST, GO_SPEC_MOCK_PORT, GO_SPEC_MOCK_METRICS_PORT\n")
	fmt.Fprintf(os.Stderr, "  GO_SPEC_MOCK_READ_TIMEOUT, GO_SPEC_MOCK_WRITE_TIMEOUT, GO_SPEC_MOCK_IDLE_TIMEOUT\n")
	fmt.Fprintf(os.Stderr, "  GO_SPEC_MOCK_MAX_REQUEST_SIZE, GO_SPEC_MOCK_SHUTDOWN_TIMEOUT, GO_SPEC_MOCK_SPEC_FILE\n")
	fmt.Fprintf(os.Stderr, "  GO_SPEC_MOCK_HOT_RELOAD, GO_SPEC_MOCK_HOT_RELOAD_DEBOUNCE\n")
	fmt.Fprintf(os.Stderr, "\nConfiguration file:\n")
	fmt.Fprintf(os.Stderr, "  go-spec-mock.yaml (default configuration file)\n")
	fmt.Fprintf(os.Stderr, "\nExample usage:\n")
	fmt.Fprintf(os.Stderr, "  %s -spec-file ./examples/petstore.yaml -auth-enabled -rate-limit-enabled\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s -spec-file ./examples/petstore.yaml -config ./config.yaml\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s -spec-file ./examples/petstore.yaml -port 8081 -metrics-port 9091\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  GO_SPEC_MOCK_PORT=8081 %s -spec-file ./examples/petstore.yaml\n", os.Args[0])
}

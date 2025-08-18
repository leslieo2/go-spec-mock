package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/leslieo2/go-spec-mock/internal/security"
	"github.com/leslieo2/go-spec-mock/internal/server"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <openapi-spec-file> [flags]\n", os.Args[0])
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
		fmt.Fprintf(os.Stderr, "  -auth-config\t\tPath to security configuration file\n")
		fmt.Fprintf(os.Stderr, "  -rate-limit-enabled\tEnable rate limiting (default: false)\n")
		fmt.Fprintf(os.Stderr, "  -rate-limit-strategy\tRate limiting strategy: ip, api_key, both (default: ip)\n")
		fmt.Fprintf(os.Stderr, "  -rate-limit-rps\t\tGlobal rate limit requests per second (default: 100)\n")
		fmt.Fprintf(os.Stderr, "  -generate-key\t\tGenerate a new API key with given name\n")
		fmt.Fprintf(os.Stderr, "\nEnvironment variables:\n")
		fmt.Fprintf(os.Stderr, "  GO_SPEC_MOCK_HOST, GO_SPEC_MOCK_PORT, GO_SPEC_MOCK_METRICS_PORT\n")
		fmt.Fprintf(os.Stderr, "  GO_SPEC_MOCK_READ_TIMEOUT, GO_SPEC_MOCK_WRITE_TIMEOUT, GO_SPEC_MOCK_IDLE_TIMEOUT\n")
		fmt.Fprintf(os.Stderr, "  GO_SPEC_MOCK_MAX_REQUEST_SIZE, GO_SPEC_MOCK_SHUTDOWN_TIMEOUT\n")
		fmt.Fprintf(os.Stderr, "\nExample usage:\n")
		fmt.Fprintf(os.Stderr, "  %s ./examples/petstore.yaml -auth-enabled -rate-limit-enabled\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s ./examples/petstore.yaml -port 8081 -metrics-port 9091\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s ./examples/petstore.yaml -read-timeout 30s -write-timeout 30s\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  GO_SPEC_MOCK_PORT=8081 %s ./examples/petstore.yaml\n", os.Args[0])
		os.Exit(1)
	}

	specFile := os.Args[1]
	port := flag.String("port", getEnvDefault("GO_SPEC_MOCK_PORT", "8080"), "Port to run the mock server on")
	host := flag.String("host", getEnvDefault("GO_SPEC_MOCK_HOST", "localhost"), "Host to run the mock server on")
	metricsPort := flag.String("metrics-port", getEnvDefault("GO_SPEC_MOCK_METRICS_PORT", "9090"), "Port to run the metrics server on")

	// Server configuration
	readTimeout := flag.Duration("read-timeout", getDurationEnvDefault("GO_SPEC_MOCK_READ_TIMEOUT", 15*time.Second), "HTTP server read timeout")
	writeTimeout := flag.Duration("write-timeout", getDurationEnvDefault("GO_SPEC_MOCK_WRITE_TIMEOUT", 15*time.Second), "HTTP server write timeout")
	idleTimeout := flag.Duration("idle-timeout", getDurationEnvDefault("GO_SPEC_MOCK_IDLE_TIMEOUT", 60*time.Second), "HTTP server idle timeout")
	maxRequestSize := flag.Int64("max-request-size", getInt64EnvDefault("GO_SPEC_MOCK_MAX_REQUEST_SIZE", 10*1024*1024), "Maximum request size in bytes")
	shutdownTimeout := flag.Duration("shutdown-timeout", getDurationEnvDefault("GO_SPEC_MOCK_SHUTDOWN_TIMEOUT", 30*time.Second), "Graceful shutdown timeout")

	// Security flags
	authEnabled := flag.Bool("auth-enabled", false, "Enable API key authentication")
	authConfig := flag.String("auth-config", "", "Path to security configuration file")
	rateLimitEnabled := flag.Bool("rate-limit-enabled", false, "Enable rate limiting")
	rateLimitStrategy := flag.String("rate-limit-strategy", "ip", "Rate limiting strategy: ip, api_key, both")
	rateLimitRPS := flag.Int("rate-limit-rps", 100, "Global rate limit requests per second")
	generateKey := flag.String("generate-key", "", "Generate a new API key with given name")

	if err := flag.CommandLine.Parse(os.Args[2:]); err != nil {
		log.Fatalf("Failed to parse flags: %v", err)
	}

	if _, err := os.Stat(specFile); os.IsNotExist(err) {
		log.Fatalf("OpenAPI spec file not found: %s", specFile)
	}

	// Load security configuration
	securityConfig, err := security.LoadConfig(*authConfig)
	if err != nil {
		log.Fatalf("Failed to load security configuration: %v", err)
	}

	// Override security settings from CLI flags
	if *authEnabled {
		securityConfig.Auth.Enabled = true
	}
	if *rateLimitEnabled {
		securityConfig.RateLimit.Enabled = true
		securityConfig.RateLimit.Strategy = *rateLimitStrategy
		securityConfig.RateLimit.Global.RequestsPerSecond = *rateLimitRPS
	}

	// Handle key generation
	if *generateKey != "" {
		authManager := security.NewAuthManager(&securityConfig.Auth)
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

	mockServer, err := server.New(specFile, *host, *port, securityConfig, *metricsPort, *readTimeout, *writeTimeout, *idleTimeout, *shutdownTimeout, *maxRequestSize)
	if err != nil {
		log.Fatalf("Failed to create mock server: %v", err)
	}

	log.Printf("Starting mock server for %s on %s:%s", specFile, *host, *port)
	if securityConfig.Auth.Enabled {
		log.Printf("API key authentication enabled")
	}
	if securityConfig.RateLimit.Enabled {
		log.Printf("Rate limiting enabled (strategy: %s, rps: %d)",
			securityConfig.RateLimit.Strategy,
			securityConfig.RateLimit.Global.RequestsPerSecond)
	}

	if err := mockServer.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// Helper functions for environment variable defaults
func getEnvDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getDurationEnvDefault(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func getInt64EnvDefault(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intVal
		}
	}
	return defaultValue
}

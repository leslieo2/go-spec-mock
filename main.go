package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/leslieo2/go-spec-mock/internal/security"
	"github.com/leslieo2/go-spec-mock/internal/server"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <openapi-spec-file> [flags]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nSecurity flags:\n")
		fmt.Fprintf(os.Stderr, "  -auth-enabled\t\tEnable API key authentication (default: false)\n")
		fmt.Fprintf(os.Stderr, "  -auth-config\t\tPath to security configuration file\n")
		fmt.Fprintf(os.Stderr, "  -rate-limit-enabled\tEnable rate limiting (default: false)\n")
		fmt.Fprintf(os.Stderr, "  -rate-limit-strategy\tRate limiting strategy: ip, api_key, both (default: ip)\n")
		fmt.Fprintf(os.Stderr, "  -rate-limit-rps\t\tGlobal rate limit requests per second (default: 100)\n")
		fmt.Fprintf(os.Stderr, "\nExample usage:\n")
		fmt.Fprintf(os.Stderr, "  %s ./examples/petstore.yaml -auth-enabled -rate-limit-enabled\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s ./examples/petstore.yaml -auth-config ./security.yaml\n", os.Args[0])
		os.Exit(1)
	}

	specFile := os.Args[1]
	port := flag.String("port", "8080", "Port to run the mock server on")
	host := flag.String("host", "localhost", "Host to run the mock server on")

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

	mockServer, err := server.New(specFile, *host, *port, securityConfig)
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

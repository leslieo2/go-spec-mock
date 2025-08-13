package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/leslieo2/go-spec-mock/internal/server"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <openapi-spec-file> [flags]\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}

	specFile := os.Args[1]
	port := flag.String("port", "8080", "Port to run the mock server on")
	host := flag.String("host", "localhost", "Host to run the mock server on")

	if err := flag.CommandLine.Parse(os.Args[2:]); err != nil {
		log.Fatalf("Failed to parse flags: %v", err)
	}

	if _, err := os.Stat(specFile); os.IsNotExist(err) {
		log.Fatalf("OpenAPI spec file not found: %s", specFile)
	}

	mockServer, err := server.New(specFile, *host, *port)
	if err != nil {
		log.Fatalf("Failed to create mock server: %v", err)
	}

	log.Printf("Starting mock server for %s on %s:%s", specFile, *host, *port)
	if err := mockServer.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

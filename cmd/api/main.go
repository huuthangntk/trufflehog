package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/trufflesecurity/trufflehog/v3/pkg/api"
)

func main() {
	addr := flag.String("addr", ":8080", "HTTP server address")
	flag.Parse()

	server, err := api.NewServer()
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	fmt.Printf("Starting TruffleHog API server on %s\n", *addr)
	fmt.Println("Endpoints:")
	fmt.Println("  POST   /api/v1/scan         - Initiate a new scan")
	fmt.Println("  GET    /api/v1/scan/status  - Get scan status")
	fmt.Println("  GET    /api/v1/scans        - List all scans")
	fmt.Println("  GET    /health              - Health check")

	if err := server.Start(*addr); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}

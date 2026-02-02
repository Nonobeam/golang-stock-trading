package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/nonobeam/golang-stock-trading/internal/client/ml"
)

func main() {
	// Load .env
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found")
	}

	port := os.Getenv("ML_SERVICE_PORT")
	if port == "" {
		port = "50051"
	}
	host := os.Getenv("ML_SERVICE_HOST")
	if host == "" {
		host = "localhost"
	}

	address := fmt.Sprintf("%s:%s", host, port)
	log.Printf("Connecting to ML service at %s...", address)

	client, err := ml.NewMLClient(address)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Ping
	start := time.Now()
	resp, err := client.Ping(context.Background())
	if err != nil {
		log.Fatalf("Ping failed: %v", err)
	}
	latency := time.Since(start)

	log.Printf("Ping successful!")
	log.Printf("Message: %s", resp.Message)
	log.Printf("Server Time: %s", resp.ServerTime)
	log.Printf("Version: %s", resp.Version)
	log.Printf("Latency: %v", latency)
}

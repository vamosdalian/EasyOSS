package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/vamosdalian/EasyOSS/internal/server"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	// Define command-line flags
	s3Port := flag.Int("s3-port", 9000, "S3 API port")
	webPort := flag.Int("web-port", 9001, "Web UI port")
	port := flag.Int("port", 0, "Combined port for both S3 API and Web UI (overrides s3-port and web-port if set)")
	dataPath := flag.String("data", "./data", "Data storage path")
	metaPath := flag.String("meta", "", "Metadata storage path (defaults to <data>/.meta)")
	showVersion := flag.Bool("version", false, "Show version information")

	flag.Parse()

	if *showVersion {
		fmt.Printf("EasyOSS %s\n", version)
		fmt.Printf("Commit: %s\n", commit)
		fmt.Printf("Built: %s\n", date)
		return
	}

	// Apply environment variable overrides
	config := &server.Config{
		S3Port:   *s3Port,
		WebPort:  *webPort,
		DataPath: *dataPath,
		MetaPath: *metaPath,
	}

	// Environment variable: EASYOSS_PORT (combined port)
	if envPort := os.Getenv("EASYOSS_PORT"); envPort != "" {
		if p, err := strconv.Atoi(envPort); err == nil {
			*port = p
		}
	}

	// Environment variable: EASYOSS_S3_PORT
	if envS3Port := os.Getenv("EASYOSS_S3_PORT"); envS3Port != "" {
		if p, err := strconv.Atoi(envS3Port); err == nil {
			config.S3Port = p
		}
	}

	// Environment variable: EASYOSS_WEB_PORT
	if envWebPort := os.Getenv("EASYOSS_WEB_PORT"); envWebPort != "" {
		if p, err := strconv.Atoi(envWebPort); err == nil {
			config.WebPort = p
		}
	}

	// Environment variable: EASYOSS_DATA_PATH
	if envDataPath := os.Getenv("EASYOSS_DATA_PATH"); envDataPath != "" {
		config.DataPath = envDataPath
	}

	// Environment variable: EASYOSS_META_PATH
	if envMetaPath := os.Getenv("EASYOSS_META_PATH"); envMetaPath != "" {
		config.MetaPath = envMetaPath
	}

	// If combined port is specified (via -port flag), use it for both S3 and Web
	if *port > 0 {
		config.S3Port = *port
		config.WebPort = *port
	}

	// Create and start server
	srv, err := server.New(config)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Channel to receive errors from server goroutines
	errChan := make(chan error, 2)

	// Start server in goroutine
	go func() {
		if err := srv.Start(); err != nil {
			errChan <- err
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigChan:
		log.Printf("Received signal %v, shutting down...", sig)
	case err := <-errChan:
		log.Printf("Server error: %v", err)
	}

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}

	log.Println("Server stopped")
}

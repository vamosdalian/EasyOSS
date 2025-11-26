package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/vamosdalian/EasyOSS/internal/config"
	"github.com/vamosdalian/EasyOSS/internal/server"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	// Load configuration
	cfg, showVersion, err := config.LoadFromArgs()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	if showVersion {
		fmt.Printf("EasyOSS %s\n", version)
		fmt.Printf("Commit: %s\n", commit)
		fmt.Printf("Built: %s\n", date)
		return
	}

	// Create server config
	srvConfig := &server.Config{
		S3Port:   cfg.S3Port,
		WebPort:  cfg.WebPort,
		DataPath: cfg.DataPath,
		MetaPath: cfg.MetaPath,
	}

	// Create and start server
	srv, err := server.New(srvConfig)
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

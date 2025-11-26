package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/vamosdalian/EasyOSS/internal/s3"
	"github.com/vamosdalian/EasyOSS/internal/storage"
	"github.com/vamosdalian/EasyOSS/internal/web"
)

// Config holds server configuration
type Config struct {
	Port     int
	DataPath string
}

// Server represents the EasyOSS server
type Server struct {
	config     *Config
	httpServer *http.Server
	storage    *storage.LocalStorage
}

// New creates a new server instance
func New(config *Config) (*Server, error) {
	localStorage, err := storage.NewLocalStorage(config.DataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage: %w", err)
	}

	return &Server{
		config:  config,
		storage: localStorage,
	}, nil
}

// Start starts the server
func (s *Server) Start() error {
	// Create handlers
	s3Handler := s3.NewS3Handler(s.storage)
	webHandler, err := web.NewWebHandler(s.storage)
	if err != nil {
		return fmt.Errorf("failed to create web handler: %w", err)
	}

	// Create router
	mux := http.NewServeMux()

	// Web UI routes
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Check if this is an S3 API request (has S3 headers or not from browser)
		isS3Request := r.Header.Get("Authorization") != "" ||
			r.Header.Get("X-Amz-Date") != "" ||
			r.Header.Get("X-Amz-Content-Sha256") != ""

		// Route based on path
		if r.URL.Path == "/" {
			if isS3Request {
				// S3 ListBuckets request
				s3Handler.ServeHTTP(w, r)
				return
			}
			http.Redirect(w, r, "/web/", http.StatusTemporaryRedirect)
			return
		}

		if strings.HasPrefix(r.URL.Path, "/web") {
			webHandler.ServeHTTP(w, r)
			return
		}

		if strings.HasPrefix(r.URL.Path, "/static/") {
			webHandler.ServeHTTP(w, r)
			return
		}

		// S3 API routes
		s3Handler.ServeHTTP(w, r)
	})

	// Create HTTP server
	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.config.Port),
		Handler:      loggingMiddleware(mux),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("Starting EasyOSS server on port %d", s.config.Port)
	log.Printf("Web UI: http://localhost:%d/web/", s.config.Port)
	log.Printf("S3 API: http://localhost:%d/", s.config.Port)
	log.Printf("Data path: %s", s.config.DataPath)

	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}

// loggingMiddleware logs all requests
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		log.Printf("%s %s %d %s", r.Method, r.URL.Path, wrapped.statusCode, time.Since(start))
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

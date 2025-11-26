package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/vamosdalian/EasyOSS/internal/s3"
	"github.com/vamosdalian/EasyOSS/internal/storage"
	"github.com/vamosdalian/EasyOSS/internal/web"
)

// Config holds server configuration
type Config struct {
	S3Port   int
	WebPort  int
	DataPath string
	MetaPath string
}

// Server represents the EasyOSS server
type Server struct {
	config       *Config
	s3Server     *http.Server
	webServer    *http.Server
	storage      *storage.LocalStorage
	separatePorts bool
}

// New creates a new server instance
func New(config *Config) (*Server, error) {
	// Determine meta path
	metaPath := config.MetaPath
	if metaPath == "" {
		metaPath = config.DataPath + "/.meta"
	}

	localStorage, err := storage.NewLocalStorageWithMeta(config.DataPath, metaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage: %w", err)
	}

	return &Server{
		config:        config,
		storage:       localStorage,
		separatePorts: config.S3Port != config.WebPort,
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

	if s.separatePorts {
		return s.startSeparateServers(s3Handler, webHandler)
	}
	return s.startCombinedServer(s3Handler, webHandler)
}

// startCombinedServer starts both S3 and Web on the same port
func (s *Server) startCombinedServer(s3Handler *s3.S3Handler, webHandler *web.WebHandler) error {
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
	s.s3Server = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.config.S3Port),
		Handler:      loggingMiddleware(mux),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("Starting EasyOSS server on port %d (combined mode)", s.config.S3Port)
	log.Printf("Web UI: http://localhost:%d/web/", s.config.S3Port)
	log.Printf("S3 API: http://localhost:%d/", s.config.S3Port)
	log.Printf("Data path: %s", s.config.DataPath)
	if s.config.MetaPath != "" {
		log.Printf("Meta path: %s", s.config.MetaPath)
	}

	return s.s3Server.ListenAndServe()
}

// startSeparateServers starts S3 and Web on separate ports
func (s *Server) startSeparateServers(s3Handler *s3.S3Handler, webHandler *web.WebHandler) error {
	var wg sync.WaitGroup
	errChan := make(chan error, 2)

	// S3 API server
	s3Mux := http.NewServeMux()
	s3Mux.HandleFunc("/", s3Handler.ServeHTTP)

	s.s3Server = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.config.S3Port),
		Handler:      loggingMiddleware(s3Mux),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Web UI server
	webMux := http.NewServeMux()
	webMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.Redirect(w, r, "/web/", http.StatusTemporaryRedirect)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/web") || strings.HasPrefix(r.URL.Path, "/static/") {
			webHandler.ServeHTTP(w, r)
			return
		}
		http.NotFound(w, r)
	})

	s.webServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.config.WebPort),
		Handler:      loggingMiddleware(webMux),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("Starting EasyOSS server (separate ports mode)")
	log.Printf("S3 API: http://localhost:%d/", s.config.S3Port)
	log.Printf("Web UI: http://localhost:%d/web/", s.config.WebPort)
	log.Printf("Data path: %s", s.config.DataPath)
	if s.config.MetaPath != "" {
		log.Printf("Meta path: %s", s.config.MetaPath)
	}

	// Start S3 server
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := s.s3Server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("S3 server error: %w", err)
		}
	}()

	// Start Web server
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := s.webServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("Web server error: %w", err)
		}
	}()

	// Wait for first error or all servers to finish
	select {
	case err := <-errChan:
		return err
	}
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	var errs []error

	if s.s3Server != nil {
		if err := s.s3Server.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("s3 server shutdown: %w", err))
		}
	}

	if s.webServer != nil {
		if err := s.webServer.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("web server shutdown: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("shutdown errors: %v", errs)
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

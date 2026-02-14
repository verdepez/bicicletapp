// Package server provides HTTP server setup and handlers
package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"bicicletapp/internal/config"
	"bicicletapp/internal/repository"
	"bicicletapp/internal/templates"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Server represents the HTTP server
type Server struct {
	config    *config.Config
	repos     *repository.Repositories
	templates *templates.Manager
	router    *chi.Mux
	http      *http.Server
}

// New creates a new server instance
func New(cfg *config.Config, repos *repository.Repositories, tmpl *templates.Manager) *Server {
	s := &Server{
		config:    cfg,
		repos:     repos,
		templates: tmpl,
		router:    chi.NewRouter(),
	}

	s.setupMiddleware()
	s.setupRoutes()

	s.http = &http.Server{
		Addr:         cfg.Address(),
		Handler:      s.router,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return s
}

// Run starts the server and handles graceful shutdown
func (s *Server) Run() error {
	// Channel to listen for errors from the server
	serverErrors := make(chan error, 1)

	// Start the server in a goroutine
	go func() {
		log.Printf("🚀 Server starting on %s", s.config.Address())
		log.Printf("📁 Debug mode: %v", s.config.Debug)
		serverErrors <- s.http.ListenAndServe()
	}()

	// Channel to listen for OS signals
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Wait for either server error or shutdown signal
	select {
	case err := <-serverErrors:
		return fmt.Errorf("server error: %w", err)

	case sig := <-shutdown:
		log.Printf("⚠️ Received %v signal, shutting down...", sig)

		// Give outstanding requests a deadline for completion
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Attempt graceful shutdown
		if err := s.http.Shutdown(ctx); err != nil {
			log.Printf("❌ Graceful shutdown failed: %v", err)
			if err := s.http.Close(); err != nil {
				return fmt.Errorf("failed to close server: %w", err)
			}
		}

		log.Println("✅ Server shutdown complete")
	}

	return nil
}

// setupMiddleware configures global middleware
func (s *Server) setupMiddleware() {
	// Real IP detection (important for logging behind proxies)
	s.router.Use(middleware.RealIP)

	// Request logging
	s.router.Use(middleware.Logger)

	// Panic recovery
	s.router.Use(middleware.Recoverer)

	// Request ID for tracing
	s.router.Use(middleware.RequestID)

	// Security headers
	s.router.Use(s.securityHeaders)

	// Response compression (level 5 is a good balance)
	s.router.Use(middleware.Compress(5))

	// Timeout for requests
	s.router.Use(middleware.Timeout(30 * time.Second))
}

// securityHeaders adds security-related headers to all responses
func (s *Server) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Prevent MIME type sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// Prevent clickjacking
		w.Header().Set("X-Frame-Options", "DENY")

		// XSS protection (legacy but still useful)
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		// Control referrer information
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Content Security Policy
		// Relaxed to allow external images/videos for ads
		csp := "default-src 'self'; " +
			"style-src 'self' 'unsafe-inline'; " +
			"script-src 'self' 'unsafe-inline'; " +
			"img-src * data:; " +
			"media-src *; " +
			"font-src 'self'"
		w.Header().Set("Content-Security-Policy", csp)

		// Permissions Policy (restrict browser features)
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		next.ServeHTTP(w, r)
	})
}

// GetRouter returns the chi router (useful for testing)
func (s *Server) GetRouter() *chi.Mux {
	return s.router
}

package server

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ConnorsApps/clipboard/internal/config"
	"github.com/rs/zerolog/log"
)

// Server represents the HTTP server
type Server struct {
	cfg              *config.Config
	clipboardHandler http.HandlerFunc
	filesHandler     http.HandlerFunc
	filesListHandler http.HandlerFunc
	loginHandler     http.HandlerFunc
	validateToken    func(string) bool
	uploadHandler    http.Handler
}

// New creates a new Server instance
func New(
	cfg *config.Config,
	clipboardHandler http.HandlerFunc,
	filesHandler http.HandlerFunc,
	filesListHandler http.HandlerFunc,
	loginHandler http.HandlerFunc,
	validateToken func(string) bool,
	uploadHandler http.Handler,
) *Server {
	return &Server{
		cfg:              cfg,
		clipboardHandler: clipboardHandler,
		filesHandler:     filesHandler,
		filesListHandler: filesListHandler,
		loginHandler:     loginHandler,
		validateToken:    validateToken,
		uploadHandler:    uploadHandler,
	}
}

// Run starts the HTTP server with graceful shutdown
func (s *Server) Run() error {
	mux := http.NewServeMux()

	srv := &http.Server{
		Addr:    ":" + s.cfg.Port,
		Handler: mux,
	}

	// Graceful shutdown
	idleConnsClosed := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)
		<-sigint

		log.Info().Msg("Shutdown signal received, shutting down server...")

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			log.Error().Err(err).Msg("HTTP server shutdown error")
		}
		close(idleConnsClosed)
	}()

	// Check for frontend directory
	frontendDir := "./frontend/dist"
	if _, err := os.Stat(frontendDir); os.IsNotExist(err) {
		log.Warn().Str("dir", frontendDir).Msg("Frontend build directory not found, will serve API only")
	} else {
		// Serve static files with SPA fallback
		fs := http.FileServer(http.Dir(frontendDir))
		mux.Handle("/", spaHandler(fs, frontendDir))
	}

	// API endpoints
	mux.HandleFunc("/api/login", withCORS(s.loginHandler))
	mux.HandleFunc("/ws", s.clipboardHandler)

	mux.HandleFunc("/api/uploads/", withAuth(http.StripPrefix("/api/uploads/", s.uploadHandler).ServeHTTP, s.validateToken))
	mux.HandleFunc("/api/uploads", withAuth(http.StripPrefix("/api/uploads", s.uploadHandler).ServeHTTP, s.validateToken))

	// File listing endpoint
	mux.HandleFunc("GET /api/files", withCORS(withAuth(s.filesListHandler, s.validateToken)))

	// File download and delete endpoint
	mux.HandleFunc("/api/files/", withCORS(withAuth(s.filesHandler, s.validateToken)))

	log.Info().Str("port", s.cfg.Port).Msg("Server started")
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatal().Err(err).Msg("Server error")
	}

	<-idleConnsClosed
	log.Info().Msg("Server stopped")
	return nil
}

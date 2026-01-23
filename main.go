package main

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/ConnorsApps/clipboard/internal/auth"
	"github.com/ConnorsApps/clipboard/internal/clipboard"
	"github.com/ConnorsApps/clipboard/internal/config"
	"github.com/ConnorsApps/clipboard/internal/files"
	"github.com/ConnorsApps/clipboard/internal/server"
	"github.com/ConnorsApps/clipboard/internal/tokenstore"
	_ "github.com/joho/godotenv/autoload"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// Setup logger
	if strings.EqualFold("true", os.Getenv("IS_LOCAL")) {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}

	ctx := context.Background()

	// Load configuration
	cfg, err := config.Load(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load config")
	}

	// Initialize token store
	var tokenStore tokenstore.Store
	if cfg.MongoURI != "" {
		storeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		store, err := tokenstore.NewMongoStore(storeCtx, cfg.MongoURI)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to connect to MongoDB")
		}
		tokenStore = store
		log.Info().Msg("Using MongoDB token store")
	} else {
		tokenStore = tokenstore.NewMemoryStore()
		log.Info().Msg("Using in-memory token store")
	}

	// Ensure token store is closed on exit
	defer func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := tokenStore.Close(closeCtx); err != nil {
			log.Error().Err(err).Msg("Failed to close token store")
		}
	}()

	// Create files directory if it doesn't exist
	if err := os.MkdirAll(cfg.FilesDir, 0755); err != nil {
		log.Fatal().Err(err).Str("dir", cfg.FilesDir).Msg("Failed to create files directory")
	}
	log.Info().Str("dir", cfg.FilesDir).Msg("Files directory initialized")

	// Initialize services
	authSvc := auth.New(cfg.Password, tokenStore)
	clipboardSvc := clipboard.New()
	
	// Create files service with broadcast callback
	filesSvc := files.New(cfg.FilesDir, func(filesList []files.FileInfo) {
		clipboardSvc.BroadcastFilesList(filesList)
	})

	// Initialize tusd handler with broadcast callback
	tusHandler, err := files.NewTusdHandler(cfg.FilesDir, filesSvc.BroadcastFilesList)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create tusd handler")
	}

	// Create and run server
	srv := server.New(
		cfg,
		clipboardSvc.HandleWebSocket(authSvc.ValidateToken),
		filesSvc.HandleFile,
		filesSvc.ListFiles,
		authSvc.HandleLogin,
		authSvc.ValidateToken,
		tusHandler,
	)

	if err := srv.Run(); err != nil {
		log.Fatal().Err(err).Msg("Server error")
	}
}

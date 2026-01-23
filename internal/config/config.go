package config

import (
	"context"

	"github.com/sethvargo/go-envconfig"
)

// Config holds all application configuration
type Config struct {
	Password string `env:"CLIPBOARD_PASSWORD,default=1234"`
	FilesDir string `env:"FILES_DIR,default=./tmp-files"`
	MongoURI string `env:"MONGODB_URI"`
	IsLocal  bool   `env:"IS_LOCAL,default=false"`
	Port     string `env:"PORT,default=8080"`
}

// Load reads configuration from environment variables
func Load(ctx context.Context) (*Config, error) {
	var cfg Config
	if err := envconfig.Process(ctx, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

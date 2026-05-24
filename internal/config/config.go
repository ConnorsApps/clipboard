package config

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/sethvargo/go-envconfig"
)

// Config holds all application configuration
type Config struct {
	Passwords   []string `env:"CLIPBOARD_PASSWORDS"`
	FilesDir    string   `env:"FILES_DIR,default=./tmp-files"`
	MongoURI    string   `env:"MONGODB_URI"`
	IsLocal     bool     `env:"IS_LOCAL,default=false"`
	Port        string   `env:"PORT,default=8080"`
	TokenExpiry string   `env:"TOKEN_EXPIRY,default=30d"`
}

// Load reads configuration from environment variables
func Load(ctx context.Context) (*Config, error) {
	var cfg Config
	if err := envconfig.Process(ctx, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// ParseTokenExpiry converts a TOKEN_EXPIRY value ("never", "10d", "30d", etc.)
// into the number of seconds for a MongoDB TTL index, or nil for no expiry.
func ParseTokenExpiry(s string) (*int32, error) {
	if strings.EqualFold(s, "never") {
		return nil, nil
	}
	s = strings.ToLower(strings.TrimSpace(s))
	if !strings.HasSuffix(s, "d") {
		return nil, fmt.Errorf("TOKEN_EXPIRY must be a number of days (e.g. \"10d\") or \"never\", got %q", s)
	}
	days, err := strconv.Atoi(strings.TrimSuffix(s, "d"))
	if err != nil || days <= 0 {
		return nil, fmt.Errorf("TOKEN_EXPIRY must be a positive number of days (e.g. \"10d\") or \"never\", got %q", s)
	}
	secs := int32(days * 86400)
	return &secs, nil
}

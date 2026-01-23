package tokenstore

import (
	"context"
	"time"
)

// Token represents a stored auth token
type Token struct {
	Token     string    `bson:"token"`
	CreatedAt time.Time `bson:"created_at"`
}

// Store defines the interface for token storage backends
type Store interface {
	// Store saves a new token
	Store(ctx context.Context, token Token) error

	// Exists checks if a token exists
	Exists(ctx context.Context, token string) (bool, error)

	// Delete removes a token
	Delete(ctx context.Context, token string) error

	// Close cleans up any resources
	Close(ctx context.Context) error
}

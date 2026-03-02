package tokenstore

import (
	"context"
	"errors"
	"time"
)

// ErrNotFound is returned when a token is not in the store (invalid or expired).
// Callers can use errors.Is(err, ErrNotFound) to distinguish from transient store errors.
var ErrNotFound = errors.New("token not found")

// Token represents a stored auth token
type Token struct {
	Token     string    `bson:"token"`
	UserID    string    `bson:"user_id"`
	CreatedAt time.Time `bson:"created_at"`
}

// Store defines the interface for token storage backends
type Store interface {
	// Store saves a new token
	Store(ctx context.Context, token Token) error

	// Exists checks if a token exists
	Exists(ctx context.Context, token string) (bool, error)

	// GetUserID returns the user ID for a token. If the token is not found, returns ErrNotFound
	// so callers can distinguish "invalid/expired token" (401) from transient store errors (503).
	GetUserID(ctx context.Context, token string) (string, bool, error)

	// Delete removes a token
	Delete(ctx context.Context, token string) error

	// Close cleans up any resources
	Close(ctx context.Context) error
}

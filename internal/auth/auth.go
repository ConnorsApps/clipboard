package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/ConnorsApps/clipboard/internal/tokenstore"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// Service manages authentication operations
type Service struct {
	password   string
	tokenStore tokenstore.Store
}

// New creates a new auth service
func New(password string, tokenStore tokenstore.Store) *Service {
	return &Service{
		password:   password,
		tokenStore: tokenStore,
	}
}

// HandleLogin processes login requests
func (s *Service) HandleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Password != s.password {
		log.Warn().Msg("Invalid login attempt")
		http.Error(w, "Invalid password", http.StatusUnauthorized)
		return
	}

	// Generate token
	token := uuid.New().String()

	// Store token
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := s.tokenStore.Store(ctx, tokenstore.Token{
		Token:     token,
		CreatedAt: time.Now(),
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to store token")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Info().Msg("Successful login")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"token": token})
}

// ValidateToken checks if a token is valid
func (s *Service) ValidateToken(token string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	exists, err := s.tokenStore.Exists(ctx, token)
	if err != nil {
		log.Error().Err(err).Msg("Failed to validate token")
		return false
	}

	return exists
}

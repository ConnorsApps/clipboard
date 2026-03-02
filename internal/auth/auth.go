package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"github.com/ConnorsApps/clipboard/internal/tokenstore"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// Service manages authentication operations
type Service struct {
	passwords  []string
	tokenStore tokenstore.Store
}

// New creates a new auth service. passwords is the list of valid passwords (each maps to a distinct user).
func New(passwords []string, tokenStore tokenstore.Store) *Service {
	return &Service{
		passwords:  passwords,
		tokenStore: tokenStore,
	}
}

// userIDFromPassword derives a stable user ID from a password (SHA-256 hex, first 16 chars).
func userIDFromPassword(password string) string {
	h := sha256.Sum256([]byte(password))
	return hex.EncodeToString(h[:])[:16]
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

	var matched bool
	for _, p := range s.passwords {
		if req.Password == p {
			matched = true
			break
		}
	}
	if !matched {
		log.Warn().Msg("Invalid login attempt")
		http.Error(w, "Invalid password", http.StatusUnauthorized)
		return
	}

	userID := userIDFromPassword(req.Password)

	// Generate token
	token := uuid.New().String()

	// Store token
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := s.tokenStore.Store(ctx, tokenstore.Token{
		Token:     token,
		UserID:    userID,
		CreatedAt: time.Now(),
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to store token")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Info().Str("userID", userID).Msg("Successful login")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"token": token})
}

// GetUserID returns the user ID for a valid token, or ("", false) if invalid
func (s *Service) GetUserID(token string) (string, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	userID, ok, err := s.tokenStore.GetUserID(ctx, token)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get user ID for token")
		return "", false
	}
	return userID, ok
}

// ValidateToken checks if a token is valid
func (s *Service) ValidateToken(token string) bool {
	_, ok := s.GetUserID(token)
	return ok
}

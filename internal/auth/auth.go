package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/ConnorsApps/clipboard/internal/tokenstore"
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

// generateToken returns a cryptographically random token (32 bytes as hex).
func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
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

	// Generate token (32 random bytes as hex = 64 chars, no need for UUID format)
	token, err := generateToken()
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate token")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Store token
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = s.tokenStore.Store(ctx, tokenstore.Token{
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

// GetUserID returns the user ID for a valid token. If the token is invalid or not found,
// returns ("", false, nil) so the caller may respond with 401. If the token store has a
// transient error, returns ("", false, err) so the caller may respond with 503 and allow retry.
func (s *Service) GetUserID(token string) (string, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	userID, ok, err := s.tokenStore.GetUserID(ctx, token)
	if err != nil {
		if errors.Is(err, tokenstore.ErrNotFound) {
			return "", false, nil
		}
		log.Error().Err(err).Msg("Failed to get user ID for token")
		return "", false, err
	}
	return userID, ok, nil
}

// ValidateToken checks if a token is valid
func (s *Service) ValidateToken(token string) bool {
	_, ok, _ := s.GetUserID(token)
	return ok
}

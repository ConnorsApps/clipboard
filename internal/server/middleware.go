package server

import (
	"context"
	"net/http"
	"os"
	"strings"
)

type contextKey string

const userIDContextKey contextKey = "userID"

// UserIDFromContext returns the user ID from the request context, or ("", false) if not set
func UserIDFromContext(ctx context.Context) (string, bool) {
	v := ctx.Value(userIDContextKey)
	if v == nil {
		return "", false
	}
	s, ok := v.(string)
	return s, ok && s != ""
}

// withCORS wraps a handler with CORS headers
func withCORS(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "*")
		w.Header().Set("Access-Control-Allow-Headers", "*")

		// Handle preflight OPTIONS request
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		handler(w, r)
	}
}

// withAuth wraps a handler with token validation and injects userID into the request context
func withAuth(handler http.HandlerFunc, getUserID func(string) (string, bool)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for HEAD requests
		if r.Method == http.MethodHead || r.Method == http.MethodOptions {
			handler(w, r)
			return
		}

		// Check Authorization header first
		authHeader := r.Header.Get("Authorization")
		token := strings.TrimPrefix(authHeader, "Bearer ")

		// If no auth header, check query param (for downloads)
		if token == "" {
			token = r.URL.Query().Get("token")
		}

		userID, ok := getUserID(token)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		*r = *r.WithContext(context.WithValue(r.Context(), userIDContextKey, userID))
		handler(w, r)
	}
}

// spaHandler wraps a file server to handle SPA routing
func spaHandler(fs http.Handler, dir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Check if file exists
		if path != "/" {
			fullPath := dir + path
			if _, err := os.Stat(fullPath); os.IsNotExist(err) {
				// File doesn't exist, serve index.html for SPA routing
				http.ServeFile(w, r, dir+"/index.html")
				return
			}
		}

		fs.ServeHTTP(w, r)
	}
}

package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// tus resumes uploads with HEAD, so it must be authenticated
func TestWithAuthHead(t *testing.T) {
	handler := withAuth(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := UserIDFromContext(r.Context()); !ok {
			t.Error("user ID not set")
		}
	}, func(token string) (string, bool, error) { return "user", token == "good", nil })

	for token, want := range map[string]int{"good": http.StatusOK, "": http.StatusUnauthorized} {
		req := httptest.NewRequest(http.MethodHead, "/api/uploads/x", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		handler(rec, req)
		if rec.Code != want {
			t.Errorf("token %q: status %d, want %d", token, rec.Code, want)
		}
	}
}

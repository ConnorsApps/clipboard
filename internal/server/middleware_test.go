package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWithAuthHead(t *testing.T) {
	getUserID := func(token string) (string, bool, error) {
		return "user", token == "good", nil
	}
	var gotUser string
	handler := withAuth(func(w http.ResponseWriter, r *http.Request) {
		gotUser, _ = UserIDFromContext(r.Context())
	}, getUserID)

	req := httptest.NewRequest(http.MethodHead, "/api/uploads/abc", nil)
	req.Header.Set("Authorization", "Bearer good")
	rec := httptest.NewRecorder()
	handler(rec, req)
	if rec.Code != http.StatusOK || gotUser != "user" {
		t.Errorf("authorized HEAD: status %d, user %q; want 200 and user injected", rec.Code, gotUser)
	}

	rec = httptest.NewRecorder()
	handler(rec, httptest.NewRequest(http.MethodHead, "/api/uploads/abc", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("unauthenticated HEAD: status %d, want 401", rec.Code)
	}
}

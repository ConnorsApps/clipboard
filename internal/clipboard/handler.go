package clipboard

import (
	"encoding/json"
	"net/http"
)

type clipboardResponse struct {
	Content string `json:"content"`
}

type clipboardRequest struct {
	Content string `json:"content"`
}

func (m *Manager) HandleGetClipboard(getUserID func(*http.Request) (string, bool)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := getUserID(r)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		content := m.GetOrCreate(userID).GetContent()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(clipboardResponse{Content: content})
	}
}

func (m *Manager) HandleSetClipboard(getUserID func(*http.Request) (string, bool)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := getUserID(r)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var req clipboardRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		svc := m.GetOrCreate(userID)
		svc.SetContent(req.Content)
		svc.Broadcast(WSMessage{Type: "content", Content: req.Content}, nil)

		w.WriteHeader(http.StatusNoContent)
	}
}

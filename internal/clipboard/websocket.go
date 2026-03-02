package clipboard

import (
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for simplicity
	},
}

// HandleWebSocket creates a WebSocket handler function that routes to the clipboard service for the authenticated user
func (m *Manager) HandleWebSocket(getUserID func(string) (string, bool)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		userID, ok := getUserID(token)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Error().Err(err).Msg("WebSocket upgrade failed")
			return
		}
		defer conn.Close()

		s := m.GetOrCreate(userID)
		s.RegisterClient(conn)
		log.Info().Str("userID", userID).Msg("WebSocket client connected")

		// Send current clipboard content
		content := s.GetContent()
		if err := conn.WriteJSON(WSMessage{Type: "content", Content: content}); err != nil {
			log.Error().Err(err).Msg("Failed to send initial content")
		}

		// Handle incoming messages
		for {
			var msg WSMessage
			if err := conn.ReadJSON(&msg); err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Error().Err(err).Msg("WebSocket read error")
				}
				break
			}

			switch msg.Type {
			case Update:
				s.SetContent(msg.Content)
				log.Debug().Str("userID", userID).Int("length", len(msg.Content)).Msg("Clipboard updated")

				// Broadcast to all clients except sender
				s.Broadcast(WSMessage{Type: "content", Content: msg.Content}, conn)

			case Clear:
				s.ClearContent()
				log.Info().Str("userID", userID).Msg("Clipboard cleared")

				// Broadcast to all clients including sender
				s.Broadcast(WSMessage{Type: "content", Content: ""}, nil)
			}
		}

		// Unregister client
		s.UnregisterClient(conn)
		log.Info().Str("userID", userID).Msg("WebSocket client disconnected")
	}
}

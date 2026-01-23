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

// HandleWebSocket creates a WebSocket handler function
func (s *Service) HandleWebSocket(validateToken func(string) bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Validate token from query param
		token := r.URL.Query().Get("token")
		if !validateToken(token) {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Error().Err(err).Msg("WebSocket upgrade failed")
			return
		}
		defer conn.Close()

		// Register client
		s.RegisterClient(conn)
		log.Info().Msg("WebSocket client connected")

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
				log.Debug().Int("length", len(msg.Content)).Msg("Clipboard updated")

				// Broadcast to all clients except sender
				s.Broadcast(WSMessage{Type: "content", Content: msg.Content}, conn)

			case Clear:
				s.ClearContent()
				log.Info().Msg("Clipboard cleared")

				// Broadcast to all clients including sender
				s.Broadcast(WSMessage{Type: "content", Content: ""}, nil)
			}
		}

		// Unregister client
		s.UnregisterClient(conn)
		log.Info().Msg("WebSocket client disconnected")
	}
}

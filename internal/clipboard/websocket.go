package clipboard

import (
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

const (
	pingInterval = 30 * time.Second
	pongTimeout  = 10 * time.Second
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for simplicity
	},
}

// HandleWebSocket creates a WebSocket handler function that routes to the clipboard service for the authenticated user.
// Store errors return 503 so the client can retry without clearing the token.
func (m *Manager) HandleWebSocket(getUserID func(string) (string, bool, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Error().Err(err).Msg("WebSocket upgrade failed")
			return
		}
		defer conn.Close()

		token := r.URL.Query().Get("token")
		userID, ok, err := getUserID(token)
		if err != nil {
			conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseTryAgainLater, "service unavailable"))
			return
		}
		if !ok {
			conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(4001, "unauthorized"))
			return
		}

		conn.SetReadDeadline(time.Now().Add(pingInterval + pongTimeout))
		conn.SetPongHandler(func(string) error {
			conn.SetReadDeadline(time.Now().Add(pingInterval + pongTimeout))
			return nil
		})

		s := m.GetOrCreate(userID)
		s.RegisterClient(conn)
		log.Info().Str("userID", userID).Msg("WebSocket client connected")

		// Send current clipboard content
		content := s.GetContent()
		if err := conn.WriteJSON(WSMessage{Type: "content", Content: content}); err != nil {
			log.Error().Err(err).Msg("Failed to send initial content")
		}

		// Ping goroutine keeps the connection alive through proxies with short idle timeouts
		done := make(chan struct{})
		defer close(done)
		go func() {
			ticker := time.NewTicker(pingInterval)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					if err := conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(pongTimeout)); err != nil {
						return
					}
				case <-done:
					return
				}
			}
		}()

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

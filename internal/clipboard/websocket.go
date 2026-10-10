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
// filesForUser, if set, supplies the file list sent to each client right after it connects.
func (m *Manager) HandleWebSocket(getUserID func(string) (string, bool, error), filesForUser func(userID string) interface{}) http.HandlerFunc {
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

		// Send current clipboard content and file list
		var files func() interface{}
		if filesForUser != nil {
			files = func() interface{} { return filesForUser(userID) }
		}
		if err := s.SendInitialState(conn, files); err != nil {
			log.Error().Err(err).Msg("Failed to send initial state")
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
				s.UpdateContent(msg.Content)
				log.Debug().Str("userID", userID).Int("length", len(msg.Content)).Msg("Clipboard updated")

			case Clear:
				s.UpdateContent("")
				log.Info().Str("userID", userID).Msg("Clipboard cleared")

			case Ping:
				if err := s.Send(conn, WSMessage{Type: Pong}); err != nil {
					log.Debug().Err(err).Msg("Failed to send pong")
				}
			}
		}

		// Unregister client
		s.UnregisterClient(conn)
		log.Info().Str("userID", userID).Msg("WebSocket client disconnected")
	}
}

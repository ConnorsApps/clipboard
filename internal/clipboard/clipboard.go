package clipboard

import (
	"sync"

	"github.com/gorilla/websocket"
)

// WSMessageType represents the type of WebSocket message
type WSMessageType string

const (
	// Update indicates a clipboard content update
	Update WSMessageType = "update"
	// Clear indicates a clipboard clear action
	Clear WSMessageType = "clear"
	// FilesList indicates a files list update
	FilesList WSMessageType = "files_list"
)

// WSMessage represents a WebSocket message
type WSMessage struct {
	Type    WSMessageType `json:"type"`
	Content string        `json:"content,omitempty"`
	Files   interface{}   `json:"files,omitempty"`
}

// Service manages clipboard state and WebSocket connections
type Service struct {
	content   string
	mu        sync.RWMutex
	clients   map[*websocket.Conn]bool
	clientsMu sync.RWMutex
}

// New creates a new clipboard service
func New() *Service {
	return &Service{
		clients: make(map[*websocket.Conn]bool),
	}
}

// GetContent returns the current clipboard content
func (s *Service) GetContent() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.content
}

// SetContent updates the clipboard content
func (s *Service) SetContent(content string) {
	s.mu.Lock()
	s.content = content
	s.mu.Unlock()
}

// ClearContent clears the clipboard content
func (s *Service) ClearContent() {
	s.mu.Lock()
	s.content = ""
	s.mu.Unlock()
}

// RegisterClient adds a WebSocket client to the service
func (s *Service) RegisterClient(conn *websocket.Conn) {
	s.clientsMu.Lock()
	s.clients[conn] = true
	s.clientsMu.Unlock()
}

// UnregisterClient removes a WebSocket client from the service
func (s *Service) UnregisterClient(conn *websocket.Conn) {
	s.clientsMu.Lock()
	delete(s.clients, conn)
	s.clientsMu.Unlock()
}

// Broadcast sends a message to all connected clients except the excluded one
func (s *Service) Broadcast(msg WSMessage, exclude *websocket.Conn) {
	s.clientsMu.RLock()
	defer s.clientsMu.RUnlock()

	for client := range s.clients {
		if client == exclude {
			continue
		}
		if err := client.WriteJSON(msg); err != nil {
			// Log error but continue broadcasting to other clients
			continue
		}
	}
}

// BroadcastFilesList sends a files list update to all connected clients
func (s *Service) BroadcastFilesList(files interface{}) {
	s.clientsMu.RLock()
	defer s.clientsMu.RUnlock()

	msg := WSMessage{
		Type:  FilesList,
		Files: files,
	}

	for client := range s.clients {
		if err := client.WriteJSON(msg); err != nil {
			// Log error but continue broadcasting to other clients
			continue
		}
	}
}

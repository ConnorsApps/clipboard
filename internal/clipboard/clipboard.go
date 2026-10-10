package clipboard

import (
	"errors"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// writeWait bounds how long a single slow client can hold up a broadcast
const writeWait = 10 * time.Second

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

// Service manages clipboard state and WebSocket connections for a single user
type Service struct {
	content string
	mu      sync.RWMutex
	// clients maps each connection to its write lock: gorilla/websocket supports only
	// one concurrent writer, and broadcasts arrive from reader and tusd goroutines
	clients   map[*websocket.Conn]*sync.Mutex
	clientsMu sync.RWMutex
}

// New creates a new clipboard service
func New() *Service {
	return &Service{
		clients: make(map[*websocket.Conn]*sync.Mutex),
	}
}

// Manager holds per-user clipboard services
type Manager struct {
	services map[string]*Service
	mu       sync.RWMutex
}

// NewManager creates a new clipboard manager
func NewManager() *Manager {
	return &Manager{
		services: make(map[string]*Service),
	}
}

// GetOrCreate returns the clipboard service for the given user ID, creating it if needed
func (m *Manager) GetOrCreate(userID string) *Service {
	m.mu.RLock()
	s, ok := m.services[userID]
	m.mu.RUnlock()
	if ok {
		return s
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok = m.services[userID]; ok {
		return s
	}
	s = New()
	m.services[userID] = s
	return s
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
	s.clients[conn] = &sync.Mutex{}
	s.clientsMu.Unlock()
}

// UnregisterClient removes a WebSocket client from the service
func (s *Service) UnregisterClient(conn *websocket.Conn) {
	s.clientsMu.Lock()
	delete(s.clients, conn)
	s.clientsMu.Unlock()
}

// writeJSON writes msg to conn while holding that connection's write lock
func writeJSON(conn *websocket.Conn, writeMu *sync.Mutex, msg WSMessage) error {
	writeMu.Lock()
	defer writeMu.Unlock()
	conn.SetWriteDeadline(time.Now().Add(writeWait))
	return conn.WriteJSON(msg)
}

// Send writes a message to a single registered client
func (s *Service) Send(conn *websocket.Conn, msg WSMessage) error {
	s.clientsMu.RLock()
	writeMu, ok := s.clients[conn]
	s.clientsMu.RUnlock()
	if !ok {
		return errors.New("client not registered")
	}
	return writeJSON(conn, writeMu, msg)
}

// Broadcast sends a message to all connected clients except the excluded one
func (s *Service) Broadcast(msg WSMessage, exclude *websocket.Conn) {
	// Snapshot the clients so a slow write doesn't hold clientsMu and stall (un)registration
	s.clientsMu.RLock()
	targets := make(map[*websocket.Conn]*sync.Mutex, len(s.clients))
	for client, writeMu := range s.clients {
		if client != exclude {
			targets[client] = writeMu
		}
	}
	s.clientsMu.RUnlock()

	for client, writeMu := range targets {
		if err := writeJSON(client, writeMu, msg); err != nil {
			// A failed or timed-out write leaves the connection unusable. Closing it ends
			// its read loop, which unregisters it; keep broadcasting to everyone else.
			client.Close()
		}
	}
}

// BroadcastFilesList sends a files list update to all connected clients
func (s *Service) BroadcastFilesList(files interface{}) {
	s.Broadcast(WSMessage{Type: FilesList, Files: files}, nil)
}

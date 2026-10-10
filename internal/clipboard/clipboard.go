package clipboard

import (
	"errors"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// writeWait bounds how long one slow client can hold up a broadcast
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
	// Ping asks for a Pong, to detect dead connections
	Ping WSMessageType = "ping"
	// Pong answers a Ping
	Pong WSMessageType = "pong"
)

// WSMessage represents a WebSocket message
type WSMessage struct {
	Type    WSMessageType `json:"type"`
	Content string        `json:"content,omitempty"`
	// Version increases with each change, so clients can drop stale messages
	Version uint64      `json:"version,omitempty"`
	Files   interface{} `json:"files,omitempty"`
}

// Service manages clipboard state and WebSocket connections for a single user
type Service struct {
	content string
	version uint64
	mu      sync.RWMutex
	// Each connection's write lock: gorilla allows one concurrent writer
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

func (s *Service) snapshot() (string, uint64) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.content, s.version
}

// SetContent updates the clipboard content and returns its new version
func (s *Service) SetContent(content string) uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.content = content
	s.version++
	return s.version
}

// UpdateContent sets the content and sends it, versioned, to every client including the
// sender, so concurrent edits converge.
func (s *Service) UpdateContent(content string) {
	version := s.SetContent(content)
	s.Broadcast(WSMessage{Type: "content", Content: content, Version: version})
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

// write sends msg on conn; callers must hold the connection's write lock
func write(conn *websocket.Conn, msg WSMessage) error {
	conn.SetWriteDeadline(time.Now().Add(writeWait))
	return conn.WriteJSON(msg)
}

// writeJSON writes msg to conn while holding that connection's write lock
func writeJSON(conn *websocket.Conn, writeMu *sync.Mutex, msg WSMessage) error {
	writeMu.Lock()
	defer writeMu.Unlock()
	return write(conn, msg)
}

func (s *Service) writeLock(conn *websocket.Conn) (*sync.Mutex, error) {
	s.clientsMu.RLock()
	defer s.clientsMu.RUnlock()
	writeMu, ok := s.clients[conn]
	if !ok {
		return nil, errors.New("client not registered")
	}
	return writeMu, nil
}

// Send writes a message to a single registered client
func (s *Service) Send(conn *websocket.Conn, msg WSMessage) error {
	writeMu, err := s.writeLock(conn)
	if err != nil {
		return err
	}
	return writeJSON(conn, writeMu, msg)
}

// SendInitialState sends a new client the content and file list, read under its write lock
// so a racing broadcast lands after this snapshot, not before.
func (s *Service) SendInitialState(conn *websocket.Conn, files func() interface{}) error {
	writeMu, err := s.writeLock(conn)
	if err != nil {
		return err
	}
	writeMu.Lock()
	defer writeMu.Unlock()
	content, version := s.snapshot()
	if err := write(conn, WSMessage{Type: "content", Content: content, Version: version}); err != nil {
		return err
	}
	if files != nil {
		return write(conn, WSMessage{Type: FilesList, Files: files()})
	}
	return nil
}

// Broadcast sends a message to all connected clients
func (s *Service) Broadcast(msg WSMessage) {
	// Copy so slow writes don't hold clientsMu
	s.clientsMu.RLock()
	targets := make(map[*websocket.Conn]*sync.Mutex, len(s.clients))
	for client, writeMu := range s.clients {
		targets[client] = writeMu
	}
	s.clientsMu.RUnlock()

	for client, writeMu := range targets {
		if err := writeJSON(client, writeMu, msg); err != nil {
			// Unusable now; closing ends its read loop, which unregisters it
			client.Close()
		}
	}
}

// BroadcastFilesList sends a files list update to all connected clients
func (s *Service) BroadcastFilesList(files interface{}) {
	s.Broadcast(WSMessage{Type: FilesList, Files: files})
}

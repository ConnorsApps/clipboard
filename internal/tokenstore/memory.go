package tokenstore

import (
	"context"
	"sync"
)

// MemoryStore implements Store using an in-memory map
type MemoryStore struct {
	tokens map[string]Token
	mu     sync.RWMutex
}

// NewMemoryStore creates a new in-memory token store
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		tokens: make(map[string]Token),
	}
}

// Store saves a new token
func (m *MemoryStore) Store(ctx context.Context, token Token) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tokens[token.Token] = token
	return nil
}

// Exists checks if a token exists
func (m *MemoryStore) Exists(ctx context.Context, token string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, exists := m.tokens[token]
	return exists, nil
}

// Delete removes a token
func (m *MemoryStore) Delete(ctx context.Context, token string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.tokens, token)
	return nil
}

// Close is a no-op for in-memory store
func (m *MemoryStore) Close(ctx context.Context) error {
	return nil
}

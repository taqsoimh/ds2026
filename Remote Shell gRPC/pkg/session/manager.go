package session

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
)

// Manager manages multiple client sessions
type Manager struct {
	sessions    map[string]*Session
	clientIndex map[string]string // clientID -> sessionID
	maxSessions int
	mu          sync.RWMutex
}

// ManagerConfig holds configuration for the session manager
type ManagerConfig struct {
	MaxSessions int
}

// DefaultManagerConfig returns the default manager configuration
func DefaultManagerConfig() ManagerConfig {
	return ManagerConfig{
		MaxSessions: 100,
	}
}

// NewManager creates a new session manager
func NewManager(cfg ManagerConfig) *Manager {
	if cfg.MaxSessions <= 0 {
		cfg.MaxSessions = 100
	}
	return &Manager{
		sessions:    make(map[string]*Session),
		clientIndex: make(map[string]string),
		maxSessions: cfg.MaxSessions,
	}
}

// Create creates a new session for a client
func (m *Manager) Create(clientID string) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if client already has a session
	if existingID, exists := m.clientIndex[clientID]; exists {
		if session, ok := m.sessions[existingID]; ok {
			session.UpdateActivity()
			return session, nil
		}
		// Clean up stale index entry
		delete(m.clientIndex, clientID)
	}

	// Check max sessions
	if len(m.sessions) >= m.maxSessions {
		return nil, ErrMaxSessions
	}

	// Generate unique session ID
	sessionID, err := generateSessionID()
	if err != nil {
		return nil, err
	}

	// Create new session
	session, err := NewSession(sessionID, clientID)
	if err != nil {
		return nil, err
	}

	m.sessions[sessionID] = session
	m.clientIndex[clientID] = sessionID

	return session, nil
}

// Get retrieves a session by ID
func (m *Manager) Get(sessionID string) (*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return nil, ErrSessionNotFound
	}

	return session, nil
}

// GetByClientID retrieves a session by client ID
func (m *Manager) GetByClientID(clientID string) (*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sessionID, exists := m.clientIndex[clientID]
	if !exists {
		return nil, ErrSessionNotFound
	}

	session, exists := m.sessions[sessionID]
	if !exists {
		return nil, ErrSessionNotFound
	}

	return session, nil
}

// Delete removes a session
func (m *Manager) Delete(sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return ErrSessionNotFound
	}

	delete(m.clientIndex, session.ClientID)
	delete(m.sessions, sessionID)

	return nil
}

// List returns all active sessions
func (m *Manager) List() []*Session {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sessions := make([]*Session, 0, len(m.sessions))
	for _, session := range m.sessions {
		sessions = append(sessions, session)
	}
	return sessions
}

// Count returns the number of active sessions
func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.sessions)
}

// generateSessionID generates a unique session ID
func generateSessionID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

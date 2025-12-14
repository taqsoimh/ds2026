// Each client connection gets a unique session that maintains state like working directory and environment variables.
package session

import (
	"errors"
	"os"
	"sync"
	"time"

	"remote-shell-rpc/pkg/executor"
)

// Common errors
var (
	ErrSessionNotFound = errors.New("session not found")
	ErrSessionExists   = errors.New("session already exists")
	ErrMaxSessions     = errors.New("maximum sessions reached")
)

// Session represents a client shell session
type Session struct {
	ID           string
	ClientID     string
	Executor     *executor.Executor
	WorkingDir   string
	Environment  map[string]string
	CreatedAt    time.Time
	LastActivity time.Time
	mu           sync.RWMutex
}

// NewSession creates a new session with the given ID and client ID
func NewSession(id, clientID string) (*Session, error) {
	// Get current working directory
	wd, err := os.Getwd()
	if err != nil {
		wd = os.TempDir()
	}

	// Create executor with default config
	cfg := executor.DefaultConfig()
	cfg.WorkingDir = wd

	exec := executor.New(cfg)

	now := time.Now()
	return &Session{
		ID:           id,
		ClientID:     clientID,
		Executor:     exec,
		WorkingDir:   wd,
		Environment:  make(map[string]string),
		CreatedAt:    now,
		LastActivity: now,
	}, nil
}

// SetWorkingDir sets the working directory for the session
func (s *Session) SetWorkingDir(dir string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.WorkingDir = dir
	s.Executor.SetWorkingDir(dir)
	s.LastActivity = time.Now()
}

// GetWorkingDir returns the current working directory
func (s *Session) GetWorkingDir() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.WorkingDir
}

// SetEnv sets an environment variable for the session
func (s *Session) SetEnv(key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Environment[key] = value
	s.updateExecutorEnv()
	s.LastActivity = time.Now()
}

// GetEnv gets an environment variable from the session
func (s *Session) GetEnv(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, ok := s.Environment[key]
	return val, ok
}

// UpdateActivity updates the last activity timestamp
func (s *Session) UpdateActivity() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.LastActivity = time.Now()
}

// GetLastActivity returns the last activity timestamp
func (s *Session) GetLastActivity() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.LastActivity
}

// updateExecutorEnv updates the executor environment from the session environment
func (s *Session) updateExecutorEnv() {
	env := os.Environ()
	for k, v := range s.Environment {
		env = append(env, k+"="+v)
	}
	s.Executor.SetEnvironment(env)
}

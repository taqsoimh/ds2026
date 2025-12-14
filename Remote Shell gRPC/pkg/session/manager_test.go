package session
package session

import (
	"testing"
)

func TestManager_Create(t *testing.T) {
	m := NewManager(DefaultManagerConfig())

	session, err := m.Create("client1")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if session == nil {
		t.Fatal("Create() returned nil session")
	}

	if session.ClientID != "client1" {
		t.Errorf("Create() clientID = %s, want client1", session.ClientID)
	}

	if session.ID == "" {
		t.Error("Create() sessionID is empty")
	}
}

func TestManager_CreateDuplicate(t *testing.T) {
	m := NewManager(DefaultManagerConfig())

	session1, err := m.Create("client1")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Creating session for same client should return existing session
	session2, err := m.Create("client1")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if session1.ID != session2.ID {
		t.Errorf("Create() returned different sessions for same client")
	}
}

func TestManager_Get(t *testing.T) {
	m := NewManager(DefaultManagerConfig())

	session, _ := m.Create("client1")

	got, err := m.Get(session.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if got.ID != session.ID {
		t.Errorf("Get() sessionID = %s, want %s", got.ID, session.ID)
	}
}

func TestManager_GetNotFound(t *testing.T) {
	m := NewManager(DefaultManagerConfig())

	_, err := m.Get("nonexistent")
	if err != ErrSessionNotFound {
		t.Errorf("Get() error = %v, want %v", err, ErrSessionNotFound)
	}
}

func TestManager_GetByClientID(t *testing.T) {
	m := NewManager(DefaultManagerConfig())

	session, _ := m.Create("client1")

	got, err := m.GetByClientID("client1")
	if err != nil {
		t.Fatalf("GetByClientID() error = %v", err)
	}

	if got.ID != session.ID {
		t.Errorf("GetByClientID() sessionID = %s, want %s", got.ID, session.ID)
	}
}

func TestManager_Delete(t *testing.T) {
	m := NewManager(DefaultManagerConfig())

	session, _ := m.Create("client1")

	err := m.Delete(session.ID)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	_, err = m.Get(session.ID)
	if err != ErrSessionNotFound {
		t.Errorf("Get() after Delete() error = %v, want %v", err, ErrSessionNotFound)
	}
}

func TestManager_DeleteNotFound(t *testing.T) {
	m := NewManager(DefaultManagerConfig())

	err := m.Delete("nonexistent")
	if err != ErrSessionNotFound {
		t.Errorf("Delete() error = %v, want %v", err, ErrSessionNotFound)
	}
}

func TestManager_List(t *testing.T) {
	m := NewManager(DefaultManagerConfig())

	m.Create("client1")
	m.Create("client2")
	m.Create("client3")

	sessions := m.List()
	if len(sessions) != 3 {
		t.Errorf("List() count = %d, want 3", len(sessions))
	}
}

func TestManager_Count(t *testing.T) {
	m := NewManager(DefaultManagerConfig())

	m.Create("client1")
	m.Create("client2")

	if m.Count() != 2 {
		t.Errorf("Count() = %d, want 2", m.Count())
	}
}

func TestManager_MaxSessions(t *testing.T) {
	cfg := ManagerConfig{MaxSessions: 2}
	m := NewManager(cfg)

	m.Create("client1")
	m.Create("client2")

	_, err := m.Create("client3")
	if err != ErrMaxSessions {
		t.Errorf("Create() error = %v, want %v", err, ErrMaxSessions)
	}
}

func TestSession_SetWorkingDir(t *testing.T) {
	session, _ := NewSession("test-id", "client1")

	session.SetWorkingDir("/tmp")

	if session.GetWorkingDir() != "/tmp" {
		t.Errorf("GetWorkingDir() = %s, want /tmp", session.GetWorkingDir())
	}
}

func TestSession_Environment(t *testing.T) {
	session, _ := NewSession("test-id", "client1")

	session.SetEnv("MY_VAR", "my_value")

	val, ok := session.GetEnv("MY_VAR")
	if !ok {
		t.Error("GetEnv() ok = false, want true")
	}

	if val != "my_value" {
		t.Errorf("GetEnv() = %s, want my_value", val)
	}
}

package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jterrazz/universe/internal/mind"
)

// Session tracks a Claude Code session for a specific mind+universe pair.
type Session struct {
	SessionID  string    `json:"session_id"`
	UniverseID string    `json:"universe_id"`
	MindID     string    `json:"mind_id"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func sessionPath(mindID, universeID string) (string, error) {
	mindPath, err := mind.ResolvePath(mindID)
	if err != nil {
		return "", err
	}
	return filepath.Join(mindPath, "sessions", universeID+".json"), nil
}

// Load reads a session file for the given mind and universe.
// Returns nil, nil if the session file does not exist.
func Load(mindID, universeID string) (*Session, error) {
	path, err := sessionPath(mindID, universeID)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading session file: %w", err)
	}
	var s Session
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parsing session file: %w", err)
	}
	return &s, nil
}

// Save writes a session to disk, creating directories if needed.
func Save(s *Session) error {
	path, err := sessionPath(s.MindID, s.UniverseID)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating session directory: %w", err)
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling session: %w", err)
	}
	return os.WriteFile(path, data, 0o644)
}

// List returns all sessions for a given mind.
func List(mindID string) ([]Session, error) {
	mindPath, err := mind.ResolvePath(mindID)
	if err != nil {
		return nil, err
	}
	sessionsDir := filepath.Join(mindPath, "sessions")
	entries, err := os.ReadDir(sessionsDir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading sessions directory: %w", err)
	}
	var sessions []Session
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(sessionsDir, e.Name()))
		if err != nil {
			continue
		}
		var s Session
		if err := json.Unmarshal(data, &s); err != nil {
			continue
		}
		sessions = append(sessions, s)
	}
	return sessions, nil
}

// Delete removes a session file for the given mind and universe.
func Delete(mindID, universeID string) error {
	path, err := sessionPath(mindID, universeID)
	if err != nil {
		return err
	}
	if err := os.Remove(path); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return fmt.Errorf("deleting session file: %w", err)
	}
	return nil
}

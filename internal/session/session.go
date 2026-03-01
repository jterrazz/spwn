package session

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Session represents a persistent conversation session between an agent and a universe.
type Session struct {
	SessionID  string    `json:"session_id"`
	UniverseID string    `json:"universe_id"`
	AgentName  string    `json:"agent_name"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// GenerateID returns a deterministic session ID for an agent+universe pair.
// Format: first 16 hex chars of sha256(agentName:universeID).
func GenerateID(agentName, universeID string) string {
	h := sha256.Sum256([]byte(agentName + ":" + universeID))
	return hex.EncodeToString(h[:])[:16]
}

// Load reads a session file from the Mind's sessions/ directory.
// Returns (nil, nil) if no session exists yet.
func Load(mindPath, universeID string) (*Session, error) {
	path := filePath(mindPath, universeID)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading session: %w", err)
	}

	var s Session
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parsing session: %w", err)
	}
	return &s, nil
}

// Save writes a session file to the Mind's sessions/ directory.
func Save(mindPath string, s *Session) error {
	dir := filepath.Join(mindPath, "sessions")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating sessions directory: %w", err)
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling session: %w", err)
	}

	return os.WriteFile(filePath(mindPath, s.UniverseID), data, 0o644)
}

// Exists returns true if a session file exists for the given universe.
func Exists(mindPath, universeID string) bool {
	_, err := os.Stat(filePath(mindPath, universeID))
	return err == nil
}

func filePath(mindPath, universeID string) string {
	return filepath.Join(mindPath, "sessions", universeID+".json")
}

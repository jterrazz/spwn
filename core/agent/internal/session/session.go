package session

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Session tracks an agent's conversation state within a universe.
type Session struct {
	ID         string `json:"id"`
	AgentName  string `json:"agent_name"`
	UniverseID string `json:"universe_id"`
	Resumed    bool   `json:"resumed"`
}

// DeterministicID generates a UUID-formatted session ID from agent name and universe ID.
// Same agent+universe pair always produces the same session ID.
// Format: xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx (UUID v4 layout with deterministic bytes).
func DeterministicID(agentName, universeID string) string {
	h := sha256.Sum256([]byte(agentName + ":" + universeID))
	hex := hex.EncodeToString(h[:16])
	// Format as UUID: 8-4-4-4-12
	return hex[0:8] + "-" + hex[8:12] + "-4" + hex[13:16] + "-a" + hex[17:20] + "-" + hex[20:32]
}

// Load reads a session file from the Mind's sessions directory.
// Returns nil if no session file exists (first spawn).
func Load(mindPath, universeID string) (*Session, error) {
	path := filePath(mindPath, universeID)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read session: %w", err)
	}

	var s Session
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse session: %w", err)
	}
	return &s, nil
}

// Save writes a session file to the Mind's sessions directory.
func Save(mindPath string, s *Session) error {
	dir := filepath.Join(mindPath, "sessions")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create sessions dir: %w", err)
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}

	path := filePath(mindPath, s.UniverseID)
	return os.WriteFile(path, data, 0644)
}

func filePath(mindPath, universeID string) string {
	return filepath.Join(mindPath, "sessions", universeID+".json")
}

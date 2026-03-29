package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/jterrazz/spwn/core/foundation"
)

// ClawState represents the persistent state of the Claw daemon.
type ClawState struct {
	Active    bool      `json:"active"`
	StartedAt time.Time `json:"started_at,omitempty"`
	Universes []string  `json:"universe_ids"`
}

// LoadClawState reads the Claw state from ~/.spwn/claw/claw.json.
// Returns an empty state if the file does not exist.
func LoadClawState() (*ClawState, error) {
	path := foundation.ClawStatePath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &ClawState{}, nil
		}
		return nil, err
	}
	var s ClawState
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// SaveClawState writes the Claw state to ~/.spwn/claw/claw.json.
func SaveClawState(s *ClawState) error {
	path := foundation.ClawStatePath()
	// Ensure claw directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

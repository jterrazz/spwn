// Package runtimestate persists the small set of mutable, per-world
// runtime data that cannot live in Docker container labels (because
// labels are immutable post-create).
//
// Currently this is:
//   - the deployed-agent list (agents are added/removed after spawn)
//   - per-agent runtime session ids (chat continuity)
//
// Each world gets one JSON file at ~/.spwn/runtime/<world-id>.json.
// When a world container disappears the file becomes orphaned and is
// reaped lazily by GC, which the state store calls on every list.
//
// We never read this package without first proving the world exists in
// Docker. The labels-on-container approach makes Docker the source of
// truth for existence; this package only stores decoration.
package runtimestate

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"spwn.sh/core/foundation"
	"spwn.sh/core/universe/internal/models"
)

// File holds the on-disk shape for a single world's runtime state.
type File struct {
	Agents     []models.AgentRecord `json:"agents,omitempty"`
	SessionIDs map[string]string    `json:"session_ids,omitempty"`
}

// Store provides mutex-protected per-world JSON persistence. A single
// process-wide mutex is fine here: writes are infrequent and the file
// is small enough that contention is invisible.
type Store struct {
	dir string
	mu  sync.Mutex
}

// NewStore returns a Store rooted at ~/.spwn/runtime/, creating the
// directory on first use.
func NewStore() (*Store, error) {
	dir := filepath.Join(foundation.BaseDir(), "runtime")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create runtime dir: %w", err)
	}
	return &Store{dir: dir}, nil
}

// NewStoreAt returns a Store rooted at an explicit directory. Used by
// tests so they don't touch the real ~/.spwn.
func NewStoreAt(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create runtime dir: %w", err)
	}
	return &Store{dir: dir}, nil
}

// Load returns the runtime state for a world. Missing files yield an
// empty File without an error — a freshly-spawned world legitimately
// has no runtime data yet.
func (s *Store) Load(worldID string) (File, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loadLocked(worldID)
}

// Save replaces the runtime state for a world. Atomic via tmp+rename.
func (s *Store) Save(worldID string, f File) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveLocked(worldID, f)
}

// Delete removes a world's runtime state file. Missing files are not
// an error.
func (s *Store) Delete(worldID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.Remove(s.path(worldID)); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// SetSessionID stores a runtime session id for one agent. Creates the
// file if it doesn't exist yet.
func (s *Store) SetSessionID(worldID, agentName, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, err := s.loadLocked(worldID)
	if err != nil {
		return err
	}
	if f.SessionIDs == nil {
		f.SessionIDs = make(map[string]string)
	}
	f.SessionIDs[agentName] = sessionID
	return s.saveLocked(worldID, f)
}

// GetSessionID returns the runtime session id for an agent, or "" if
// none has been recorded.
func (s *Store) GetSessionID(worldID, agentName string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, _ := s.loadLocked(worldID)
	return f.SessionIDs[agentName]
}

// AddAgent appends an agent record to the world's deployed-agent list.
// Idempotent on AgentID.
func (s *Store) AddAgent(worldID string, agent models.AgentRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, err := s.loadLocked(worldID)
	if err != nil {
		return err
	}
	for i := range f.Agents {
		if f.Agents[i].AgentID == agent.AgentID {
			f.Agents[i] = agent
			return s.saveLocked(worldID, f)
		}
	}
	f.Agents = append(f.Agents, agent)
	return s.saveLocked(worldID, f)
}

// RemoveAgent drops the agent with the given AgentID from a world.
// Missing agents are not an error.
func (s *Store) RemoveAgent(worldID, agentID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, err := s.loadLocked(worldID)
	if err != nil {
		return err
	}
	out := f.Agents[:0]
	for _, a := range f.Agents {
		if a.AgentID != agentID {
			out = append(out, a)
		}
	}
	f.Agents = out
	return s.saveLocked(worldID, f)
}

// UpdateAgentStatus mutates one agent's status in place.
func (s *Store) UpdateAgentStatus(worldID, agentID string, status models.Status) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, err := s.loadLocked(worldID)
	if err != nil {
		return err
	}
	for i := range f.Agents {
		if f.Agents[i].AgentID == agentID {
			f.Agents[i].Status = status
			return s.saveLocked(worldID, f)
		}
	}
	// No-op if the agent isn't tracked yet — the status update is best-effort.
	return nil
}

// GC removes runtime state files for any world id that is not in the
// supplied liveIDs set. Called by the state store on every List() so
// the directory naturally stays in sync with Docker.
func (s *Store) GC(liveIDs []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	live := make(map[string]struct{}, len(liveIDs))
	for _, id := range liveIDs {
		live[id] = struct{}{}
	}
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if filepath.Ext(name) != ".json" {
			continue
		}
		id := name[:len(name)-len(".json")]
		if _, ok := live[id]; ok {
			continue
		}
		_ = os.Remove(filepath.Join(s.dir, name))
	}
	return nil
}

// ── internals ──────────────────────────────────────────────────────────

func (s *Store) path(worldID string) string {
	return filepath.Join(s.dir, worldID+".json")
}

func (s *Store) loadLocked(worldID string) (File, error) {
	data, err := os.ReadFile(s.path(worldID))
	if err != nil {
		if os.IsNotExist(err) {
			return File{}, nil
		}
		return File{}, err
	}
	if len(data) == 0 {
		return File{}, nil
	}
	var f File
	if err := json.Unmarshal(data, &f); err != nil {
		return File{}, fmt.Errorf("parse runtime state for %s: %w", worldID, err)
	}
	return f, nil
}

func (s *Store) saveLocked(worldID string, f File) error {
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path(worldID) + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path(worldID))
}

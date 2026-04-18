// Package runtimestate owns the world-facing store: enumerating live
// worlds from Docker labels and persisting the small set of mutable,
// per-world data that cannot live in labels (because labels are
// immutable post-create).
//
// The split this package encodes:
//
//   - Docker container LABELS are the canonical source of truth for
//     "does this world exist" and for creation-time metadata (id,
//     config, workspaces, creation timestamp, …). Enumeration goes
//     through the container backend, so `docker rm` or a crashed
//     container disappear from List() naturally — no state-file drift.
//
//   - Mutable per-world data lives as JSON under
//     ~/.spwn/world-states/<world-id>/runtime.json. This covers:
//       • the deployed-agent list (agents added/removed after spawn)
//       • per-agent runtime session ids (chat continuity)
//       • DisplayName — the editable human name that overrides the
//         creation-time label for rendering. Supports `spwn world
//         rename` without destroying + recreating the container.
//
// The same world-state directory also holds the world's manifest,
// physics.md, faculties.md, roster.md and shared/ scratchpad. Co-
// locating runtime.json there means everything per-world lives in
// one place that can be inspected, archived, or destroyed atomically.
//
// GC removes runtime.json from any world-state directory whose
// container is no longer present — the rest of the directory is left
// alone, because destroying a container should never destroy user-
// authored notes.
package runtimestate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"spwn.sh/packages/container/backend"
	"spwn.sh/packages/platform"
	"spwn.sh/packages/world/labels"
	"spwn.sh/packages/world/models"
)

// ErrNotFound is returned by Get when the world id has no matching
// container. Callers should compare with errors.Is.
var ErrNotFound = errors.New("not found")

// File holds the on-disk shape for a single world's runtime state.
type File struct {
	// Agents is the deployed-agent list, including agents added
	// after initial spawn via hot-deploy.
	Agents []models.AgentRecord `json:"agents,omitempty"`
	// SessionIDs is agent-name → runtime session id, used for chat
	// continuity across talks.
	SessionIDs map[string]string `json:"session_ids,omitempty"`
	// DisplayName overrides the container label's name for UI
	// rendering. Populated by `spwn world rename`. Empty means "use
	// whatever the label says".
	DisplayName string `json:"display_name,omitempty"`
}

// Store reads world state from Docker container labels and persists
// per-world mutable data under ~/.spwn/world-states/<id>/runtime.json.
//
// A Store without a backend is valid for callers that only exercise
// the mutable-data methods (SetSessionID, AddAgent, …). Enumeration
// methods (List, Get) require a backend and return an error when
// none is wired in.
type Store struct {
	dir     string
	backend backend.Backend

	mu sync.Mutex
}

// NewStore returns a production Store wired to the host Docker
// daemon and rooted at ~/.spwn/world-states/. The legacy
// ~/.spwn/state.json file (from pre-labels installs) is removed on
// first construction — keeping it around would invite confusion.
func NewStore() (*Store, error) {
	docker, err := backend.NewDocker()
	if err != nil {
		return nil, fmt.Errorf("docker backend: %w", err)
	}
	dir := filepath.Join(platform.LocalStateDir(), "world-states")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create world-states dir: %w", err)
	}
	s := &Store{dir: dir, backend: docker}
	s.evictLegacyStateFile()
	return s, nil
}

// NewStoreWith creates a Store from an explicit backend + root dir.
// Used by tests and by callers that already hold a backend (notably
// apps/api's test fixtures).
func NewStoreWith(b backend.Backend, dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create world-states dir: %w", err)
	}
	return &Store{dir: dir, backend: b}, nil
}

// NewStoreAt returns a Store rooted at dir with no backend. Suitable
// for tests that only exercise mutable-state methods. Calls to List
// and Get error with "no backend configured".
func NewStoreAt(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create world-states dir: %w", err)
	}
	return &Store{dir: dir}, nil
}

// evictLegacyStateFile removes the old ~/.spwn/state.json and the
// short-lived ~/.spwn/runtime/ tree (used by an earlier refactor) if
// either is present. Safe to call repeatedly.
func (s *Store) evictLegacyStateFile() {
	legacy := platform.LegacyStatePath()
	if legacy != "" {
		if _, err := os.Stat(legacy); err == nil {
			_ = os.Remove(legacy)
			_ = os.Remove(legacy + ".bak")
		}
	}
	if oldRuntime := filepath.Join(platform.BaseDir(), "runtime"); oldRuntime != "" {
		_ = os.RemoveAll(oldRuntime)
	}
}

// ── World enumeration (Docker-label-backed) ───────────────────────────

// testRunFilter returns the SPWN_TEST_LABEL value when set, empty
// otherwise. Used to scope enumeration to a single test run so
// parallel tests with identically-named worlds don't collide at
// routing time. In production the env var is unset and this is a
// no-op pass-through.
func testRunFilter() string { return os.Getenv(labels.TestRunEnv) }

func belongsToTestRun(containerLabels map[string]string) bool {
	scope := testRunFilter()
	if scope == "" {
		return true
	}
	return containerLabels[labels.TestRun] == scope
}

// List returns every world the daemon currently knows about.
// Hydrates each one with mutable runtime state and GCs orphaned
// runtime files in the same pass so the runtime directory stays in
// sync with Docker.
func (s *Store) List() ([]models.World, error) {
	if s.backend == nil {
		return nil, errors.New("runtimestate: no backend configured for List")
	}
	ctx := context.Background()
	containers, err := s.backend.ListContainersByLabel(ctx, labels.KindKey, labels.KindWorld)
	if err != nil {
		return nil, fmt.Errorf("list world containers: %w", err)
	}

	worlds := make([]models.World, 0, len(containers))
	liveIDs := make([]string, 0, len(containers))
	for _, c := range containers {
		if !belongsToTestRun(c.Labels) {
			continue
		}
		w, err := labels.ParseWorld(c.Labels)
		if err != nil {
			// Container tagged as a world but with unparseable
			// metadata is a developer bug, not a user-facing error.
			// Skip rather than failing the whole list.
			continue
		}
		w.ContainerID = c.ID
		w.Status = derivedStatus(c)
		s.hydrate(&w)
		worlds = append(worlds, w)
		liveIDs = append(liveIDs, w.ID)
	}

	// Best-effort GC. Failures here never block a list call.
	_ = s.GC(liveIDs)
	return worlds, nil
}

// Get returns a single world by id. Returns ErrNotFound when no
// container with the matching id label exists.
func (s *Store) Get(id string) (*models.World, error) {
	if s.backend == nil {
		return nil, errors.New("runtimestate: no backend configured for Get")
	}
	ctx := context.Background()
	containers, err := s.backend.ListContainersByLabel(ctx, labels.WorldID, id)
	if err != nil {
		return nil, err
	}
	for _, c := range containers {
		if !belongsToTestRun(c.Labels) {
			continue
		}
		w, err := labels.ParseWorld(c.Labels)
		if err != nil {
			continue
		}
		w.ContainerID = c.ID
		w.Status = derivedStatus(c)
		s.hydrate(&w)
		return &w, nil
	}
	return nil, fmt.Errorf("world %s %w", id, ErrNotFound)
}

// hydrate fills the mutable bits of a World from runtime state.
// Labels carry the creation-time agent list + creation-time name;
// runtimestate carries post-spawn add/remove, per-agent mutable
// fields (status, session ids), and the display-name override. The
// two must be merged so hot-deployed agents appear alongside
// originals without discarding either, and so `spwn world rename`
// surfaces in every subsequent List.
func (s *Store) hydrate(w *models.World) {
	rs, _ := s.Load(w.ID)
	if len(rs.SessionIDs) > 0 {
		w.SessionIDs = rs.SessionIDs
	}
	if rs.DisplayName != "" {
		w.Name = rs.DisplayName
	}
	if len(rs.Agents) == 0 {
		return
	}
	byID := make(map[string]models.AgentRecord, len(w.Agents))
	order := make([]string, 0, len(w.Agents))
	for _, a := range w.Agents {
		if _, seen := byID[a.AgentID]; !seen {
			order = append(order, a.AgentID)
		}
		byID[a.AgentID] = a
	}
	for _, a := range rs.Agents {
		if _, seen := byID[a.AgentID]; !seen {
			order = append(order, a.AgentID)
		}
		byID[a.AgentID] = a
	}
	merged := make([]models.AgentRecord, 0, len(order))
	for _, id := range order {
		merged = append(merged, byID[id])
	}
	w.Agents = merged
}

// derivedStatus maps a container's runtime state into the spwn
// Status vocabulary. Stopped containers are NOT removed from
// listings — they remain a world, just one the user can restart.
func derivedStatus(c backend.ContainerInfo) models.Status {
	if c.Running {
		return models.StatusRunning
	}
	switch c.Status {
	case "exited", "dead":
		return models.StatusStopped
	case "created":
		return models.StatusCreating
	}
	return models.StatusIdle
}

// ── Per-world mutable state ───────────────────────────────────────────

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
// an error. The rest of the world-state directory (manifest,
// physics, shared scratchpad…) is intentionally left alone —
// destroying a container should not destroy user-authored notes.
func (s *Store) Delete(worldID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.Remove(s.path(worldID)); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// SetSessionID stores a runtime session id for one agent.
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

// AddAgent appends an agent record to the world's deployed-agent
// list. Idempotent on AgentID.
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

// UpdateAgentStatus mutates one agent's status in place. No-op when
// the agent isn't tracked yet — status updates are best-effort.
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
	return nil
}

// SetDisplayName records an editable name for a world. The next
// Load/Get/List overrides the container-label name with this value
// when non-empty. Empty clears the override; the label-derived name
// wins again.
//
// Returns ErrNotFound when no container with this id is running —
// this is the one mutable-state method that checks liveness, because
// renaming a ghost world is a user mistake worth surfacing. When the
// Store has no backend configured, the liveness check is skipped
// (tests exercising the raw mutable path).
func (s *Store) SetDisplayName(worldID, name string) error {
	if s.backend != nil {
		if _, err := s.Get(worldID); err != nil {
			return err
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	f, err := s.loadLocked(worldID)
	if err != nil {
		return err
	}
	f.DisplayName = name
	return s.saveLocked(worldID, f)
}

// GC removes runtime.json from any world-state directory whose id is
// not in the supplied liveIDs set. Called by List() on every call so
// runtime data naturally stays in sync with Docker.
//
// Only runtime.json is removed — the world-state directory itself
// (manifest, physics, shared scratchpad…) is left alone. Use a
// dedicated `spwn world prune` flow for archival cleanup.
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
		if !e.IsDir() {
			continue
		}
		id := e.Name()
		if _, ok := live[id]; ok {
			continue
		}
		_ = os.Remove(filepath.Join(s.dir, id, "runtime.json"))
	}
	return nil
}

// ── internals ──────────────────────────────────────────────────────────

func (s *Store) path(worldID string) string {
	return filepath.Join(s.dir, worldID, "runtime.json")
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
	worldDir := filepath.Join(s.dir, worldID)
	if err := os.MkdirAll(worldDir, 0o755); err != nil {
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

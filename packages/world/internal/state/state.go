// Package state implements the world store. Despite the name, this
// package no longer keeps a JSON file of worlds — Docker container
// labels are the canonical source of truth, and the only mutable
// per-world data lives in the runtimestate package as small per-world
// JSON files. The Store type is a thin façade that preserves the
// historical API so callers don't have to change.
//
// What this gives us:
//
//   - Listing worlds is `docker ps --filter label=sh.spwn.kind=world`.
//     If the user runs `docker rm` (or the container dies for any
//     reason), the next List() call will not see it. Drift is not
//     possible.
//   - Per-world ephemeral data (deployed-agent list, runtime session
//     IDs) lives in ~/.spwn/runtime/<world-id>.json and is GC'd
//     against the live world set on every List(). Orphans don't
//     accumulate.
//   - Save() and Delete() on the world record itself become no-ops.
//     The labels were already written at container create time, and
//     the destroy code path removes the container — those are the
//     only events that change "does this world exist".
package state

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"spwn.sh/packages/foundation"
	"spwn.sh/packages/world/internal/backend"
	"spwn.sh/packages/world/internal/labels"
	"spwn.sh/packages/world/internal/models"
	"spwn.sh/packages/world/internal/runtimestate"
)

// Store reads world state from Docker container labels and per-world
// runtime files. The legacy state.json (if present from an older spwn
// install) is removed on first construction.
type Store struct {
	backend backend.Backend
	rstate  *runtimestate.Store
}

// NewStore returns a Docker-backed Store using the default backend and
// runtime directory. It also evicts the legacy ~/.spwn/state.json file
// if one is left over from an older spwn version — the file is no
// longer authoritative and keeping it around invites confusion.
func NewStore() (*Store, error) {
	docker, err := backend.NewDocker()
	if err != nil {
		return nil, fmt.Errorf("docker backend: %w", err)
	}
	rs, err := runtimestate.NewStore()
	if err != nil {
		return nil, fmt.Errorf("runtime state: %w", err)
	}
	s := &Store{backend: docker, rstate: rs}
	s.evictLegacyStateFile()
	return s, nil
}

// NewStoreWith creates a Store from explicit dependencies. Used by
// tests and by callers that already hold a Backend.
func NewStoreWith(b backend.Backend, rs *runtimestate.Store) *Store {
	return &Store{backend: b, rstate: rs}
}

// NewStoreAt is kept for backwards compatibility with old code that
// passed an explicit state file path. The path is now ignored — there
// is no state file. We log a one-time eviction if anything exists at
// the supplied path.
func NewStoreAt(path string) (*Store, error) {
	s, err := NewStore()
	if err != nil {
		return nil, err
	}
	if path != "" {
		if _, err := os.Stat(path); err == nil {
			_ = os.Remove(path)
		}
	}
	return s, nil
}

// evictLegacyStateFile removes the old ~/.spwn/state.json and the
// short-lived ~/.spwn/runtime/ tree (used by an earlier version of
// this refactor) if either is present. Safe to call repeatedly.
func (s *Store) evictLegacyStateFile() {
	legacy := foundation.StatePath()
	if legacy != "" {
		if _, err := os.Stat(legacy); err == nil {
			_ = os.Remove(legacy)
			_ = os.Remove(legacy + ".bak")
		}
	}
	// The ~/.spwn/runtime/ flat directory is replaced by per-world
	// subdirs under ~/.spwn/world-states/. Sweep it if it exists.
	if oldRuntime := filepath.Join(foundation.BaseDir(), "runtime"); oldRuntime != "" {
		_ = os.RemoveAll(oldRuntime)
	}
	// Make sure the new world-states root exists for first-time installs.
	_ = os.MkdirAll(filepath.Join(foundation.LocalStateDir(), "world-states"), 0o755)
}

// ── Read API ──────────────────────────────────────────────────────────

// List returns every world the daemon currently knows about. Hydrates
// each one with mutable runtime state and GCs orphaned runtime files
// in the same pass so the runtime directory stays in sync with Docker.
func (s *Store) List() ([]models.World, error) {
	ctx := context.Background()

	containers, err := s.backend.ListContainersByLabel(ctx, labels.KindKey, labels.KindWorld)
	if err != nil {
		return nil, fmt.Errorf("list world containers: %w", err)
	}

	worlds := make([]models.World, 0, len(containers))
	liveIDs := make([]string, 0, len(containers))
	for _, c := range containers {
		w, err := labels.ParseWorld(c.Labels)
		if err != nil {
			// A container with our kind label but unparseable metadata
			// is a developer bug, not a user-facing error. Skip it
			// rather than failing the whole list.
			continue
		}
		w.ContainerID = c.ID
		w.Status = derivedStatus(c)
		s.hydrate(&w)
		worlds = append(worlds, w)
		liveIDs = append(liveIDs, w.ID)
	}

	// Best-effort GC. Failures here never block a list call.
	_ = s.rstate.GC(liveIDs)

	return worlds, nil
}

// Get returns a single world by ID, or an error if no container with
// that label exists.
func (s *Store) Get(id string) (*models.World, error) {
	ctx := context.Background()
	containers, err := s.backend.ListContainersByLabel(ctx, labels.WorldID, id)
	if err != nil {
		return nil, err
	}
	for _, c := range containers {
		w, err := labels.ParseWorld(c.Labels)
		if err != nil {
			continue
		}
		w.ContainerID = c.ID
		w.Status = derivedStatus(c)
		s.hydrate(&w)
		return &w, nil
	}
	return nil, fmt.Errorf("world %s not found", id)
}

// hydrate fills the mutable bits of a World from runtime state. Labels
// carry the *creation-time* agent list; runtimestate carries post-spawn
// add/remove plus per-agent mutable fields (status, session ids). The
// two must be merged so hot-deployed agents appear alongside original
// ones without discarding either.
func (s *Store) hydrate(w *models.World) {
	rs, _ := s.rstate.Load(w.ID)
	if len(rs.SessionIDs) > 0 {
		w.SessionIDs = rs.SessionIDs
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

// derivedStatus maps a container's runtime state into the spwn Status
// vocabulary. Stopped containers are NOT removed from listings — they
// remain a world, just one that the user can restart.
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

// ── Write API (mostly no-ops, here for API stability) ─────────────────

// Save is a no-op. Worlds are persisted via Docker labels at container
// create time. Kept on the API so existing callers compile unchanged.
func (s *Store) Save(_ models.World) error { return nil }

// Delete drops any per-world runtime state. The container itself
// must be removed via the backend; this method only cleans up the
// runtimestate file. Safe to call for missing worlds.
func (s *Store) Delete(id string) error {
	return s.rstate.Delete(id)
}

// Rename used to mutate the world's display name in state.json. With
// labels as truth this is now a no-op (labels are immutable post-
// create). The historical UX of "renaming a world" required restarting
// the container; we choose to make it a no-op rather than silently
// destroy + recreate. Returns nil so callers don't break.
func (s *Store) Rename(id, name string) error {
	if _, err := s.Get(id); err != nil {
		return err
	}
	// TODO: support rename via container restart with new labels.
	return nil
}

// UpdateStatus is a no-op. Status is derived from container state.
func (s *Store) UpdateStatus(_ string, _ models.Status) error { return nil }

// ── Mutable per-world state — delegated to runtimestate ──────────────

// SetSessionID stores a runtime session id for an agent. Errors if the
// world does not exist.
func (s *Store) SetSessionID(worldID, agentName, sessionID string) error {
	if _, err := s.Get(worldID); err != nil {
		return err
	}
	return s.rstate.SetSessionID(worldID, agentName, sessionID)
}

// GetSessionID returns the runtime session id for an agent in a world.
// Returns "" if no session has been recorded or the world is gone.
func (s *Store) GetSessionID(worldID, agentName string) string {
	return s.rstate.GetSessionID(worldID, agentName)
}

// AddAgent adds or updates an agent record in a world's runtimestate.
func (s *Store) AddAgent(worldID string, agent models.AgentRecord) error {
	if _, err := s.Get(worldID); err != nil {
		return err
	}
	return s.rstate.AddAgent(worldID, agent)
}

// RemoveAgent drops an agent from a world's runtimestate.
func (s *Store) RemoveAgent(worldID, agentID string) error {
	if _, err := s.Get(worldID); err != nil {
		return err
	}
	return s.rstate.RemoveAgent(worldID, agentID)
}

// UpdateAgentStatus mutates an agent's status. No-ops if the agent is
// not tracked in runtimestate yet.
func (s *Store) UpdateAgentStatus(worldID, agentID string, status models.Status) error {
	if _, err := s.Get(worldID); err != nil {
		return err
	}
	return s.rstate.UpdateAgentStatus(worldID, agentID, status)
}

// ── Errors ────────────────────────────────────────────────────────────

// ErrNotFound is returned when a world id has no matching container.
var ErrNotFound = errors.New("world not found")

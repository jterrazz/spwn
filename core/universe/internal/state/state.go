package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"spwn.sh/core/universe/internal/models"
	"spwn.sh/core/foundation"
)

// Store provides mutex-protected JSON persistence for world state.
type Store struct {
	path string
	mu   sync.Mutex
}

// NewStore creates a Store at ~/.spwn/state.json, creating the directory if needed.
func NewStore() (*Store, error) {
	dir := foundation.BaseDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create base dir: %w", err)
	}
	return &Store{path: foundation.StatePath()}, nil
}

// NewStoreAt creates a Store at an explicit path, creating parent directories if needed.
func NewStoreAt(path string) (*Store, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create dir: %w", err)
	}
	return &Store{path: path}, nil
}

// List returns all worlds.
func (s *Store) List() ([]models.World, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.load()
}

// Get returns a world by ID.
func (s *Store) Get(id string) (*models.World, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	universes, err := s.load()
	if err != nil {
		return nil, err
	}
	for i := range universes {
		if universes[i].ID == id {
			return &universes[i], nil
		}
	}
	return nil, fmt.Errorf("world %s not found", id)
}

// Save adds or updates a world.
func (s *Store) Save(u models.World) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	universes, err := s.load()
	if err != nil {
		return err
	}

	found := false
	for i := range universes {
		if universes[i].ID == u.ID {
			universes[i] = u
			found = true
			break
		}
	}
	if !found {
		universes = append(universes, u)
	}
	return s.save(universes)
}

// Delete removes a world by ID.
func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	universes, err := s.load()
	if err != nil {
		return err
	}

	filtered := make([]models.World, 0, len(universes))
	for _, u := range universes {
		if u.ID != id {
			filtered = append(filtered, u)
		}
	}
	return s.save(filtered)
}

// Rename updates the display name of a world.
func (s *Store) Rename(id, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	universes, err := s.load()
	if err != nil {
		return err
	}
	for i := range universes {
		if universes[i].ID == id {
			universes[i].Name = name
			return s.save(universes)
		}
	}
	return fmt.Errorf("world %s not found", id)
}

// UpdateStatus changes the status of a world.
func (s *Store) UpdateStatus(id string, status models.Status) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	universes, err := s.load()
	if err != nil {
		return err
	}
	for i := range universes {
		if universes[i].ID == id {
			universes[i].Status = status
			return s.save(universes)
		}
	}
	return fmt.Errorf("world %s not found", id)
}

// AddAgent adds an agent record to a world.
func (s *Store) AddAgent(worldID string, agent models.AgentRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	universes, err := s.load()
	if err != nil {
		return err
	}
	for i := range universes {
		if universes[i].ID == worldID {
			universes[i].Agents = append(universes[i].Agents, agent)
			return s.save(universes)
		}
	}
	return fmt.Errorf("world %s not found", worldID)
}

// RemoveAgent removes an agent from a world.
func (s *Store) RemoveAgent(worldID, agentID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	universes, err := s.load()
	if err != nil {
		return err
	}
	for i := range universes {
		if universes[i].ID == worldID {
			filtered := make([]models.AgentRecord, 0, len(universes[i].Agents))
			for _, a := range universes[i].Agents {
				if a.AgentID != agentID {
					filtered = append(filtered, a)
				}
			}
			universes[i].Agents = filtered
			return s.save(universes)
		}
	}
	return fmt.Errorf("world %s not found", worldID)
}

// UpdateAgentStatus updates a specific agent's status within a world.
func (s *Store) UpdateAgentStatus(worldID, agentID string, status models.Status) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	universes, err := s.load()
	if err != nil {
		return err
	}
	for i := range universes {
		if universes[i].ID == worldID {
			for j := range universes[i].Agents {
				if universes[i].Agents[j].AgentID == agentID {
					universes[i].Agents[j].Status = status
					return s.save(universes)
				}
			}
			return fmt.Errorf("agent %s not found in world %s", agentID, worldID)
		}
	}
	return fmt.Errorf("world %s not found", worldID)
}

func (s *Store) load() ([]models.World, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if len(data) == 0 {
		return nil, nil
	}
	var universes []models.World
	if err := json.Unmarshal(data, &universes); err != nil {
		return nil, fmt.Errorf("parse state: %w", err)
	}
	// Migrate legacy single-workspace field into Workspaces slice.
	for i := range universes {
		if len(universes[i].Workspaces) == 0 && universes[i].Workspace != "" {
			universes[i].Workspaces = []models.Workspace{{Name: "default", Path: universes[i].Workspace}}
		}
		universes[i].Workspace = ""
	}
	return universes, nil
}

func (s *Store) save(universes []models.World) error {
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(universes, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0644)
}

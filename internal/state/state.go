package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/jterrazz/universe/internal/config"
)

// Store provides mutex-protected JSON persistence for universe state.
type Store struct {
	path string
	mu   sync.Mutex
}

// NewStore creates a Store at ~/.universe/state.json, creating the directory if needed.
func NewStore() (*Store, error) {
	dir := config.BaseDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create base dir: %w", err)
	}
	return &Store{path: config.StatePath()}, nil
}

// NewStoreAt creates a Store at an explicit path, creating parent directories if needed.
func NewStoreAt(path string) (*Store, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create dir: %w", err)
	}
	return &Store{path: path}, nil
}

// List returns all universes.
func (s *Store) List() ([]config.Universe, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.load()
}

// Get returns a universe by ID.
func (s *Store) Get(id string) (*config.Universe, error) {
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
	return nil, fmt.Errorf("universe %s not found", id)
}

// Save adds or updates a universe.
func (s *Store) Save(u config.Universe) error {
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

// Delete removes a universe by ID.
func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	universes, err := s.load()
	if err != nil {
		return err
	}

	filtered := make([]config.Universe, 0, len(universes))
	for _, u := range universes {
		if u.ID != id {
			filtered = append(filtered, u)
		}
	}
	return s.save(filtered)
}

// UpdateStatus changes the status of a universe.
func (s *Store) UpdateStatus(id string, status config.UniverseStatus) error {
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
	return fmt.Errorf("universe %s not found", id)
}

func (s *Store) load() ([]config.Universe, error) {
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
	var universes []config.Universe
	if err := json.Unmarshal(data, &universes); err != nil {
		return nil, fmt.Errorf("parse state: %w", err)
	}
	return universes, nil
}

func (s *Store) save(universes []config.Universe) error {
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

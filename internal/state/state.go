package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/jterrazz/universe/internal/config"
)

// Store manages universe state persistence in a JSON file.
type Store struct {
	path string
	mu   sync.Mutex
}

// NewStore creates a Store that persists to ~/.universe/universes.json.
func NewStore() (*Store, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("getting home directory: %w", err)
	}

	dir := filepath.Join(home, config.UniverseBaseDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("creating universe directory: %w", err)
	}

	return &Store{
		path: filepath.Join(dir, config.StateFileName),
	}, nil
}

// List returns all stored universes.
func (s *Store) List() ([]config.Universe, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.load()
}

// Get returns a universe by ID.
func (s *Store) Get(id string) (config.Universe, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	universes, err := s.load()
	if err != nil {
		return config.Universe{}, err
	}

	for _, u := range universes {
		if u.ID == id {
			return u, nil
		}
	}

	return config.Universe{}, fmt.Errorf("universe %s not found", id)
}

// Save adds or updates a universe in the store.
func (s *Store) Save(u config.Universe) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	universes, err := s.load()
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	// Update if exists, append if new.
	found := false
	for i, existing := range universes {
		if existing.ID == u.ID {
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

// Delete removes a universe from the store.
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

	for i, u := range universes {
		if u.ID == id {
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
		return nil, fmt.Errorf("reading state file: %w", err)
	}

	var universes []config.Universe
	if err := json.Unmarshal(data, &universes); err != nil {
		return nil, fmt.Errorf("parsing state file: %w", err)
	}

	return universes, nil
}

func (s *Store) save(universes []config.Universe) error {
	data, err := json.MarshalIndent(universes, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling state: %w", err)
	}

	return os.WriteFile(s.path, data, 0o644)
}

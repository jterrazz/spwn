//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"spwn.sh/core/universe"
	"spwn.sh/core/universe/tests/e2e/setup"
)

func TestConcurrent_SpawnThreeUniverses(t *testing.T) {
	// GIVEN three different universe configurations
	tc := setup.NewTestContext(t)

	var mu sync.Mutex
	var chains []*setup.AssertionChain
	var wg sync.WaitGroup

	configs := []string{
		`
physics:
  constants:
    cpu: 1
    memory: 256m
  tools:
    - "@unix"
`,
		`
physics:
  constants:
    cpu: 1
    memory: 512m
  tools:
    - "@unix"
`,
		`
physics:
  constants:
    cpu: 1
    memory: 384m
  tools:
    - "@unix"
`,
	}

	// WHEN spawning all three concurrently
	wg.Add(len(configs))
	for _, cfg := range configs {
		cfg := cfg // capture loop variable
		go func() {
			defer wg.Done()
			chain := tc.Spawn().
				WithConfigYAML(cfg).
				NoAgent().
				Execute()
			mu.Lock()
			chains = append(chains, chain)
			mu.Unlock()
		}()
	}
	wg.Wait()

	// THEN all three should be created successfully
	if len(chains) != 3 {
		t.Fatalf("Expected 3 chains, got %d", len(chains))
	}

	// AND state should track all three
	universes := tc.LoadState()
	if len(universes) != 3 {
		t.Fatalf("Expected 3 worlds in state, got %d", len(universes))
	}

	// AND all IDs should be unique
	ids := make(map[string]bool)
	for _, chain := range chains {
		id := chain.Universe().ID
		if ids[id] {
			t.Fatalf("Duplicate world ID: %s", id)
		}
		ids[id] = true
	}
}

func TestConcurrent_ListShowsAll(t *testing.T) {
	// GIVEN three sequentially spawned universes
	tc := setup.NewTestContext(t)

	chain1 := tc.Spawn().NoAgent().Execute()
	chain2 := tc.Spawn().NoAgent().Execute()
	chain3 := tc.Spawn().NoAgent().Execute()

	// WHEN listing all universes
	list, err := tc.Arc.List(context.Background())
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	// THEN all three should be present and idle
	if len(list) != 3 {
		t.Fatalf("Expected 3 worlds in list, got %d", len(list))
	}

	for _, u := range list {
		if u.Status != universe.StatusIdle {
			t.Fatalf("Expected status %q, got %q for universe %s", universe.StatusIdle, u.Status, u.ID)
		}
	}

	_ = chain1
	_ = chain2
	_ = chain3
}

func TestConcurrent_DestroyAllCleansState(t *testing.T) {
	// GIVEN three spawned universes
	tc := setup.NewTestContext(t)

	chains := make([]*setup.AssertionChain, 3)
	for i := 0; i < 3; i++ {
		chains[i] = tc.Spawn().NoAgent().Execute()
	}

	universes := tc.LoadState()
	if len(universes) != 3 {
		t.Fatalf("Expected 3 worlds before destroy, got %d", len(universes))
	}

	// WHEN all three are destroyed
	for _, chain := range chains {
		_, err := tc.Arc.Destroy(context.Background(), chain.Universe().ID)
		if err != nil {
			t.Fatalf("Destroy failed for %s: %v", chain.Universe().ID, err)
		}
	}

	// THEN the state should be empty
	universes = tc.LoadState()
	if len(universes) != 0 {
		t.Fatalf("Expected 0 worlds after destroy all, got %d", len(universes))
	}
}

func TestConcurrent_UniqueIDs(t *testing.T) {
	// GIVEN five sequentially spawned universes
	tc := setup.NewTestContext(t)

	const count = 5
	chains := make([]*setup.AssertionChain, count)
	for i := 0; i < count; i++ {
		chains[i] = tc.Spawn().NoAgent().Execute()
	}

	// THEN all IDs should be unique
	ids := make(map[string]bool)
	for _, chain := range chains {
		id := chain.Universe().ID
		if ids[id] {
			t.Fatalf("Duplicate world ID detected: %s", id)
		}
		ids[id] = true
	}

	if len(ids) != count {
		t.Fatalf("Expected %d unique IDs, got %d", count, len(ids))
	}
}

func TestConcurrent_FiveWorldsStateSafety(t *testing.T) {
	// GIVEN a fresh test context
	tc := setup.NewTestContext(t)

	const count = 5
	var mu sync.Mutex
	var chains []*setup.AssertionChain
	var wg sync.WaitGroup

	// WHEN spawning 5 worlds concurrently
	wg.Add(count)
	for i := 0; i < count; i++ {
		go func() {
			defer wg.Done()
			chain := tc.Spawn().NoAgent().Execute()
			mu.Lock()
			chains = append(chains, chain)
			mu.Unlock()
		}()
	}
	wg.Wait()

	// THEN all 5 should be created
	if len(chains) != count {
		t.Fatalf("Expected %d chains, got %d", count, len(chains))
	}

	// AND state.json should have exactly 5 entries
	universes := tc.LoadState()
	if len(universes) != count {
		t.Fatalf("Expected %d worlds in state, got %d", count, len(universes))
	}

	// AND state.json should be valid JSON (no corruption from concurrent writes)
	statePath := filepath.Join(tc.BaseDir, "state.json")
	data, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatalf("Failed to read state.json: %v", err)
	}
	if !json.Valid(data) {
		t.Fatalf("state.json is not valid JSON after concurrent spawns:\n%s", string(data))
	}

	// AND all IDs should be unique
	ids := make(map[string]bool)
	for _, chain := range chains {
		id := chain.Universe().ID
		if ids[id] {
			t.Fatalf("Duplicate world ID: %s", id)
		}
		ids[id] = true
	}

	// WHEN destroying all 5 concurrently
	wg.Add(count)
	for _, chain := range chains {
		chain := chain // capture
		go func() {
			defer wg.Done()
			_, err := tc.Arc.Destroy(context.Background(), chain.Universe().ID)
			if err != nil {
				mu.Lock()
				t.Errorf("Destroy failed for %s: %v", chain.Universe().ID, err)
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	// THEN state.json should be empty
	universes = tc.LoadState()
	if len(universes) != 0 {
		t.Fatalf("Expected 0 worlds after concurrent destroy, got %d", len(universes))
	}

	// AND state.json should still be valid JSON
	data, err = os.ReadFile(statePath)
	if err != nil {
		t.Fatalf("Failed to read state.json after destroy: %v", err)
	}
	if !json.Valid(data) {
		t.Fatalf("state.json is not valid JSON after concurrent destroys:\n%s", string(data))
	}
}

func TestConcurrent_ListDuringSpawn(t *testing.T) {
	// GIVEN a fresh test context
	tc := setup.NewTestContext(t)

	const count = 3
	var wg sync.WaitGroup

	// WHEN spawning worlds and listing concurrently
	wg.Add(count + 1)

	for i := 0; i < count; i++ {
		go func() {
			defer wg.Done()
			tc.Spawn().NoAgent().Execute()
		}()
	}

	// Concurrent list should not panic or return error
	var listErr error
	go func() {
		defer wg.Done()
		_, listErr = tc.Arc.List(context.Background())
	}()

	wg.Wait()

	if listErr != nil {
		t.Fatalf("Concurrent list failed: %v", listErr)
	}

	// Final state should have all worlds
	universes := tc.LoadState()
	if len(universes) != count {
		t.Fatalf("Expected %d worlds after concurrent spawn+list, got %d", count, len(universes))
	}
}

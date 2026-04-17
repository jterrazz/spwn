//go:build e2e

package e2e

import (
	"context"
	"sync"
	"testing"

	"spwn.sh/packages/world"
	"spwn.sh/packages/world/tests/e2e/setup"
)

func TestConcurrent_SpawnThreeWorlds(t *testing.T) {
	// Given - three different world configurations
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
dependencies:
  - "spwn:unix"
`,
		`
physics:
  constants:
    cpu: 1
    memory: 512m
dependencies:
  - "spwn:unix"
`,
		`
physics:
  constants:
    cpu: 1
    memory: 384m
dependencies:
  - "spwn:unix"
`,
	}

	// When - spawning all three concurrently
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

	// Then - all three should be created successfully
	if len(chains) != 3 {
		t.Fatalf("Expected 3 chains, got %d", len(chains))
	}

	// AND state should track all three
	worlds := tc.LoadState()
	if len(worlds) != 3 {
		t.Fatalf("Expected 3 worlds in state, got %d", len(worlds))
	}

	// AND all IDs should be unique
	ids := make(map[string]bool)
	for _, chain := range chains {
		id := chain.World().ID
		if ids[id] {
			t.Fatalf("Duplicate world ID: %s", id)
		}
		ids[id] = true
	}
}

func TestConcurrent_ListShowsAll(t *testing.T) {
	// Given - three sequentially spawned worlds
	tc := setup.NewTestContext(t)

	chain1 := tc.Spawn().NoAgent().Execute()
	chain2 := tc.Spawn().NoAgent().Execute()
	chain3 := tc.Spawn().NoAgent().Execute()

	// When - listing all worlds
	list, err := tc.Arc.List(context.Background())
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	// Then - all three should be present and idle
	if len(list) != 3 {
		t.Fatalf("Expected 3 worlds in list, got %d", len(list))
	}

	for _, u := range list {
		if u.Status != world.StatusRunning {
			t.Fatalf("Expected status %q, got %q for world %s", world.StatusRunning, u.Status, u.ID)
		}
	}

	_ = chain1
	_ = chain2
	_ = chain3
}

func TestConcurrent_DestroyAllCleansState(t *testing.T) {
	// Given - three spawned worlds
	tc := setup.NewTestContext(t)

	chains := make([]*setup.AssertionChain, 3)
	for i := 0; i < 3; i++ {
		chains[i] = tc.Spawn().NoAgent().Execute()
	}

	worlds := tc.LoadState()
	if len(worlds) != 3 {
		t.Fatalf("Expected 3 worlds before destroy, got %d", len(worlds))
	}

	// When - all three are destroyed
	for _, chain := range chains {
		_, err := tc.Arc.Destroy(context.Background(), chain.World().ID)
		if err != nil {
			t.Fatalf("Destroy failed for %s: %v", chain.World().ID, err)
		}
	}

	// Then - the state should be empty
	worlds = tc.LoadState()
	if len(worlds) != 0 {
		t.Fatalf("Expected 0 worlds after destroy all, got %d", len(worlds))
	}
}

func TestConcurrent_UniqueIDs(t *testing.T) {
	// Given - five sequentially spawned worlds
	tc := setup.NewTestContext(t)

	const count = 5
	chains := make([]*setup.AssertionChain, count)
	for i := 0; i < count; i++ {
		chains[i] = tc.Spawn().NoAgent().Execute()
	}

	// Then - all IDs should be unique
	ids := make(map[string]bool)
	for _, chain := range chains {
		id := chain.World().ID
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
	// Given - a fresh test context
	tc := setup.NewTestContext(t)

	const count = 5
	var mu sync.Mutex
	var chains []*setup.AssertionChain
	var wg sync.WaitGroup

	// When - spawning 5 worlds concurrently
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

	// Then - all 5 should be created
	if len(chains) != count {
		t.Fatalf("Expected %d chains, got %d", count, len(chains))
	}

	// AND state (container labels) should have exactly 5 entries
	worlds := tc.LoadState()
	if len(worlds) != count {
		t.Fatalf("Expected %d worlds in state, got %d", count, len(worlds))
	}

	// AND all IDs should be unique
	ids := make(map[string]bool)
	for _, chain := range chains {
		id := chain.World().ID
		if ids[id] {
			t.Fatalf("Duplicate world ID: %s", id)
		}
		ids[id] = true
	}

	// When - destroying all 5 concurrently
	wg.Add(count)
	for _, chain := range chains {
		chain := chain // capture
		go func() {
			defer wg.Done()
			_, err := tc.Arc.Destroy(context.Background(), chain.World().ID)
			if err != nil {
				mu.Lock()
				t.Errorf("Destroy failed for %s: %v", chain.World().ID, err)
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	// Then - state should be empty
	worlds = tc.LoadState()
	if len(worlds) != 0 {
		t.Fatalf("Expected 0 worlds after concurrent destroy, got %d", len(worlds))
	}
}

func TestConcurrent_ListDuringSpawn(t *testing.T) {
	// Given - a fresh test context
	tc := setup.NewTestContext(t)

	const count = 3
	var wg sync.WaitGroup

	// When - spawning worlds and listing concurrently
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
	worlds := tc.LoadState()
	if len(worlds) != count {
		t.Fatalf("Expected %d worlds after concurrent spawn+list, got %d", count, len(worlds))
	}
}

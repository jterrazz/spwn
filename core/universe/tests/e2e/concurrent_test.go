//go:build e2e

package e2e

import (
	"context"
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
  elements:
    - "@unix"
`,
		`
physics:
  constants:
    cpu: 1
    memory: 512m
  elements:
    - "@unix"
`,
		`
physics:
  constants:
    cpu: 1
    memory: 384m
  elements:
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
		t.Fatalf("Expected 3 universes in state, got %d", len(universes))
	}

	// AND all IDs should be unique
	ids := make(map[string]bool)
	for _, chain := range chains {
		id := chain.Universe().ID
		if ids[id] {
			t.Fatalf("Duplicate universe ID: %s", id)
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
		t.Fatalf("Expected 3 universes in list, got %d", len(list))
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
		t.Fatalf("Expected 3 universes before destroy, got %d", len(universes))
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
		t.Fatalf("Expected 0 universes after destroy all, got %d", len(universes))
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
			t.Fatalf("Duplicate universe ID detected: %s", id)
		}
		ids[id] = true
	}

	if len(ids) != count {
		t.Fatalf("Expected %d unique IDs, got %d", count, len(ids))
	}
}

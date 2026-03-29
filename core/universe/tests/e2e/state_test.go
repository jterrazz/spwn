//go:build e2e

package e2e

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jterrazz/spwn/core/universe"
	"github.com/jterrazz/spwn/core/universe/tests/e2e/setup"
)

func TestState_FileCreatedOnFirstSpawn(t *testing.T) {
	tc := setup.NewTestContext(t)

	statePath := filepath.Join(tc.BaseDir, "state.json")

	// State file should exist (created by NewTestContext via NewStoreAt)
	if _, err := os.Stat(statePath); err != nil {
		t.Fatalf("State file should exist after test context creation: %v", err)
	}

	// Spawn a universe
	tc.Spawn().
		NoAgent().
		Execute().
		ExpectState(func(s *setup.StateAssertion) {
			s.UniverseCount(1)
		})

	// State file should still exist and be non-empty
	info, err := os.Stat(statePath)
	if err != nil {
		t.Fatalf("State file missing after spawn: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("State file should be non-empty after spawn")
	}
}

func TestState_UpdatedOnDestroy(t *testing.T) {
	tc := setup.NewTestContext(t)

	chain := tc.Spawn().
		NoAgent().
		Execute()

	// Should have 1 universe
	chain.ExpectState(func(s *setup.StateAssertion) {
		s.UniverseCount(1)
	})

	// Destroy removes it from state
	chain.Destroy().
		ExpectState(func(s *setup.StateAssertion) {
			s.UniverseCount(0)
		})
}

func TestState_MultipleUniversesTracked(t *testing.T) {
	tc := setup.NewTestContext(t)

	tc.Spawn().NoAgent().Execute()
	tc.Spawn().NoAgent().Execute()
	tc.Spawn().NoAgent().Execute()

	universes := tc.LoadState()
	if len(universes) != 3 {
		t.Fatalf("Expected 3 universes in state, got %d", len(universes))
	}

	// Each should have a unique ID
	ids := make(map[string]bool)
	for _, u := range universes {
		if ids[u.ID] {
			t.Fatalf("Duplicate ID in state: %s", u.ID)
		}
		ids[u.ID] = true
	}
}

func TestState_StatusIdleAfterSpawnWithoutAgent(t *testing.T) {
	setup.NewSpawnBuilder(t).
		NoAgent().
		Execute().
		ExpectState(func(s *setup.StateAssertion) {
			s.UniverseCount(1)
			s.UniverseStatus(universe.StatusIdle)
		})
}

func TestState_StatusRunningWithDetachedAgent(t *testing.T) {
	tc := setup.NewTestContext(t)
	tc.InitAgent("state-running-agent")

	chain := tc.Spawn().
		WithAgent("state-running-agent").
		Detached().
		Execute()

	universes := tc.LoadState()
	found := false
	for _, u := range universes {
		if u.ID == chain.Universe().ID {
			found = true
			if u.Status != universe.StatusRunning {
				t.Fatalf("Expected status %q after detached spawn, got %q", universe.StatusRunning, u.Status)
			}
		}
	}
	if !found {
		t.Fatal("Universe not found in state after detached spawn")
	}
}

func TestState_StatusIdleAfterAgentCompletes(t *testing.T) {
	tc := setup.NewTestContext(t)
	tc.InitAgent("state-complete-agent")

	chain := tc.Spawn().
		WithAgent("state-complete-agent").
		RunAgent().
		Execute()

	universes := tc.LoadState()
	found := false
	for _, u := range universes {
		if u.ID == chain.Universe().ID {
			found = true
			if u.Status != universe.StatusIdle {
				t.Fatalf("Expected status %q after agent completion, got %q", universe.StatusIdle, u.Status)
			}
		}
	}
	if !found {
		t.Fatal("Universe not found in state after agent completion")
	}
}

func TestState_PartialDestroyLeavesOthers(t *testing.T) {
	tc := setup.NewTestContext(t)

	chain1 := tc.Spawn().NoAgent().Execute()
	chain2 := tc.Spawn().NoAgent().Execute()
	chain3 := tc.Spawn().NoAgent().Execute()

	// Destroy only the second one
	_, err := tc.Arc.Destroy(context.Background(), chain2.Universe().ID)
	if err != nil {
		t.Fatalf("Destroy failed: %v", err)
	}

	universes := tc.LoadState()
	if len(universes) != 2 {
		t.Fatalf("Expected 2 universes after partial destroy, got %d", len(universes))
	}

	// Remaining should be chain1 and chain3
	remainingIDs := make(map[string]bool)
	for _, u := range universes {
		remainingIDs[u.ID] = true
	}
	if !remainingIDs[chain1.Universe().ID] {
		t.Fatal("Expected universe 1 to still exist")
	}
	if !remainingIDs[chain3.Universe().ID] {
		t.Fatal("Expected universe 3 to still exist")
	}
	if remainingIDs[chain2.Universe().ID] {
		t.Fatal("Expected universe 2 to be removed")
	}
}

func TestState_AgentNameTracked(t *testing.T) {
	tc := setup.NewTestContext(t)
	tc.InitAgent("tracked-agent")

	tc.Spawn().
		WithAgent("tracked-agent").
		Execute().
		ExpectState(func(s *setup.StateAssertion) {
			s.UniverseCount(1)
			s.HasAgent("tracked-agent")
		})
}

func TestState_NoAgentNameWhenSpawnedWithoutAgent(t *testing.T) {
	setup.NewSpawnBuilder(t).
		NoAgent().
		Execute().
		ExpectState(func(s *setup.StateAssertion) {
			s.UniverseCount(1)
			s.HasNoAgent()
		})
}

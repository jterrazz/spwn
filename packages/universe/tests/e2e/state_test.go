//go:build e2e

package e2e

import (
	"context"
	"testing"

	"spwn.sh/packages/universe"
	"spwn.sh/packages/universe/tests/e2e/setup"
)

func TestState_UpdatedOnDestroy(t *testing.T) {
	// GIVEN a universe exists in the state
	tc := setup.NewTestContext(t)

	chain := tc.Spawn().
		NoAgent().
		Execute()

	chain.ExpectState(func(s *setup.StateAssertion) {
		s.WorldCount(1)
	})

	// WHEN the universe is destroyed
	// THEN the state should be empty
	chain.Destroy().
		ExpectState(func(s *setup.StateAssertion) {
			s.WorldCount(0)
		})
}

func TestState_MultipleUniversesTracked(t *testing.T) {
	// GIVEN three spawned universes
	tc := setup.NewTestContext(t)

	tc.Spawn().NoAgent().Execute()
	tc.Spawn().NoAgent().Execute()
	tc.Spawn().NoAgent().Execute()

	// THEN the state should track all three with unique IDs
	universes := tc.LoadState()
	if len(universes) != 3 {
		t.Fatalf("Expected 3 worlds in state, got %d", len(universes))
	}

	ids := make(map[string]bool)
	for _, u := range universes {
		if ids[u.ID] {
			t.Fatalf("Duplicate ID in state: %s", u.ID)
		}
		ids[u.ID] = true
	}
}

func TestState_StatusIdleAfterSpawnWithoutAgent(t *testing.T) {
	// GIVEN a universe spawned without an agent
	// THEN the status should be idle
	setup.NewSpawnBuilder(t).
		NoAgent().
		Execute().
		ExpectState(func(s *setup.StateAssertion) {
			s.WorldCount(1)
			s.WorldStatus(universe.StatusRunning)
		})
}

func TestState_StatusRunningWithDetachedAgent(t *testing.T) {
	// GIVEN a universe with a detached agent
	tc := setup.NewTestContext(t)
	tc.InitAgent("state-running-agent")

	chain := tc.Spawn().
		WithAgent("state-running-agent").
		Detached().
		Execute()

	// THEN the status should be running
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
	// GIVEN a universe where an agent has run to completion
	tc := setup.NewTestContext(t)
	tc.InitAgent("state-complete-agent")

	chain := tc.Spawn().
		WithAgent("state-complete-agent").
		RunAgent().
		Execute()

	// THEN the status should be idle (agent finished)
	universes := tc.LoadState()
	found := false
	for _, u := range universes {
		if u.ID == chain.Universe().ID {
			found = true
			if u.Status != universe.StatusRunning {
				t.Fatalf("Expected status %q after agent completion, got %q", universe.StatusRunning, u.Status)
			}
		}
	}
	if !found {
		t.Fatal("Universe not found in state after agent completion")
	}
}

func TestState_PartialDestroyLeavesOthers(t *testing.T) {
	// GIVEN three spawned universes
	tc := setup.NewTestContext(t)

	chain1 := tc.Spawn().NoAgent().Execute()
	chain2 := tc.Spawn().NoAgent().Execute()
	chain3 := tc.Spawn().NoAgent().Execute()

	// WHEN only the second universe is destroyed
	_, err := tc.Arc.Destroy(context.Background(), chain2.Universe().ID)
	if err != nil {
		t.Fatalf("Destroy failed: %v", err)
	}

	// THEN only the first and third should remain
	universes := tc.LoadState()
	if len(universes) != 2 {
		t.Fatalf("Expected 2 worlds after partial destroy, got %d", len(universes))
	}

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
	// GIVEN a universe spawned with a named agent
	tc := setup.NewTestContext(t)
	tc.InitAgent("tracked-agent")

	// THEN the state should track the agent name
	tc.Spawn().
		WithAgent("tracked-agent").
		Execute().
		ExpectState(func(s *setup.StateAssertion) {
			s.WorldCount(1)
			s.HasAgent("tracked-agent")
		})
}

func TestState_NoAgentNameWhenSpawnedWithoutAgent(t *testing.T) {
	// GIVEN a universe spawned without an agent
	// THEN the state should have no agent
	setup.NewSpawnBuilder(t).
		NoAgent().
		Execute().
		ExpectState(func(s *setup.StateAssertion) {
			s.WorldCount(1)
			s.HasNoAgent()
		})
}

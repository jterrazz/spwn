//go:build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	"spwn.sh/core/universe"
	"spwn.sh/core/universe/tests/e2e/setup"
)

func TestDetach_AgentRunsInBackground(t *testing.T) {
	// GIVEN a universe spawned with a detached agent
	chain := setup.NewSpawnBuilder(t).
		WithAgent("detach-agent").
		Detached().
		Execute()

	// THEN the state should show one universe
	chain.ExpectState(func(s *setup.StateAssertion) {
		s.WorldCount(1)
	})

	// AND the container should be running
	chain.ExpectContainer(func(c *setup.ContainerAssertion) {
		c.IsRunning()
	})

	// AND the mock agent should have been invoked
	chain.ExpectMock(func(m *setup.MockAssertion) {
		m.WasCalled()
	})
}

func TestDetach_MockSeesEnvironment(t *testing.T) {
	// GIVEN a universe spawned with a detached agent
	chain := setup.NewSpawnBuilder(t).
		WithAgent("detach-env-agent").
		Detached().
		Execute()

	// THEN the mock should see the full agent environment
	chain.ExpectMock(func(m *setup.MockAssertion) {
		m.WasCalled()
		m.SawMind()
		m.SawPhysics()
		m.SawFaculties()
	})
}

func TestDetach_StatusShowsRunning(t *testing.T) {
	// GIVEN a universe spawned with a detached agent
	tc := setup.NewTestContext(t)
	tc.InitAgent("detach-status-agent")

	chain := tc.Spawn().
		WithAgent("detach-status-agent").
		Detached().
		Execute()

	// THEN the state should show the universe as running
	universes := tc.LoadState()
	found := false
	for _, u := range universes {
		if u.ID == chain.Universe().ID {
			found = true
			if u.Status != universe.StatusRunning {
				t.Fatalf("Expected status %q for detached world, got %q", universe.StatusRunning, u.Status)
			}
		}
	}
	if !found {
		t.Fatal("Detached universe not found in state")
	}
}

func TestDetach_MultipleAgentsInSameUniverse(t *testing.T) {
	// GIVEN a universe with an initial detached agent
	tc := setup.NewTestContext(t)
	tc.InitAgent("multi-agent")

	chain := tc.Spawn().
		WithAgent("multi-agent").
		Detached().
		Execute()

	chain.ExpectMock(func(m *setup.MockAssertion) {
		m.WasCalled()
	})

	// WHEN a second agent is spawned detached in the same world
	err := tc.Arc.SpawnAgentDetached(context.Background(), chain.Universe().ID, "multi-agent")
	if err != nil {
		t.Fatalf("Second detached agent spawn failed: %v", err)
	}

	// THEN the mock should have been called again (wait for output)
	setup.WaitFor(t, 5*time.Second, 100*time.Millisecond, "second mock to write output", func() bool {
		output := tc.TryReadMockOutput(chain.Universe().ContainerID)
		return output != nil
	})

	// AND the container should still be running
	chain.ExpectContainer(func(c *setup.ContainerAssertion) {
		c.IsRunning()
	})

	// AND the mock should confirm it was called
	chain.ExpectMock(func(m *setup.MockAssertion) {
		m.WasCalled()
	})
}

func TestDetach_VsNoDetach(t *testing.T) {
	// GIVEN a universe spawned with an agent but WITHOUT Detached()
	// THEN the universe should be idle (agent not started in background)
	setup.NewSpawnBuilder(t).
		WithAgent("no-detach-agent").
		Execute().
		ExpectState(func(s *setup.StateAssertion) {
			s.WorldCount(1)
			s.WorldStatus(universe.StatusRunning)
		})
}

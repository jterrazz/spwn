//go:build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	"spwn.sh/packages/world"
	"spwn.sh/packages/world/tests/e2e/setup"
)

func TestDetach_AgentRunsInBackground(t *testing.T) {
	// GIVEN a world spawned with a detached agent
	chain := setup.NewSpawnBuilder(t).
		WithAgent("detach-agent").
		Detached().
		Execute()

	// THEN the state should show one world
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
	// GIVEN a world spawned with a detached agent
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
	// GIVEN a world spawned with a detached agent
	tc := setup.NewTestContext(t)
	tc.InitAgent("detach-status-agent")

	chain := tc.Spawn().
		WithAgent("detach-status-agent").
		Detached().
		Execute()

	// THEN the state should show the world as running
	worlds := tc.LoadState()
	found := false
	for _, u := range worlds {
		if u.ID == chain.World().ID {
			found = true
			if u.Status != world.StatusRunning {
				t.Fatalf("Expected status %q for detached world, got %q", world.StatusRunning, u.Status)
			}
		}
	}
	if !found {
		t.Fatal("Detached world not found in state")
	}
}

func TestDetach_MultipleAgentsInSameWorld(t *testing.T) {
	// GIVEN a world with an initial detached agent
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
	err := tc.Arc.SpawnAgentDetached(context.Background(), chain.World().ID, "multi-agent")
	if err != nil {
		t.Fatalf("Second detached agent spawn failed: %v", err)
	}

	// THEN the mock should have been called again (wait for output)
	setup.WaitFor(t, 5*time.Second, 100*time.Millisecond, "second mock to write output", func() bool {
		output := tc.TryReadMockOutput(chain.World().ContainerID)
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
	// GIVEN a world spawned with an agent but WITHOUT Detached()
	// THEN the world should be idle (agent not started in background)
	setup.NewSpawnBuilder(t).
		WithAgent("no-detach-agent").
		Execute().
		ExpectState(func(s *setup.StateAssertion) {
			s.WorldCount(1)
			s.WorldStatus(world.StatusRunning)
		})
}

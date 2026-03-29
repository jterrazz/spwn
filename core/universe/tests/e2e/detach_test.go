//go:build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/jterrazz/spwn/core/universe"
	"github.com/jterrazz/spwn/core/universe/tests/e2e/setup"
)

func TestDetach_AgentRunsInBackground(t *testing.T) {
	setup.NewSpawnBuilder(t).
		WithAgent("detach-agent").
		Detached().
		Execute().
		ExpectState(func(s *setup.StateAssertion) {
			s.UniverseCount(1)
		}).
		ExpectContainer(func(c *setup.ContainerAssertion) {
			c.IsRunning()
		}).
		ExpectMock(func(m *setup.MockAssertion) {
			m.WasCalled()
		})
}

func TestDetach_MockSeesEnvironment(t *testing.T) {
	setup.NewSpawnBuilder(t).
		WithAgent("detach-env-agent").
		Detached().
		Execute().
		ExpectMock(func(m *setup.MockAssertion) {
			m.WasCalled()
			m.SawMind()
			m.SawPhysics()
			m.SawFaculties()
		})
}

func TestDetach_StatusShowsRunning(t *testing.T) {
	tc := setup.NewTestContext(t)
	tc.InitAgent("detach-status-agent")

	chain := tc.Spawn().
		WithAgent("detach-status-agent").
		Detached().
		Execute()

	// Immediately after detached spawn, status should be running
	universes := tc.LoadState()
	found := false
	for _, u := range universes {
		if u.ID == chain.Universe().ID {
			found = true
			if u.Status != universe.StatusRunning {
				t.Fatalf("Expected status %q for detached universe, got %q", universe.StatusRunning, u.Status)
			}
		}
	}
	if !found {
		t.Fatal("Detached universe not found in state")
	}
}

func TestDetach_MultipleAgentsInSameUniverse(t *testing.T) {
	tc := setup.NewTestContext(t)
	tc.InitAgent("multi-agent")

	chain := tc.Spawn().
		WithAgent("multi-agent").
		Detached().
		Execute()

	chain.ExpectMock(func(m *setup.MockAssertion) {
		m.WasCalled()
	})

	// Spawn a second agent detached in the same universe
	err := tc.Arc.SpawnAgentDetached(context.Background(), chain.Universe().ID, "multi-agent")
	if err != nil {
		t.Fatalf("Second detached agent spawn failed: %v", err)
	}

	// Give the second agent time to execute
	time.Sleep(500 * time.Millisecond)

	// Container should still be running
	chain.ExpectContainer(func(c *setup.ContainerAssertion) {
		c.IsRunning()
	})

	// Mock should have been called (second invocation overwrites the mock output)
	chain.ExpectMock(func(m *setup.MockAssertion) {
		m.WasCalled()
	})
}

func TestDetach_VsNoDetach(t *testing.T) {
	// Without Detached(), a spawn with agent but no RunAgent() should leave the universe idle
	setup.NewSpawnBuilder(t).
		WithAgent("no-detach-agent").
		Execute().
		ExpectState(func(s *setup.StateAssertion) {
			s.UniverseCount(1)
			s.UniverseStatus(universe.StatusIdle)
		})
}

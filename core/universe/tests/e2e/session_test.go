//go:build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	"spwn.sh/core/universe/tests/e2e/setup"
	agentDomain "spwn.sh/core/agent"
)

func TestSession_FirstSpawnCreatesSession(t *testing.T) {
	// GIVEN a universe with a detached agent
	tc := setup.NewTestContext(t)
	tc.InitAgent("sess-agent")

	chain := tc.Spawn().
		WithAgent("sess-agent").
		Detached().
		Execute()

	// THEN the mock should have been called with a session ID but not resumed
	chain.ExpectMock(func(m *setup.MockAssertion) {
		m.WasCalled()
		m.HasSessionID()
		m.WasNotResumed()
	})

	// AND a session file should exist for this universe
	chain.ExpectMind(func(m *setup.MindAssertion) {
		m.HasSessionFile(chain.Universe().ID)
	})
}

func TestSession_DeterministicID(t *testing.T) {
	// GIVEN two calls to DeterministicSessionID with the same inputs
	id1 := agentDomain.DeterministicSessionID("test-agent", "u-default-12345")
	id2 := agentDomain.DeterministicSessionID("test-agent", "u-default-12345")

	// THEN the IDs should be identical
	if id1 != id2 {
		t.Fatalf("Session IDs not deterministic: %q != %q", id1, id2)
	}

	// AND the ID should be 16 characters
	if len(id1) != 16 {
		t.Fatalf("Expected 16-char session ID, got %d: %q", len(id1), id1)
	}

	// AND different inputs should produce different IDs
	id3 := agentDomain.DeterministicSessionID("other-agent", "u-default-12345")
	if id1 == id3 {
		t.Fatalf("Different agents should produce different session IDs")
	}
}

func TestSession_SecondSpawnResumes(t *testing.T) {
	// GIVEN a universe where an agent has already been spawned once
	tc := setup.NewTestContext(t)
	tc.InitAgent("resume-agent")

	chain := tc.Spawn().
		WithAgent("resume-agent").
		Detached().
		Execute()

	universeID := chain.Universe().ID

	chain.ExpectMock(func(m *setup.MockAssertion) {
		m.WasCalled()
		m.HasSessionID()
		m.WasNotResumed()
	})

	// WHEN a second agent is spawned in the same world
	err := tc.Arc.SpawnAgentDetached(context.Background(), universeID, "resume-agent")
	if err != nil {
		t.Fatalf("Second spawn failed: %v", err)
	}

	// THEN the mock should be called again (wait for it to write output)
	setup.WaitFor(t, 5*time.Second, 100*time.Millisecond, "second mock to write resumed output", func() bool {
		output := tc.TryReadMockOutput(chain.Universe().ContainerID)
		return output != nil && output.Resume
	})

	// AND the mock should show it was resumed with a session ID
	chain.ExpectMock(func(m *setup.MockAssertion) {
		m.WasCalled()
		m.HasSessionID()
		m.WasResumed()
	})
}

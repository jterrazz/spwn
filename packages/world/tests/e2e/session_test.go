//go:build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	"spwn.sh/packages/mind"
	"spwn.sh/packages/world/tests/e2e/setup"
)

func TestSession_FirstSpawnCreatesSession(t *testing.T) {
	// GIVEN a world with a detached agent
	tc := setup.NewTestContext(t)
	tc.InitAgent("sess-agent")

	chain := tc.Spawn().
		WithAgent("sess-agent").
		Detached().
		Execute()

	// THEN the mock should have been called and not resumed
	chain.ExpectMock(func(m *setup.MockAssertion) {
		m.WasCalled()
		m.WasNotResumed()
	})

	// AND a session file should exist for this world
	chain.ExpectMind(func(m *setup.MindAssertion) {
		m.HasSessionFile(chain.World().ID)
	})
}

func TestSession_DeterministicID(t *testing.T) {
	// GIVEN two calls to DeterministicSessionID with the same inputs
	id1 := mind.DeterministicSessionID("test-agent", "u-default-12345")
	id2 := mind.DeterministicSessionID("test-agent", "u-default-12345")

	// THEN the IDs should be identical
	if id1 != id2 {
		t.Fatalf("Session IDs not deterministic: %q != %q", id1, id2)
	}

	// AND the ID should be a 36-char UUID
	if len(id1) != 36 {
		t.Fatalf("Expected 36-char UUID session ID, got %d: %q", len(id1), id1)
	}

	// AND different inputs should produce different IDs
	id3 := mind.DeterministicSessionID("other-agent", "u-default-12345")
	if id1 == id3 {
		t.Fatalf("Different agents should produce different session IDs")
	}
}

func TestSession_FileContainsCorrectWorldID(t *testing.T) {
	// GIVEN a world with a detached agent
	tc := setup.NewTestContext(t)
	tc.InitAgent("sess-wid-agent")

	chain := tc.Spawn().
		WithAgent("sess-wid-agent").
		Detached().
		Execute()

	worldID := chain.World().ID

	chain.ExpectMock(func(m *setup.MockAssertion) {
		m.WasCalled()
	})

	// THEN the session file should contain the correct world ID
	mindPath := mind.AgentDir("sess-wid-agent")
	sess, err := mind.LoadSession(mindPath, worldID)
	if err != nil {
		t.Fatalf("Failed to load session: %v", err)
	}
	if sess == nil {
		t.Fatalf("Expected session file for world %s, not found", worldID)
	}
	if sess.WorldID != worldID {
		t.Fatalf("Session world ID mismatch: expected %q, got %q", worldID, sess.WorldID)
	}
}

func TestSession_PersistsAfterDestroy(t *testing.T) {
	// GIVEN a world where an agent has been spawned and a session created
	tc := setup.NewTestContext(t)
	tc.InitAgent("persist-agent")

	chain := tc.Spawn().
		WithAgent("persist-agent").
		Detached().
		Execute()

	worldID := chain.World().ID

	chain.ExpectMock(func(m *setup.MockAssertion) {
		m.WasCalled()
	})

	// Verify session file exists before destroy
	chain.ExpectMind(func(m *setup.MindAssertion) {
		m.HasSessionFile(worldID)
	})

	// WHEN the world is destroyed
	chain.Destroy()

	// THEN the session file should still exist (persist after destruction)
	mindPath := mind.AgentDir("persist-agent")
	sess, err := mind.LoadSession(mindPath, worldID)
	if err != nil {
		t.Fatalf("Failed to load session after destroy: %v", err)
	}
	if sess == nil {
		t.Fatalf("Session file should persist after world destruction, but it's gone")
	}
}

func TestSession_DifferentWorldsDifferentSessions(t *testing.T) {
	// GIVEN two separate worlds with the same agent
	tc := setup.NewTestContext(t)
	tc.InitAgent("multi-sess-agent")

	chain1 := tc.Spawn().
		WithAgent("multi-sess-agent").
		Detached().
		Execute()

	chain1.ExpectMock(func(m *setup.MockAssertion) {
		m.WasCalled()
	})

	chain2 := tc.Spawn().
		WithAgent("multi-sess-agent").
		Detached().
		Execute()

	chain2.ExpectMock(func(m *setup.MockAssertion) {
		m.WasCalled()
	})

	worldID1 := chain1.World().ID
	worldID2 := chain2.World().ID

	// THEN each world should have its own session file
	mindPath := mind.AgentDir("multi-sess-agent")

	sess1, err := mind.LoadSession(mindPath, worldID1)
	if err != nil || sess1 == nil {
		t.Fatalf("Expected session for world %s", worldID1)
	}
	sess2, err := mind.LoadSession(mindPath, worldID2)
	if err != nil || sess2 == nil {
		t.Fatalf("Expected session for world %s", worldID2)
	}

	// AND the session IDs should be different
	if sess1.ID == sess2.ID {
		t.Fatalf("Different worlds should have different session IDs, both got %q", sess1.ID)
	}
}

func TestSession_SecondSpawnPreservesSessionID(t *testing.T) {
	// GIVEN a world where an agent has already been spawned once
	tc := setup.NewTestContext(t)
	tc.InitAgent("resume-agent")

	chain := tc.Spawn().
		WithAgent("resume-agent").
		Detached().
		Execute()

	worldID := chain.World().ID

	chain.ExpectMock(func(m *setup.MockAssertion) {
		m.WasCalled()
	})

	// Capture the session ID stored after the first spawn.
	mindPath := mind.AgentDir("resume-agent")
	first, err := mind.LoadSession(mindPath, worldID)
	if err != nil || first == nil {
		t.Fatalf("Expected session after first spawn, got err=%v sess=%v", err, first)
	}

	// WHEN a second agent is spawned in the same world
	if err := tc.Arc.SpawnAgentDetached(context.Background(), worldID, "resume-agent"); err != nil {
		t.Fatalf("Second spawn failed: %v", err)
	}

	// Wait for the second mock invocation to land.
	setup.WaitFor(t, 5*time.Second, 100*time.Millisecond, "second mock to write output", func() bool {
		return tc.TryReadMockOutput(chain.World().ContainerID) != nil
	})

	// THEN the session file should still have the same deterministic ID.
	second, err := mind.LoadSession(mindPath, worldID)
	if err != nil || second == nil {
		t.Fatalf("Expected session after second spawn, got err=%v sess=%v", err, second)
	}
	if second.ID != first.ID {
		t.Fatalf("Deterministic session ID changed between spawns: %q vs %q", first.ID, second.ID)
	}
}

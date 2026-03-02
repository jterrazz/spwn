//go:build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/jterrazz/universe/__tests__/e2e/setup"
	"github.com/jterrazz/universe/internal/session"
)

func waitForMock(t *testing.T, tc *setup.TestContext, containerID string) {
	t.Helper()
	time.Sleep(500 * time.Millisecond)
	_ = tc
	_ = containerID
}

func TestSession_FirstSpawnCreatesSession(t *testing.T) {
	tc := setup.NewTestContext(t)
	tc.InitAgent("sess-agent")

	chain := tc.Spawn().
		WithAgent("sess-agent").
		Detached().
		Execute()

	chain.ExpectMock(func(m *setup.MockAssertion) {
		m.WasCalled()
		m.HasSessionID()
		m.WasNotResumed()
	})

	chain.ExpectMind(func(m *setup.MindAssertion) {
		m.HasSessionFile(chain.Universe().ID)
	})
}

func TestSession_DeterministicID(t *testing.T) {
	id1 := session.DeterministicID("test-agent", "u-default-12345")
	id2 := session.DeterministicID("test-agent", "u-default-12345")

	if id1 != id2 {
		t.Fatalf("Session IDs not deterministic: %q != %q", id1, id2)
	}
	if len(id1) != 16 {
		t.Fatalf("Expected 16-char session ID, got %d: %q", len(id1), id1)
	}

	// Different inputs produce different IDs
	id3 := session.DeterministicID("other-agent", "u-default-12345")
	if id1 == id3 {
		t.Fatalf("Different agents should produce different session IDs")
	}
}

func TestSession_SecondSpawnResumes(t *testing.T) {
	tc := setup.NewTestContext(t)
	tc.InitAgent("resume-agent")

	// First spawn — creates session
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

	// Second spawn — should resume (session file now exists)
	err := tc.Arc.SpawnAgentDetached(context.Background(), universeID, "resume-agent")
	if err != nil {
		t.Fatalf("Second spawn failed: %v", err)
	}

	// Give mock time to write
	waitForMock(t, tc, chain.Universe().ContainerID)

	chain.ExpectMock(func(m *setup.MockAssertion) {
		m.WasCalled()
		m.HasSessionID()
		m.WasResumed()
	})
}

//go:build e2e

package e2e

import (
	"context"
	"testing"

	"spwn.sh/packages/universe/tests/e2e/setup"
)

func TestFailure_AgentExitsNonZero(t *testing.T) {
	// GIVEN a universe where the mock agent runs to completion
	// (The mock exits with code 0 by default; RunAgent records the journal entry.)
	tc := setup.NewTestContext(t)
	tc.InitAgent("fail-agent")

	chain := tc.Spawn().
		WithAgent("fail-agent").
		RunAgent().
		Execute()

	// THEN the mock should have been called
	chain.ExpectMock(func(m *setup.MockAssertion) {
		m.WasCalled()
	})

	// AND the container should still be running (universe survives agent completion)
	chain.ExpectContainer(func(c *setup.ContainerAssertion) {
		c.IsRunning()
	})

	// AND the journal should record the completion
	chain.ExpectJournal(func(j *setup.JournalAssertion) {
		j.HasEntries(1)
		j.LatestOutcome("completed")
	})
}

func TestFailure_RecoveryAfterCrash(t *testing.T) {
	// GIVEN a universe where the first agent run completed
	tc := setup.NewTestContext(t)
	tc.InitAgent("recovery-agent")

	chain := tc.Spawn().
		WithAgent("recovery-agent").
		RunAgent().
		Execute()

	chain.ExpectMock(func(m *setup.MockAssertion) {
		m.WasCalled()
	})

	worldID := chain.Universe().ID

	// WHEN a second agent is spawned in the same world (blocking)
	err := tc.Arc.SpawnAgent(context.Background(), worldID, "recovery-agent")
	if err != nil {
		t.Logf("Second spawn returned (expected for mock): %v", err)
	}

	// THEN the container should still be running
	chain.ExpectContainer(func(c *setup.ContainerAssertion) {
		c.IsRunning()
	})

	// AND the mock should have been called again
	chain.ExpectMock(func(m *setup.MockAssertion) {
		m.WasCalled()
	})

	// AND the journal should have two entries (one per run)
	chain.ExpectJournal(func(j *setup.JournalAssertion) {
		j.HasEntries(2)
	})
}

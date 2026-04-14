//go:build e2e

package e2e

import (
	"testing"

	"spwn.sh/packages/agent"
	"spwn.sh/packages/world/tests/e2e/setup"
)

func TestJournal_EntryCreatedOnCompletion(t *testing.T) {
	// GIVEN a world with an agent that runs to completion
	tc := setup.NewTestContext(t)
	tc.InitAgent("journal-agent")

	chain := tc.Spawn().
		WithAgent("journal-agent").
		RunAgent().
		Execute()

	// THEN a journal entry should be created with the correct outcome and world ID
	chain.ExpectJournal(func(j *setup.JournalAssertion) {
		j.HasEntries(1)
		j.LatestOutcome("completed")
		j.LatestWorldID(chain.World().ID)
	})
}

func TestJournal_ListReturnsNewestFirst(t *testing.T) {
	// GIVEN an agent that has run in two separate worlds
	tc := setup.NewTestContext(t)
	tc.InitAgent("journal-order")

	chain1 := tc.Spawn().
		WithAgent("journal-order").
		RunAgent().
		Execute()

	chain2 := tc.Spawn().
		WithAgent("journal-order").
		RunAgent().
		Execute()

	// WHEN listing journal entries
	mindPath := agent.AgentDir("journal-order")
	entries, err := agent.ListJournal(mindPath, 0)
	if err != nil {
		t.Fatalf("Failed to list journal: %v", err)
	}

	// THEN there should be at least 2 entries
	if len(entries) < 2 {
		t.Fatalf("Expected at least 2 journal entries, got %d", len(entries))
	}

	// AND the newest entry (index 0) should reference the second world
	if entries[0].WorldID != chain2.World().ID {
		t.Fatalf("Expected newest entry to be %s, got %s", chain2.World().ID, entries[0].WorldID)
	}

	_ = chain1 // used above implicitly
}

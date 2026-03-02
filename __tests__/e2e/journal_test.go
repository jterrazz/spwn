//go:build e2e

package e2e

import (
	"testing"
	"time"

	"github.com/jterrazz/universe/__tests__/e2e/setup"
	"github.com/jterrazz/universe/internal/journal"
	"github.com/jterrazz/universe/internal/mind"
)

func TestJournal_EntryCreatedOnCompletion(t *testing.T) {
	tc := setup.NewTestContext(t)
	tc.InitAgent("journal-agent")

	chain := tc.Spawn().
		WithAgent("journal-agent").
		RunAgent().
		Execute()

	chain.ExpectJournal(func(j *setup.JournalAssertion) {
		j.HasEntries(1)
		j.LatestOutcome("completed")
		j.LatestUniverseID(chain.Universe().ID)
	})
}

func TestJournal_ListReturnsNewestFirst(t *testing.T) {
	tc := setup.NewTestContext(t)
	tc.InitAgent("journal-order")

	// First spawn
	chain1 := tc.Spawn().
		WithAgent("journal-order").
		RunAgent().
		Execute()

	// Small delay to ensure different timestamps
	time.Sleep(100 * time.Millisecond)

	// Second spawn (new universe)
	chain2 := tc.Spawn().
		WithAgent("journal-order").
		RunAgent().
		Execute()

	mindPath := mind.AgentDir("journal-order")
	entries, err := journal.List(mindPath, 0)
	if err != nil {
		t.Fatalf("Failed to list journal: %v", err)
	}

	if len(entries) < 2 {
		t.Fatalf("Expected at least 2 journal entries, got %d", len(entries))
	}

	// Newest first — second spawn's universe ID should be first
	if entries[0].UniverseID != chain2.Universe().ID {
		t.Fatalf("Expected newest entry to be %s, got %s", chain2.Universe().ID, entries[0].UniverseID)
	}

	_ = chain1 // used above implicitly
}

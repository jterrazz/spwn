package activity

import (
	"os"
	"testing"
	"time"
)

func TestLogAndRead(t *testing.T) {
	// Isolated dir
	tmp := t.TempDir()
	t.Setenv("SPWN_HOME", tmp)

	// Log 3 events
	Log(Event{
		Type:    TypeAgentDreamed,
		Actor:   "neo",
		Verb:    "dreamed",
		Target:  "neo",
		Phrase:  PhraseAgentDreamed("Neo", 2),
		AgentID: "neo",
	})
	Log(Event{
		Type:    TypeWorldSpawned,
		Actor:   "architect",
		Verb:    "spawned",
		Target:  "w-saturn-12345",
		Phrase:  PhraseWorldSpawned("w-saturn-12345", []string{"Neo"}),
		WorldID: "w-saturn-12345",
	})
	Log(Event{
		Type:    TypeAgentJoined,
		Actor:   "architect",
		Verb:    "joined",
		Target:  "w-saturn-12345",
		Phrase:  PhraseAgentJoined("Neo", "w-saturn-12345"),
		WorldID: "w-saturn-12345",
		AgentID: "neo",
	})

	// Read all
	events, err := Read(ReadOpts{})
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}

	// Check filter: by world
	worldEvents, _ := Read(ReadOpts{WorldID: "w-saturn-12345"})
	if len(worldEvents) != 2 {
		t.Fatalf("expected 2 events for world, got %d", len(worldEvents))
	}

	// Check filter: by type
	dreamEvents, _ := Read(ReadOpts{Type: TypeAgentDreamed})
	if len(dreamEvents) != 1 {
		t.Fatalf("expected 1 dream event, got %d", len(dreamEvents))
	}

	// Check filter: limit
	limited, _ := Read(ReadOpts{Limit: 2})
	if len(limited) != 2 {
		t.Fatalf("expected 2 events with limit, got %d", len(limited))
	}

	// Check phrases
	if events[0].Phrase == "" {
		t.Fatal("expected non-empty phrase")
	}
}

func TestReadEmpty(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("SPWN_HOME", tmp)

	events, err := Read(ReadOpts{})
	if err != nil {
		t.Fatalf("read empty: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("expected 0 events, got %d", len(events))
	}
}

func TestSinceFilter(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("SPWN_HOME", tmp)

	now := time.Now().UTC()
	Log(Event{Timestamp: now.Add(-2 * time.Hour), Type: TypeAgentDreamed, Phrase: "old"})
	Log(Event{Timestamp: now, Type: TypeAgentDreamed, Phrase: "new"})

	recent, _ := Read(ReadOpts{Since: now.Add(-1 * time.Hour)})
	if len(recent) != 1 {
		t.Fatalf("expected 1 recent event, got %d", len(recent))
	}
	if recent[0].Phrase != "new" {
		t.Fatalf("expected new, got %s", recent[0].Phrase)
	}
}

func TestFileCreated(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("SPWN_HOME", tmp)

	Log(Event{Type: TypeWorldSpawned, Phrase: "test"})

	if _, err := os.Stat(tmp + "/activity.jsonl"); err != nil {
		t.Fatalf("activity file not created: %v", err)
	}
}

package runtimestate

import (
	"testing"

	"spwn.sh/packages/world/internal/models"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := NewStoreAt(t.TempDir())
	if err != nil {
		t.Fatalf("NewStoreAt: %v", err)
	}
	return s
}

func TestSessionRoundTrip(t *testing.T) {
	s := newTestStore(t)
	if got := s.GetSessionID("w-1", "neo"); got != "" {
		t.Errorf("expected empty session before set, got %q", got)
	}
	if err := s.SetSessionID("w-1", "neo", "sess-abc"); err != nil {
		t.Fatalf("SetSessionID: %v", err)
	}
	if got := s.GetSessionID("w-1", "neo"); got != "sess-abc" {
		t.Errorf("got %q want sess-abc", got)
	}
	// Overwrite
	if err := s.SetSessionID("w-1", "neo", "sess-def"); err != nil {
		t.Fatalf("SetSessionID: %v", err)
	}
	if got := s.GetSessionID("w-1", "neo"); got != "sess-def" {
		t.Errorf("got %q want sess-def", got)
	}
}

func TestAddRemoveAgent(t *testing.T) {
	s := newTestStore(t)
	a := models.AgentRecord{Name: "neo", AgentID: "a-neo-1", Role: "worker"}
	b := models.AgentRecord{Name: "morph", AgentID: "a-morph-2", Role: "chief"}

	if err := s.AddAgent("w-1", a); err != nil {
		t.Fatal(err)
	}
	if err := s.AddAgent("w-1", b); err != nil {
		t.Fatal(err)
	}
	f, _ := s.Load("w-1")
	if len(f.Agents) != 2 {
		t.Fatalf("expected 2 agents, got %d", len(f.Agents))
	}

	// Idempotent on AgentID
	if err := s.AddAgent("w-1", a); err != nil {
		t.Fatal(err)
	}
	f, _ = s.Load("w-1")
	if len(f.Agents) != 2 {
		t.Fatalf("AddAgent should be idempotent on AgentID, got %d agents", len(f.Agents))
	}

	if err := s.RemoveAgent("w-1", "a-neo-1"); err != nil {
		t.Fatal(err)
	}
	f, _ = s.Load("w-1")
	if len(f.Agents) != 1 || f.Agents[0].AgentID != "a-morph-2" {
		t.Fatalf("RemoveAgent did not remove the right entry: %#v", f.Agents)
	}
}

func TestUpdateAgentStatus(t *testing.T) {
	s := newTestStore(t)
	a := models.AgentRecord{Name: "neo", AgentID: "a-1", Status: models.StatusIdle}
	_ = s.AddAgent("w-1", a)
	if err := s.UpdateAgentStatus("w-1", "a-1", models.StatusRunning); err != nil {
		t.Fatal(err)
	}
	f, _ := s.Load("w-1")
	if f.Agents[0].Status != models.StatusRunning {
		t.Errorf("status not updated: %v", f.Agents[0].Status)
	}
	// Missing agent is a no-op, not an error.
	if err := s.UpdateAgentStatus("w-1", "missing", models.StatusRunning); err != nil {
		t.Errorf("UpdateAgentStatus on missing agent should be no-op: %v", err)
	}
}

func TestGC_RemovesOrphans(t *testing.T) {
	s := newTestStore(t)
	_ = s.SetSessionID("w-live", "n", "s")
	_ = s.SetSessionID("w-dead-1", "n", "s")
	_ = s.SetSessionID("w-dead-2", "n", "s")

	if err := s.GC([]string{"w-live"}); err != nil {
		t.Fatal(err)
	}

	if got := s.GetSessionID("w-live", "n"); got != "s" {
		t.Errorf("live world wiped: %q", got)
	}
	if got := s.GetSessionID("w-dead-1", "n"); got != "" {
		t.Errorf("dead world not GC'd: %q", got)
	}
	if got := s.GetSessionID("w-dead-2", "n"); got != "" {
		t.Errorf("dead world not GC'd: %q", got)
	}
}

func TestDelete(t *testing.T) {
	s := newTestStore(t)
	_ = s.SetSessionID("w-1", "n", "s")
	if err := s.Delete("w-1"); err != nil {
		t.Fatal(err)
	}
	if got := s.GetSessionID("w-1", "n"); got != "" {
		t.Errorf("file should be gone: %q", got)
	}
	// Deleting a missing world is a no-op.
	if err := s.Delete("w-missing"); err != nil {
		t.Errorf("Delete missing should be no-op: %v", err)
	}
}

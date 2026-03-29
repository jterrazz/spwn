package state

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jterrazz/spwn/core/universe/internal/models"
)

func tempStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	s, err := NewStoreAt(filepath.Join(dir, "state.json"))
	if err != nil {
		t.Fatalf("NewStoreAt: %v", err)
	}
	return s
}

func seedUniverse(t *testing.T, s *Store, id string) {
	t.Helper()
	if err := s.Save(models.World{ID: id, Status: models.StatusIdle}); err != nil {
		t.Fatalf("Save: %v", err)
	}
}

func TestAddAgent(t *testing.T) {
	s := tempStore(t)
	seedUniverse(t, s, "u1")

	agent := models.AgentRecord{
		Name:    "neo",
		AgentID: "a-neo-12345",
		Tier:    "governor",
		Status:  models.StatusIdle,
	}
	if err := s.AddAgent("u1", agent); err != nil {
		t.Fatalf("AddAgent: %v", err)
	}

	u, err := s.Get("u1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(u.Agents) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(u.Agents))
	}
	if u.Agents[0].Name != "neo" {
		t.Errorf("expected agent name 'neo', got %q", u.Agents[0].Name)
	}
	if u.Agents[0].Tier != "governor" {
		t.Errorf("expected tier 'governor', got %q", u.Agents[0].Tier)
	}
}

func TestAddAgent_MultipleAgents(t *testing.T) {
	s := tempStore(t)
	seedUniverse(t, s, "u1")

	a1 := models.AgentRecord{Name: "gov", AgentID: "a-gov-111", Tier: "governor", Status: models.StatusIdle}
	a2 := models.AgentRecord{Name: "cit", AgentID: "a-cit-222", Tier: "citizen", Status: models.StatusIdle}

	if err := s.AddAgent("u1", a1); err != nil {
		t.Fatalf("AddAgent gov: %v", err)
	}
	if err := s.AddAgent("u1", a2); err != nil {
		t.Fatalf("AddAgent cit: %v", err)
	}

	u, _ := s.Get("u1")
	if len(u.Agents) != 2 {
		t.Fatalf("expected 2 agents, got %d", len(u.Agents))
	}
}

func TestAddAgent_UniverseNotFound(t *testing.T) {
	s := tempStore(t)
	agent := models.AgentRecord{Name: "neo", AgentID: "a-neo-12345"}
	if err := s.AddAgent("nonexistent", agent); err == nil {
		t.Fatal("expected error for nonexistent universe")
	}
}

func TestRemoveAgent(t *testing.T) {
	s := tempStore(t)
	seedUniverse(t, s, "u1")

	a1 := models.AgentRecord{Name: "gov", AgentID: "a-gov-111", Tier: "governor"}
	a2 := models.AgentRecord{Name: "cit", AgentID: "a-cit-222", Tier: "citizen"}
	s.AddAgent("u1", a1)
	s.AddAgent("u1", a2)

	if err := s.RemoveAgent("u1", "a-gov-111"); err != nil {
		t.Fatalf("RemoveAgent: %v", err)
	}

	u, _ := s.Get("u1")
	if len(u.Agents) != 1 {
		t.Fatalf("expected 1 agent after removal, got %d", len(u.Agents))
	}
	if u.Agents[0].AgentID != "a-cit-222" {
		t.Errorf("expected remaining agent 'a-cit-222', got %q", u.Agents[0].AgentID)
	}
}

func TestRemoveAgent_UniverseNotFound(t *testing.T) {
	s := tempStore(t)
	if err := s.RemoveAgent("nonexistent", "a-neo-12345"); err == nil {
		t.Fatal("expected error for nonexistent universe")
	}
}

func TestUpdateAgentStatus(t *testing.T) {
	s := tempStore(t)
	seedUniverse(t, s, "u1")

	agent := models.AgentRecord{Name: "neo", AgentID: "a-neo-12345", Status: models.StatusIdle}
	s.AddAgent("u1", agent)

	if err := s.UpdateAgentStatus("u1", "a-neo-12345", models.StatusRunning); err != nil {
		t.Fatalf("UpdateAgentStatus: %v", err)
	}

	u, _ := s.Get("u1")
	if u.Agents[0].Status != models.StatusRunning {
		t.Errorf("expected status 'running', got %q", u.Agents[0].Status)
	}
}

func TestUpdateAgentStatus_AgentNotFound(t *testing.T) {
	s := tempStore(t)
	seedUniverse(t, s, "u1")

	if err := s.UpdateAgentStatus("u1", "nonexistent", models.StatusRunning); err == nil {
		t.Fatal("expected error for nonexistent agent")
	}
}

func TestUpdateAgentStatus_UniverseNotFound(t *testing.T) {
	s := tempStore(t)
	if err := s.UpdateAgentStatus("nonexistent", "a-1", models.StatusRunning); err == nil {
		t.Fatal("expected error for nonexistent universe")
	}
}

// TestStatePersistence verifies agents survive a reload from disk.
func TestStatePersistence(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	s1, _ := NewStoreAt(path)
	seedUniverse(t, s1, "u1")
	s1.AddAgent("u1", models.AgentRecord{Name: "neo", AgentID: "a-neo-111", Tier: "governor", Status: models.StatusIdle})

	// Create a fresh store pointing at the same file
	s2, _ := NewStoreAt(path)
	u, err := s2.Get("u1")
	if err != nil {
		t.Fatalf("Get from fresh store: %v", err)
	}
	if len(u.Agents) != 1 {
		t.Fatalf("expected 1 agent after reload, got %d", len(u.Agents))
	}
	if u.Agents[0].Name != "neo" {
		t.Errorf("expected agent name 'neo', got %q", u.Agents[0].Name)
	}

	// Clean up
	os.Remove(path)
}

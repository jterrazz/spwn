package mind

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestInitMind_CreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SPWN_HOME", dir)

	path, err := InitMind("test-agent")
	if err != nil {
		t.Fatalf("InitMind failed: %v", err)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatalf("Expected agent directory at %s, not found", path)
	}
}

func TestInitMind_HasStandardLayers(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SPWN_HOME", dir)

	_, err := InitMind("test-agent")
	if err != nil {
		t.Fatalf("InitMind failed: %v", err)
	}

	info, err := InspectAgent("test-agent")
	if err != nil {
		t.Fatalf("InspectAgent failed: %v", err)
	}

	expectedLayers := []string{"core", "skills", "knowledge", "playbooks", "journal"}
	for _, layer := range expectedLayers {
		if _, ok := info.Layers[layer]; !ok {
			t.Errorf("Expected layer %q not found in Mind", layer)
		}
	}
}

func TestAgentDir_ReturnsPath(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SPWN_HOME", dir)

	agentDir := AgentDir("my-agent")
	expected := filepath.Join(dir, "agents", "my-agent")
	if agentDir != expected {
		t.Errorf("AgentDir = %q, want %q", agentDir, expected)
	}
}

func TestDeleteAgent_RemovesDirectory(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SPWN_HOME", dir)

	_, err := InitMind("delete-me")
	if err != nil {
		t.Fatalf("InitMind failed: %v", err)
	}

	err = DeleteAgent("delete-me")
	if err != nil {
		t.Fatalf("DeleteAgent failed: %v", err)
	}

	agentDir := AgentDir("delete-me")
	if _, err := os.Stat(agentDir); !os.IsNotExist(err) {
		t.Fatalf("Expected agent directory to be removed, but it still exists")
	}
}

func TestDeleteAgent_NonexistentAgent(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SPWN_HOME", dir)

	err := DeleteAgent("nonexistent")
	if err == nil {
		t.Fatal("Expected error deleting nonexistent agent, got nil")
	}
}

func TestValidateMind_ValidAgent(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SPWN_HOME", dir)

	_, err := InitMind("valid-agent")
	if err != nil {
		t.Fatalf("InitMind failed: %v", err)
	}

	err = ValidateMind("valid-agent")
	if err != nil {
		t.Fatalf("ValidateMind failed for valid agent: %v", err)
	}
}

func TestValidateMind_NonexistentAgent(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SPWN_HOME", dir)

	err := ValidateMind("nonexistent")
	if err == nil {
		t.Fatal("Expected error validating nonexistent agent, got nil")
	}
}

func TestListAgents_Empty(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SPWN_HOME", dir)
	os.MkdirAll(filepath.Join(dir, "agents"), 0755)

	agents, err := ListAgents()
	if err != nil {
		t.Fatalf("ListAgents failed: %v", err)
	}
	if len(agents) != 0 {
		t.Errorf("Expected 0 agents, got %d", len(agents))
	}
}

func TestListAgents_WithAgents(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SPWN_HOME", dir)

	InitMind("agent-1")
	InitMind("agent-2")

	agents, err := ListAgents()
	if err != nil {
		t.Fatalf("ListAgents failed: %v", err)
	}
	if len(agents) != 2 {
		t.Errorf("Expected 2 agents, got %d", len(agents))
	}
}

func TestDeterministicSessionID_Stable(t *testing.T) {
	id1 := DeterministicSessionID("agent", "world-123")
	id2 := DeterministicSessionID("agent", "world-123")
	if id1 != id2 {
		t.Errorf("Session IDs not deterministic: %q != %q", id1, id2)
	}
	// Session IDs should be non-empty (format may vary)
	if id1 == "" {
		t.Error("Expected non-empty session ID")
	}
}

func TestDeterministicSessionID_DifferentInputs(t *testing.T) {
	id1 := DeterministicSessionID("agent-a", "world-1")
	id2 := DeterministicSessionID("agent-b", "world-1")
	id3 := DeterministicSessionID("agent-a", "world-2")

	if id1 == id2 {
		t.Error("Different agents should produce different session IDs")
	}
	if id1 == id3 {
		t.Error("Different worlds should produce different session IDs")
	}
}

func TestLayerCount_ReturnsNonZero(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SPWN_HOME", dir)

	InitMind("test-agent")

	info, err := InspectAgent("test-agent")
	if err != nil {
		t.Fatalf("InspectAgent failed: %v", err)
	}

	count := LayerCount(info)
	if count == 0 {
		t.Error("Expected non-zero layer count for freshly initialized agent")
	}
}

func TestLayerCount_EmptyLayers(t *testing.T) {
	info := &Info{
		Name:   "empty",
		Layers: map[string][]string{},
	}
	if got := LayerCount(info); got != 0 {
		t.Errorf("expected 0 for empty layers, got %d", got)
	}
}

func TestLayerCount_MixedLayers(t *testing.T) {
	info := &Info{
		Name: "mixed",
		Layers: map[string][]string{
			"core":      {"profile.md"},
			"skills":    {},
			"knowledge": {"fact.md"},
			"journal":   nil,
		},
	}
	if got := LayerCount(info); got != 2 {
		t.Errorf("expected 2, got %d", got)
	}
}

func TestInspectAgent_Success(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SPWN_HOME", dir)

	_, err := InitMind("inspect-me")
	if err != nil {
		t.Fatalf("InitMind: %v", err)
	}

	info, err := InspectAgent("inspect-me")
	if err != nil {
		t.Fatalf("InspectAgent: %v", err)
	}
	if info.Name != "inspect-me" {
		t.Errorf("expected name 'inspect-me', got %q", info.Name)
	}
	if info.Layers == nil {
		t.Error("expected non-nil Layers map")
	}
}

func TestInspectAgent_NotFound(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SPWN_HOME", dir)

	_, err := InspectAgent("ghost")
	if err == nil {
		t.Error("expected error for non-existent agent")
	}
}

func TestInitMind_DuplicateErrors(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SPWN_HOME", dir)

	_, err := InitMind("dup-agent")
	if err != nil {
		t.Fatalf("first InitMind: %v", err)
	}

	_, err = InitMind("dup-agent")
	if err == nil {
		t.Error("expected error on duplicate InitMind")
	}
}

func TestListAgents_ReturnsCorrectNames(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SPWN_HOME", dir)

	InitMind("alpha")
	InitMind("beta")

	agents, err := ListAgents()
	if err != nil {
		t.Fatalf("ListAgents: %v", err)
	}

	names := map[string]bool{}
	for _, a := range agents {
		names[a.Name] = true
	}
	if !names["alpha"] || !names["beta"] {
		t.Errorf("expected agents alpha and beta, got %v", names)
	}
}

func TestAppendJournal_CreatesEntry(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SPWN_HOME", dir)

	_, err := InitMind("journal-agent")
	if err != nil {
		t.Fatalf("InitMind failed: %v", err)
	}

	mindPath := AgentDir("journal-agent")
	err = AppendJournal(mindPath, "world-123", 0, 5*time.Minute)
	if err != nil {
		t.Fatalf("AppendJournal failed: %v", err)
	}

	entries, err := ListJournal(mindPath, 10)
	if err != nil {
		t.Fatalf("ListJournal failed: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("Expected 1 journal entry, got %d", len(entries))
	}
	if entries[0].WorldID != "world-123" {
		t.Errorf("Expected world ID %q, got %q", "world-123", entries[0].WorldID)
	}
}

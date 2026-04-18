package user

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestRetireAgentKnowledge_empty(t *testing.T) {
	base := t.TempDir()
	agentDir := filepath.Join(base, "agents", "alice")
	knowledgeDir := filepath.Join(agentDir, "knowledge")
	if err := os.MkdirAll(knowledgeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Only a .gitkeep — counts as trivially empty.
	if err := os.WriteFile(filepath.Join(knowledgeDir, ".gitkeep"), nil, 0o644); err != nil {
		t.Fatal(err)
	}

	if err := RetireAgentKnowledge.Apply(context.Background(), base); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	if _, err := os.Stat(knowledgeDir); !os.IsNotExist(err) {
		t.Fatalf("expected knowledge dir removed, got err=%v", err)
	}
	// Agent dir itself should survive.
	if _, err := os.Stat(agentDir); err != nil {
		t.Fatalf("agent dir should still exist: %v", err)
	}
}

func TestRetireAgentKnowledge_nonEmpty(t *testing.T) {
	base := t.TempDir()
	agentDir := filepath.Join(base, "agents", "bob")
	knowledgeDir := filepath.Join(agentDir, "knowledge")
	if err := os.MkdirAll(knowledgeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(knowledgeDir, "facts.md"), []byte("# Facts"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := RetireAgentKnowledge.Apply(context.Background(), base); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	if _, err := os.Stat(knowledgeDir); !os.IsNotExist(err) {
		t.Fatalf("expected knowledge dir renamed away, got err=%v", err)
	}
	retired := knowledgeDir + ".retired"
	data, err := os.ReadFile(filepath.Join(retired, "facts.md"))
	if err != nil {
		t.Fatalf("read retired facts: %v", err)
	}
	if string(data) != "# Facts" {
		t.Fatalf("unexpected content: %q", string(data))
	}
}

func TestRetireAgentKnowledge_noAgentsDir(t *testing.T) {
	base := t.TempDir()
	if err := RetireAgentKnowledge.Apply(context.Background(), base); err != nil {
		t.Fatalf("Apply on empty base should be no-op, got: %v", err)
	}
}

func TestRetireAgentKnowledge_noKnowledgeDir(t *testing.T) {
	base := t.TempDir()
	agentDir := filepath.Join(base, "agents", "carol")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := RetireAgentKnowledge.Apply(context.Background(), base); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if _, err := os.Stat(agentDir); err != nil {
		t.Fatalf("agent dir should survive: %v", err)
	}
}

package user

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// TestRetireAgentKnowledge_Fixture covers the byte-deterministic
// happy path: agents/<name>/knowledge/ with only a .gitkeep inside
// (trivially empty) is removed cleanly; the agent dir survives.
// Fixture at testdata/user/014_retire_agent_knowledge/.
func TestRetireAgentKnowledge_Fixture(t *testing.T) {
	runFixture(t, RetireAgentKnowledge, "014_retire_agent_knowledge")
}

// TestRetireAgentKnowledge_nonEmpty lives inline because the
// retired dir name embeds a timestamp when a second retire ever
// happens — not byte-stable enough for a golden fixture.
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

// TestRetireAgentKnowledge_noAgentsDir: fresh install, no agents/ at
// all — migration must be a no-op.
func TestRetireAgentKnowledge_noAgentsDir(t *testing.T) {
	base := t.TempDir()
	if err := RetireAgentKnowledge.Apply(context.Background(), base); err != nil {
		t.Fatalf("Apply on empty base should be no-op, got: %v", err)
	}
}

// TestRetireAgentKnowledge_noKnowledgeDir: agents exist but none
// have a knowledge/ dir — another no-op, agent dirs survive.
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

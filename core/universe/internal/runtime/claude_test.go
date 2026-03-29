package runtime

import (
	"testing"
)

func TestClaudeCodeName(t *testing.T) {
	c := NewClaudeCode()
	if c.Name() != "claude-code" {
		t.Errorf("Name() = %q, want %q", c.Name(), "claude-code")
	}
}

func TestBuildCommandNoMindPath(t *testing.T) {
	c := NewClaudeCode()
	cmd := c.BuildCommand(SpawnConfig{})
	expected := []string{"claude", "--dangerously-skip-permissions"}
	if len(cmd) != len(expected) {
		t.Fatalf("BuildCommand() = %v, want %v", cmd, expected)
	}
	for i := range cmd {
		if cmd[i] != expected[i] {
			t.Errorf("BuildCommand()[%d] = %q, want %q", i, cmd[i], expected[i])
		}
	}
}

func TestBuildCommandWithMindPath(t *testing.T) {
	c := NewClaudeCode()
	// Use a non-existent mind path so LoadSession returns nil (new session)
	cmd := c.BuildCommand(SpawnConfig{
		MindPath:   "/tmp/nonexistent-mind-path-for-test",
		AgentName:  "test-agent",
		UniverseID: "test-universe",
	})

	// Should have: claude --dangerously-skip-permissions --session-id <id>
	// No --resume since no session file exists
	if len(cmd) != 4 {
		t.Fatalf("BuildCommand() = %v, want 4 elements", cmd)
	}
	if cmd[0] != "claude" {
		t.Errorf("cmd[0] = %q, want %q", cmd[0], "claude")
	}
	if cmd[1] != "--dangerously-skip-permissions" {
		t.Errorf("cmd[1] = %q, want %q", cmd[1], "--dangerously-skip-permissions")
	}
	if cmd[2] != "--session-id" {
		t.Errorf("cmd[2] = %q, want %q", cmd[2], "--session-id")
	}
	if cmd[3] == "" {
		t.Error("cmd[3] (session ID) should not be empty")
	}
}

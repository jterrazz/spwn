package worldbook

import (
	"testing"
)

func TestArchitectSystemFiles_HasRequiredFiles(t *testing.T) {
	files := ArchitectSystemFiles()

	required := []string{
		"system/architect/ARCHITECT.md",
		"system/AGENTS.md",
		"system/skills/mind-management.md",
		"system/skills/collaboration.md",
		"system/skills/world-awareness.md",
		"system/skills/self-evolution.md",
		"system/architect/skills/fleet-ops.md",
		"system/architect/skills/task-planning.md",
		"system/architect/skills/monitoring.md",
		"system/architect/stack.md",
	}

	for _, name := range required {
		content, ok := files[name]
		if !ok {
			t.Errorf("missing required file: %s", name)
			continue
		}
		if content == "" {
			t.Errorf("file %s has empty content", name)
		}
	}
}

func TestArchitectSystemFiles_ContentNotEmpty(t *testing.T) {
	files := ArchitectSystemFiles()

	for name, content := range files {
		if content == "" {
			t.Errorf("file %s has empty content", name)
		}
		if len(content) < 10 {
			t.Errorf("file %s seems too short (%d bytes)", name, len(content))
		}
	}
}

func TestArchitectIdentity_ContainsKeyContent(t *testing.T) {
	if len(ArchitectIdentity) == 0 {
		t.Fatal("ArchitectIdentity is empty")
	}
	// Should contain the key "Architect" identity marker
	if !containsString(ArchitectIdentity, "Architect") {
		t.Error("ArchitectIdentity should mention 'Architect'")
	}
	if !containsString(ArchitectIdentity, "STACK_PUSH") {
		t.Error("ArchitectIdentity should mention STACK_PUSH")
	}
}

func containsString(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && contains(s, substr)
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

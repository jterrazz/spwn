package foundation

import (
	"strings"
	"testing"
	"unicode"
)

func TestGenerateWorldID_Format(t *testing.T) {
	tests := []struct {
		name       string
		configName string
		wantPrefix string
	}{
		{"default config", "default", "w-default-"},
		{"custom config", "nebula", "w-nebula-"},
		{"single char", "x", "w-x-"},
		{"hyphenated name", "my-config", "w-my-config-"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := GenerateWorldID(tt.configName)
			if !strings.HasPrefix(id, tt.wantPrefix) {
				t.Errorf("GenerateWorldID(%q) = %q, want prefix %q", tt.configName, id, tt.wantPrefix)
			}
		})
	}
}

func TestGenerateWorldID_SuffixIs5Digits(t *testing.T) {
	id := GenerateWorldID("test")
	// Format: w-test-XXXXX
	parts := strings.SplitN(id, "-", 3)
	if len(parts) != 3 {
		t.Fatalf("expected 3 parts separated by '-', got %d in %q", len(parts), id)
	}

	suffix := parts[2]
	if len(suffix) != 5 {
		t.Errorf("suffix length = %d, want 5, in %q", len(suffix), id)
	}
	for _, r := range suffix {
		if !unicode.IsDigit(r) {
			t.Errorf("suffix contains non-digit %q in %q", r, id)
		}
	}
}

func TestGenerateWorldID_Uniqueness(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 50; i++ {
		id := GenerateWorldID("test")
		if seen[id] {
			t.Logf("warning: duplicate ID %q after %d iterations (unlikely but possible)", id, i)
		}
		seen[id] = true
	}
	// With 5 random digits (100k combinations), 50 calls should almost never collide
	if len(seen) < 2 {
		t.Error("generated 50 IDs but all were identical — randomness is broken")
	}
}

func TestGenerateAgentID_Format(t *testing.T) {
	tests := []struct {
		name      string
		agentName string
		wantPrefix string
	}{
		{"neo agent", "neo", "a-neo-"},
		{"aurora agent", "aurora", "a-aurora-"},
		{"single char", "z", "a-z-"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := GenerateAgentID(tt.agentName)
			if !strings.HasPrefix(id, tt.wantPrefix) {
				t.Errorf("GenerateAgentID(%q) = %q, want prefix %q", tt.agentName, id, tt.wantPrefix)
			}
		})
	}
}

func TestGenerateAgentID_SuffixIs5Digits(t *testing.T) {
	id := GenerateAgentID("neo")
	// Format: a-neo-XXXXX — but name could contain hyphens, so split carefully
	prefix := "a-neo-"
	if !strings.HasPrefix(id, prefix) {
		t.Fatalf("id %q does not start with %q", id, prefix)
	}
	suffix := id[len(prefix):]
	if len(suffix) != 5 {
		t.Errorf("suffix length = %d, want 5, in %q", len(suffix), id)
	}
	for _, r := range suffix {
		if !unicode.IsDigit(r) {
			t.Errorf("suffix contains non-digit %q in %q", r, id)
		}
	}
}

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
		{"custom config", "nebula", "spwn-world-nebula-"},
		{"single char", "x", "spwn-world-x-"},
		{"hyphenated name", "my-config", "spwn-world-my-config-"},
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

func TestGenerateWorldID_DefaultUsesPlanetName(t *testing.T) {
	planetSet := make(map[string]bool)
	for _, p := range PlanetNames {
		planetSet[p] = true
	}

	for i := 0; i < 20; i++ {
		id := GenerateWorldID("default")
		if strings.HasPrefix(id, "spwn-world-default-") {
			t.Errorf("default config should use planet name, got %q", id)
		}
		// Extract planet name: spwn-world-{planet}-{digits}
		trimmed := strings.TrimPrefix(id, "spwn-world-")
		lastDash := strings.LastIndex(trimmed, "-")
		planet := trimmed[:lastDash]
		if !planetSet[planet] {
			t.Errorf("planet name %q not in PlanetNames list, from id %q", planet, id)
		}
	}
}

func TestGenerateWorldID_SuffixIs5Digits(t *testing.T) {
	id := GenerateWorldID("test")
	// Format: spwn-world-test-XXXXX
	const prefix = "spwn-world-test-"
	if !strings.HasPrefix(id, prefix) {
		t.Fatalf("expected prefix %q, got %q", prefix, id)
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
		t.Error("generated 50 IDs but all were identical - randomness is broken")
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

func TestGenerateWorldID_Uniqueness1000(t *testing.T) {
	seen := make(map[string]bool, 1000)
	dupes := 0
	for i := 0; i < 1000; i++ {
		id := GenerateWorldID("test")
		if seen[id] {
			dupes++
		}
		seen[id] = true
	}
	// With 5 digits (100K space), birthday paradox gives ~1% collision chance per pair.
	// We expect some dupes in 1000 iterations, but should have >950 unique.
	if len(seen) < 950 {
		t.Errorf("expected at least 950 unique IDs out of 1000, got %d (dupes=%d)", len(seen), dupes)
	}
}

func TestGenerateAgentID_Uniqueness1000(t *testing.T) {
	seen := make(map[string]bool, 1000)
	dupes := 0
	for i := 0; i < 1000; i++ {
		id := GenerateAgentID("test")
		if seen[id] {
			dupes++
		}
		seen[id] = true
	}
	if len(seen) < 950 {
		t.Errorf("expected at least 950 unique IDs out of 1000, got %d (dupes=%d)", len(seen), dupes)
	}
}

func TestGenerateWorldID_EmptyConfig(t *testing.T) {
	id := GenerateWorldID("")
	if !strings.HasPrefix(id, "spwn-world--") {
		t.Errorf("empty config should produce 'spwn-world--' prefix, got %q", id)
	}
}

func TestGenerateAgentID_EmptyName(t *testing.T) {
	id := GenerateAgentID("")
	if !strings.HasPrefix(id, "a--") {
		t.Errorf("empty name should produce 'a--' prefix, got %q", id)
	}
}

func TestGenerateAgentID_SpecialChars(t *testing.T) {
	tests := []struct {
		name      string
		agentName string
	}{
		{"unicode", "ñoño"},
		{"spaces", "my agent"},
		{"symbols", "agent@v2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := GenerateAgentID(tt.agentName)
			if id == "" {
				t.Error("expected non-empty ID")
			}
			if !strings.HasPrefix(id, "a-") {
				t.Errorf("expected 'a-' prefix, got %q", id)
			}
		})
	}
}

func TestGenerateAgentID_SuffixIs5Digits(t *testing.T) {
	id := GenerateAgentID("neo")
	// Format: a-neo-XXXXX - but name could contain hyphens, so split carefully
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

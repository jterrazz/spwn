package world

import (
	"testing"
)

func TestParseGateFlag_TwoParts(t *testing.T) {
	bridge, err := parseGateFlag("mcp/slack:slack-send")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bridge.Source != "mcp/slack" {
		t.Errorf("Source = %q, want %q", bridge.Source, "mcp/slack")
	}
	if bridge.As != "slack-send" {
		t.Errorf("As = %q, want %q", bridge.As, "slack-send")
	}
	if len(bridge.Capabilities) != 0 {
		t.Errorf("Capabilities = %v, want empty", bridge.Capabilities)
	}
}

func TestParseGateFlag_ThreeParts(t *testing.T) {
	bridge, err := parseGateFlag("mcp/github:github:read,write")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bridge.Source != "mcp/github" {
		t.Errorf("Source = %q, want %q", bridge.Source, "mcp/github")
	}
	if bridge.As != "github" {
		t.Errorf("As = %q, want %q", bridge.As, "github")
	}
	if len(bridge.Capabilities) != 2 {
		t.Fatalf("Capabilities len = %d, want 2", len(bridge.Capabilities))
	}
	if bridge.Capabilities[0] != "read" {
		t.Errorf("Capabilities[0] = %q, want %q", bridge.Capabilities[0], "read")
	}
	if bridge.Capabilities[1] != "write" {
		t.Errorf("Capabilities[1] = %q, want %q", bridge.Capabilities[1], "write")
	}
}

func TestParseGateFlag_ThreePartsWithSend(t *testing.T) {
	bridge, err := parseGateFlag("mcp/slack:slack-send:send")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bridge.Source != "mcp/slack" {
		t.Errorf("Source = %q, want %q", bridge.Source, "mcp/slack")
	}
	if bridge.As != "slack-send" {
		t.Errorf("As = %q, want %q", bridge.As, "slack-send")
	}
	if len(bridge.Capabilities) != 1 || bridge.Capabilities[0] != "send" {
		t.Errorf("Capabilities = %v, want [send]", bridge.Capabilities)
	}
}

func TestParseGateFlag_EmptyCapabilities(t *testing.T) {
	bridge, err := parseGateFlag("src:alias:")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(bridge.Capabilities) != 0 {
		t.Errorf("empty cap string should yield no capabilities, got %v", bridge.Capabilities)
	}
}

func TestParseGateFlag_InvalidFormat(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"no colon", "bad-format"},
		{"empty string", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseGateFlag(tt.input)
			if err == nil {
				t.Errorf("parseGateFlag(%q) should return error", tt.input)
			}
		})
	}
}

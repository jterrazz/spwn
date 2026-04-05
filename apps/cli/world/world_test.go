package world

import (
	"testing"
)

func TestParseWorkspaceFlags(t *testing.T) {
	tests := []struct {
		name     string
		in       []string
		wantName string // name of first workspace
		wantPath string // path of first workspace
		wantRO   bool
		wantLen  int
		wantErr  bool
	}{
		{name: "empty → ephemeral", in: nil, wantLen: 0},
		{name: "single bare path", in: []string{"/host/a"}, wantName: "default", wantPath: "/host/a", wantLen: 1},
		{name: "single named", in: []string{"web=/host/a"}, wantName: "web", wantPath: "/host/a", wantLen: 1},
		{name: "read-only", in: []string{"docs=/host/d:ro"}, wantName: "docs", wantPath: "/host/d", wantRO: true, wantLen: 1},
		{name: "multi named", in: []string{"web=/a", "api=/b"}, wantName: "web", wantPath: "/a", wantLen: 2},
		{name: "bare in multi gets w-index", in: []string{"/a", "/b"}, wantName: "w0", wantPath: "/a", wantLen: 2},
		{name: "empty path errors", in: []string{"name="}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseWorkspaceFlags(tt.in)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got workspaces=%+v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != tt.wantLen {
				t.Fatalf("len = %d, want %d; got=%+v", len(got), tt.wantLen, got)
			}
			if tt.wantLen == 0 {
				return
			}
			if got[0].Name != tt.wantName {
				t.Errorf("Name = %q, want %q", got[0].Name, tt.wantName)
			}
			if got[0].Path != tt.wantPath {
				t.Errorf("Path = %q, want %q", got[0].Path, tt.wantPath)
			}
			if got[0].ReadOnly != tt.wantRO {
				t.Errorf("ReadOnly = %v, want %v", got[0].ReadOnly, tt.wantRO)
			}
		})
	}
}

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

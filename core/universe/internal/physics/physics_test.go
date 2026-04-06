package physics

import (
	"strings"
	"testing"

	"spwn.sh/core/gate"
	"spwn.sh/core/universe/internal/models"
)

func TestGeneratePhysics(t *testing.T) {
	tests := []struct {
		name      string
		manifest  models.Manifest
		wantParts []string
	}{
		{
			name: "contains_all_sections",
			manifest: models.Manifest{
				Physics: models.PhysicsManifest{
					Constants: models.ConstantsManifest{
						CPU: 2, Memory: "1g", Disk: "5g", Timeout: "1h",
					},
				},
			},
			wantParts: []string{
				"# Physics of This Universe",
				"## Constants",
				"2 core(s)",
				"1g",
				"5g",
				"1h",
				"## Laws",
				"bridge (outbound access enabled)",
				"## Tools",
				"/workspace",
				"/mind",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GeneratePhysics(tt.manifest)
			for _, part := range tt.wantParts {
				if !strings.Contains(got, part) {
					t.Errorf("GeneratePhysics() missing %q in output:\n%s", part, got)
				}
			}
		})
	}
}

func TestGenerateFaculties(t *testing.T) {
	tests := []struct {
		name      string
		tools     []string
		bridges   []gate.Bridge
		wantParts []string
	}{
		{
			name:     "with_tools_and_bridges",
			tools:    []string{"bash", "git", "node"},
			bridges: []gate.Bridge{
				{Source: "host:claude", As: "claude", Capabilities: []string{"code", "chat"}},
			},
			wantParts: []string{
				"# Faculties",
				"## Tools",
				"bash, git, node",
				"## Gate Bridges",
				"`claude`",
				"host:claude",
				"[code, chat]",
			},
		},
		{
			name:     "no_tools",
			tools:    nil,
			bridges:  nil,
			wantParts: []string{
				"(none verified)",
			},
		},
		{
			name:     "tools_no_bridges",
			tools:    []string{"curl"},
			bridges:  nil,
			wantParts: []string{
				"curl",
			},
		},
		{
			name:     "bridge_without_capabilities",
			tools:    []string{"sh"},
			bridges: []gate.Bridge{
				{Source: "host:tool", As: "tool"},
			},
			wantParts: []string{
				"`tool`",
				"host:tool",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateFaculties(tt.tools, tt.bridges)
			for _, part := range tt.wantParts {
				if !strings.Contains(got, part) {
					t.Errorf("GenerateFaculties() missing %q in output:\n%s", part, got)
				}
			}
		})
	}

	// Ensure no Gate Bridges section when no bridges
	t.Run("no_gate_bridges_section_when_empty", func(t *testing.T) {
		got := GenerateFaculties([]string{"bash"}, nil)
		if strings.Contains(got, "## Gate Bridges") {
			t.Errorf("should not contain Gate Bridges section when no bridges:\n%s", got)
		}
	})
}

package physics

import (
	"strings"
	"testing"

	"github.com/jterrazz/spwn/core/gate"
	"github.com/jterrazz/spwn/core/universe/internal/models"
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
					Laws: models.LawsManifest{
						Network: "none", MaxProcesses: 64,
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
				"No outbound network access",
				"64",
				"## Elements",
				"/workspace",
				"/mind",
			},
		},
		{
			name: "bridge_network",
			manifest: models.Manifest{
				Physics: models.PhysicsManifest{
					Constants: models.ConstantsManifest{CPU: 1, Memory: "512m", Disk: "2g", Timeout: "30m"},
					Laws:      models.LawsManifest{Network: "bridge", MaxProcesses: 128},
				},
			},
			wantParts: []string{"bridge mode"},
		},
		{
			name: "host_network",
			manifest: models.Manifest{
				Physics: models.PhysicsManifest{
					Constants: models.ConstantsManifest{CPU: 1, Memory: "512m", Disk: "2g", Timeout: "30m"},
					Laws:      models.LawsManifest{Network: "host", MaxProcesses: 128},
				},
			},
			wantParts: []string{"host mode"},
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
		elements  []string
		bridges   []gate.Bridge
		wantParts []string
	}{
		{
			name:     "with_elements_and_bridges",
			elements: []string{"bash", "git", "node"},
			bridges: []gate.Bridge{
				{Source: "host:claude", As: "claude", Capabilities: []string{"code", "chat"}},
			},
			wantParts: []string{
				"# Faculties",
				"## Elements",
				"bash, git, node",
				"## Gate Bridges",
				"`claude`",
				"host:claude",
				"[code, chat]",
			},
		},
		{
			name:     "no_elements",
			elements: nil,
			bridges:  nil,
			wantParts: []string{
				"(none verified)",
			},
		},
		{
			name:     "elements_no_bridges",
			elements: []string{"curl"},
			bridges:  nil,
			wantParts: []string{
				"curl",
			},
		},
		{
			name:     "bridge_without_capabilities",
			elements: []string{"sh"},
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
			got := GenerateFaculties(tt.elements, tt.bridges)
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

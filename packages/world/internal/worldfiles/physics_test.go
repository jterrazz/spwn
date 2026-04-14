package worldfiles

import (
	"strings"
	"testing"

	"spwn.sh/packages/world/internal/models"
)

func TestGeneratePhysics(t *testing.T) {
	tests := []struct {
		name      string
		manifest  models.Manifest
		wantParts []string
	}{
		{
			name:     "contains_all_sections",
			manifest: models.Manifest{},
			wantParts: []string{
				"# Physics of This World",
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
		wantParts []string
	}{
		{
			name:  "with_tools",
			tools: []string{"bash", "git", "node"},
			wantParts: []string{
				"# Faculties",
				"## Tools",
				"bash, git, node",
			},
		},
		{
			name:  "no_tools",
			tools: nil,
			wantParts: []string{
				"(none verified)",
			},
		},
		{
			name:  "single_tool",
			tools: []string{"curl"},
			wantParts: []string{
				"curl",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateFaculties(tt.tools)
			for _, part := range tt.wantParts {
				if !strings.Contains(got, part) {
					t.Errorf("GenerateFaculties() missing %q in output:\n%s", part, got)
				}
			}
		})
	}
}

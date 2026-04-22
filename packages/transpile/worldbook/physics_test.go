package worldbook

import (
	"strings"
	"testing"

	)

func TestGeneratePhysics(t *testing.T) {
	tests := []struct {
		name             string
		knowledgeMounted bool
		wantParts        []string
		forbidParts      []string
	}{
		{
			name:             "contains_all_sections_with_knowledge",
			knowledgeMounted: true,
			wantParts: []string{
				"# Physics of This World",
				"## Laws",
				"bridge (outbound access enabled)",
				"## Topology",
				"/workspaces/",
				"/agents/<your-name>/",
				"knowledge/, inbox/<name>/",
			},
		},
		{
			name:             "omits_knowledge_when_not_mounted",
			knowledgeMounted: false,
			wantParts: []string{
				"## Topology",
				"world-shared state: inbox/<name>/",
			},
			forbidParts: []string{
				"knowledge/",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GeneratePhysics(tt.knowledgeMounted)
			for _, part := range tt.wantParts {
				if !strings.Contains(got, part) {
					t.Errorf("GeneratePhysics() missing %q in output:\n%s", part, got)
				}
			}
			for _, part := range tt.forbidParts {
				if strings.Contains(got, part) {
					t.Errorf("GeneratePhysics() should NOT contain %q:\n%s", part, got)
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

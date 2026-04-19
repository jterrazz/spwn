package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"spwn.sh/packages/transpile"
	"spwn.sh/packages/transpile/source"
)

// Test_requireAgentPrompts pins the QA-surfaced behaviour: compile
// refuses to render an empty / missing AGENTS.md rather than
// templating an empty CLAUDE.md. Before the fix, `spwn compile`
// succeeded silently while `spwn check --deep` flagged it — a loud
// inconsistency between the two commands.
func Test_requireAgentPrompts(t *testing.T) {
	cases := []struct {
		name    string
		src     *source.ProjectSource
		input   transpile.Input
		wantErr string
	}{
		{
			name: "non-empty AGENTS.md passes",
			src: &source.ProjectSource{Agents: []source.AgentSource{
				{Name: "neo", AgentMD: []byte("# neo\n\nhi there")},
			}},
			input: transpile.Input{Agents: []transpile.AgentInput{{Name: "neo"}}},
		},
		{
			name: "empty AGENTS.md errors",
			src: &source.ProjectSource{Agents: []source.AgentSource{
				{Name: "neo", AgentMD: []byte("")},
			}},
			input:   transpile.Input{Agents: []transpile.AgentInput{{Name: "neo"}}},
			wantErr: "agent prompt is missing or empty for: neo",
		},
		{
			name: "whitespace-only AGENTS.md errors",
			src: &source.ProjectSource{Agents: []source.AgentSource{
				{Name: "neo", AgentMD: []byte("   \n\t\n")},
			}},
			input:   transpile.Input{Agents: []transpile.AgentInput{{Name: "neo"}}},
			wantErr: "agent prompt is missing or empty for: neo",
		},
		{
			name: "multiple agents, one missing",
			src: &source.ProjectSource{Agents: []source.AgentSource{
				{Name: "neo", AgentMD: []byte("# neo")},
				{Name: "trin", AgentMD: nil},
			}},
			input: transpile.Input{Agents: []transpile.AgentInput{
				{Name: "neo"}, {Name: "trin"},
			}},
			wantErr: "agent prompt is missing or empty for: trin",
		},
		{
			name:  "nil source is a no-op",
			src:   nil,
			input: transpile.Input{Agents: []transpile.AgentInput{{Name: "neo"}}},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := requireAgentPrompts(tc.src, tc.input)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("want error containing %q, got nil", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error %q does not contain %q", err.Error(), tc.wantErr)
			}
		})
	}
}

// Test_crossCheckRuntimeAdapters locks the `spwn check` warning that
// fires when an agent declares a catalog-known but compile-unimplemented
// runtime (e.g. `spwn:codex`). Without this warning, `check` says
// "valid" and `compile` fails with "unknown runtime" — a silent gap
// between the two commands.
func Test_crossCheckRuntimeAdapters(t *testing.T) {
	// Build a minimal project: spwn.yaml + one agent declaring the
	// codex backend. The claude-code compile adapter is the only one
	// registered at test time, so this should surface as a warning.
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "spwn.yaml"),
		[]byte(`version: 1
name: qa
worlds:
  home:
    agents: [neo]
    workspaces: ["."]
`), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	agentDir := filepath.Join(root, "spwn", "agents", "neo")
	if err := os.MkdirAll(filepath.Join(agentDir, "core"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "AGENTS.md"), []byte("# neo"), 0o644); err != nil {
		t.Fatalf("write agent md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "agent.yaml"), []byte(`name: neo
runtime:
  backend: "spwn:codex"
packages: ["spwn:unix"]
`), 0o644); err != nil {
		t.Fatalf("write agent yaml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "core", "profile.md"), []byte("profile"), 0o644); err != nil {
		t.Fatalf("write profile: %v", err)
	}

	issues := crossCheckRuntimeAdapters(root)
	if len(issues) != 1 {
		t.Fatalf("want 1 issue, got %d: %v", len(issues), issues)
	}
	if issues[0].Level != "warning" {
		t.Fatalf("want warning, got %q", issues[0].Level)
	}
	if !strings.Contains(issues[0].Message, "spwn:codex") {
		t.Fatalf("message does not mention spwn:codex: %q", issues[0].Message)
	}
	if !strings.Contains(issues[0].Hint, "claude-code") {
		t.Fatalf("hint should list claude-code: %q", issues[0].Hint)
	}
}

// Test_safeRemoveOutDir locks the --force cleanup guard. Non-existent
// paths are a no-op, filesystem roots are refused, and a regular dir
// is fully removed. The guard exists to prevent a stray --out flag
// from wiping anything outside the compile tree.
func Test_safeRemoveOutDir(t *testing.T) {
	t.Run("nonexistent is noop", func(t *testing.T) {
		if err := safeRemoveOutDir(filepath.Join(t.TempDir(), "does-not-exist")); err != nil {
			t.Fatalf("want nil, got %v", err)
		}
	})

	t.Run("refuses root", func(t *testing.T) {
		if err := safeRemoveOutDir("/"); err == nil {
			t.Fatal("expected refusal on root")
		}
	})

	t.Run("refuses dot", func(t *testing.T) {
		if err := safeRemoveOutDir("."); err == nil {
			t.Fatal("expected refusal on '.'")
		}
	})

	t.Run("removes regular dir", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "out")
		if err := os.MkdirAll(filepath.Join(dir, "sub"), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir, "f.txt"), []byte("x"), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
		if err := safeRemoveOutDir(dir); err != nil {
			t.Fatalf("remove: %v", err)
		}
		if _, err := os.Stat(dir); !os.IsNotExist(err) {
			t.Fatalf("want dir gone, got stat err=%v", err)
		}
	})

	t.Run("refuses file target", func(t *testing.T) {
		f := filepath.Join(t.TempDir(), "file.txt")
		if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
		if err := safeRemoveOutDir(f); err == nil {
			t.Fatal("expected refusal on non-dir target")
		}
	})
}

package validate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	intmanifest "spwn.sh/packages/project/internal/manifest"
)

// Tests for ruleAutomations. The rule covers 11 numbered checks (see
// the godoc on ruleAutomations) — each test below targets one or two
// of them with a minimal failure case + a happy-path counterpart.
//
// Test naming: TestAutomations_<rule-area>_<scenario>.

// ── (1) Slug ────────────────────────────────────────────────────────

func TestAutomations_Slug_RejectsUppercase(t *testing.T) {
	in := automationInput(t, t.TempDir(), automationFixture{
		Worlds: map[string]worldFixture{
			"brain": {
				Agents: []string{"editor"},
				Automations: map[string]intmanifest.Automation{
					"MorningBrief": {
						On:     intmanifest.Trigger{Cron: "0 6 * * *"},
						Agent:  "editor",
						Prompt: "go",
					},
				},
			},
		},
	})
	expectIssue(t, ruleAutomations(in), LevelError, "must match")
}

// ── (2) Trigger XOR ─────────────────────────────────────────────────

func TestAutomations_Trigger_BothRejected(t *testing.T) {
	in := automationInput(t, t.TempDir(), automationFixture{
		Worlds: map[string]worldFixture{
			"brain": {
				Agents: []string{"editor"},
				Automations: map[string]intmanifest.Automation{
					"x": {
						On: intmanifest.Trigger{
							Cron: "0 6 * * *",
							FS:   &intmanifest.FSTrigger{Path: "."},
						},
						Agent:  "editor",
						Prompt: "go",
					},
				},
			},
		},
	})
	expectIssue(t, ruleAutomations(in), LevelError, "exactly one trigger")
}

func TestAutomations_Trigger_NeitherRejected(t *testing.T) {
	in := automationInput(t, t.TempDir(), automationFixture{
		Worlds: map[string]worldFixture{
			"brain": {
				Agents: []string{"editor"},
				Automations: map[string]intmanifest.Automation{
					"x": {
						Agent:  "editor",
						Prompt: "go",
					},
				},
			},
		},
	})
	expectIssue(t, ruleAutomations(in), LevelError, "trigger missing")
}

// ── (3) Cron parses ─────────────────────────────────────────────────

func TestAutomations_Cron_InvalidExprRejected(t *testing.T) {
	in := automationInput(t, t.TempDir(), automationFixture{
		Worlds: map[string]worldFixture{
			"brain": {
				Agents: []string{"editor"},
				Automations: map[string]intmanifest.Automation{
					"x": {
						On:     intmanifest.Trigger{Cron: "garbage cron"},
						Agent:  "editor",
						Prompt: "go",
					},
				},
			},
		},
	})
	expectIssue(t, ruleAutomations(in), LevelError, "invalid cron expression")
}

func TestAutomations_Cron_StandardFiveFieldAccepted(t *testing.T) {
	exprs := []string{
		"0 6 * * *",
		"*/5 * * * *",
		"0 0 1 1 *",
		"0 0 * * 0",
		"30 14 * * 1-5",
	}
	for _, e := range exprs {
		t.Run(e, func(t *testing.T) {
			in := automationInput(t, t.TempDir(), automationFixture{
				Worlds: map[string]worldFixture{
					"brain": {
						Agents: []string{"editor"},
						Automations: map[string]intmanifest.Automation{
							"x": {
								On:      intmanifest.Trigger{Cron: e},
								Agent:   "editor",
								Prompt:  "go",
								Catchup: "collapse",
							},
						},
					},
				},
			})
			issues := ruleAutomations(in)
			for _, iss := range issues {
				if iss.Level == LevelError {
					t.Errorf("expr %q produced unexpected error: %+v", e, iss)
				}
			}
		})
	}
}

// ── (4) FS path on disk ─────────────────────────────────────────────

func TestAutomations_FS_MissingPathErrors(t *testing.T) {
	// Upgraded from warning to error: filepath.Walk in the watcher's
	// Add fails at register time when the root is missing, killing
	// the daemon with a confusing lstat error. Block at validate
	// time instead.
	in := automationInput(t, t.TempDir(), automationFixture{
		Worlds: map[string]worldFixture{
			"brain": {
				Agents: []string{"curator"},
				Automations: map[string]intmanifest.Automation{
					"inbox": {
						On:     intmanifest.Trigger{FS: &intmanifest.FSTrigger{Path: "./does-not-exist"}},
						Agent:  "curator",
						Prompt: "go",
					},
				},
			},
		},
	})
	expectIssue(t, ruleAutomations(in), LevelError, "does not exist")
}

func TestAutomations_FS_GlobInPathRejected(t *testing.T) {
	in := automationInput(t, t.TempDir(), automationFixture{
		Worlds: map[string]worldFixture{
			"brain": {
				Agents: []string{"curator"},
				Automations: map[string]intmanifest.Automation{
					"inbox": {
						On:     intmanifest.Trigger{FS: &intmanifest.FSTrigger{Path: "./inbox/*.md"}},
						Agent:  "curator",
						Prompt: "go",
					},
				},
			},
		},
	})
	expectIssue(t, ruleAutomations(in), LevelError, "contains glob characters")
}

func TestAutomations_FS_BraceExpansionRejected(t *testing.T) {
	root := t.TempDir()
	mkdir(t, root, "inbox")
	in := automationInput(t, root, automationFixture{
		Worlds: map[string]worldFixture{
			"brain": {
				Agents: []string{"curator"},
				Automations: map[string]intmanifest.Automation{
					"inbox": {
						On: intmanifest.Trigger{FS: &intmanifest.FSTrigger{
							Path:     "./inbox",
							Patterns: []string{"*.{md,txt}"},
						}},
						Agent:  "curator",
						Prompt: "go",
					},
				},
			},
		},
	})
	expectIssue(t, ruleAutomations(in), LevelError, "brace expansion")
}

func TestAutomations_FS_DoublestarRejected(t *testing.T) {
	root := t.TempDir()
	mkdir(t, root, "inbox")
	in := automationInput(t, root, automationFixture{
		Worlds: map[string]worldFixture{
			"brain": {
				Agents: []string{"curator"},
				Automations: map[string]intmanifest.Automation{
					"inbox": {
						On: intmanifest.Trigger{FS: &intmanifest.FSTrigger{
							Path:     "./inbox",
							Patterns: []string{"**/*.md"},
						}},
						Agent:  "curator",
						Prompt: "go",
					},
				},
			},
		},
	})
	expectIssue(t, ruleAutomations(in), LevelError, "doublestar")
}

func TestAutomations_FS_FileInsteadOfDirErrors(t *testing.T) {
	root := t.TempDir()
	// Create a file at the watched path — fs triggers want directories.
	filePath := filepath.Join(root, "inbox.md")
	if err := os.WriteFile(filePath, []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	in := automationInput(t, root, automationFixture{
		Worlds: map[string]worldFixture{
			"brain": {
				Agents: []string{"curator"},
				Automations: map[string]intmanifest.Automation{
					"inbox": {
						On:     intmanifest.Trigger{FS: &intmanifest.FSTrigger{Path: "./inbox.md"}},
						Agent:  "curator",
						Prompt: "go",
					},
				},
			},
		},
	})
	expectIssue(t, ruleAutomations(in), LevelError, "is a file")
}

// ── (5) FS event allow-list ─────────────────────────────────────────

func TestAutomations_FS_UnknownEventRejected(t *testing.T) {
	root := t.TempDir()
	mkdir(t, root, "inbox")
	in := automationInput(t, root, automationFixture{
		Worlds: map[string]worldFixture{
			"brain": {
				Agents: []string{"curator"},
				Automations: map[string]intmanifest.Automation{
					"inbox": {
						On: intmanifest.Trigger{FS: &intmanifest.FSTrigger{
							Path:   "./inbox",
							Events: []string{"create", "delete"}, // delete unsupported
						}},
						Agent:  "curator",
						Prompt: "go",
					},
				},
			},
		},
	})
	expectIssue(t, ruleAutomations(in), LevelError, "unknown fs event \"delete\"")
}

// ── (6) FS debounce range ───────────────────────────────────────────

func TestAutomations_FS_DebounceTooShortRejected(t *testing.T) {
	root := t.TempDir()
	mkdir(t, root, "inbox")
	in := automationInput(t, root, automationFixture{
		Worlds: map[string]worldFixture{
			"brain": {
				Agents: []string{"curator"},
				Automations: map[string]intmanifest.Automation{
					"inbox": {
						On: intmanifest.Trigger{FS: &intmanifest.FSTrigger{
							Path:     "./inbox",
							Debounce: intmanifest.Duration(50 * time.Millisecond),
						}},
						Agent:  "curator",
						Prompt: "go",
					},
				},
			},
		},
	})
	expectIssue(t, ruleAutomations(in), LevelError, "below the 100ms minimum")
}

func TestAutomations_FS_DebounceTooLongRejected(t *testing.T) {
	root := t.TempDir()
	mkdir(t, root, "inbox")
	in := automationInput(t, root, automationFixture{
		Worlds: map[string]worldFixture{
			"brain": {
				Agents: []string{"curator"},
				Automations: map[string]intmanifest.Automation{
					"inbox": {
						On: intmanifest.Trigger{FS: &intmanifest.FSTrigger{
							Path:     "./inbox",
							Debounce: intmanifest.Duration(2 * time.Hour),
						}},
						Agent:  "curator",
						Prompt: "go",
					},
				},
			},
		},
	})
	expectIssue(t, ruleAutomations(in), LevelError, "exceeds the 1h maximum")
}

// ── (7) Patterns non-empty strings ──────────────────────────────────

func TestAutomations_FS_EmptyPatternRejected(t *testing.T) {
	root := t.TempDir()
	mkdir(t, root, "inbox")
	in := automationInput(t, root, automationFixture{
		Worlds: map[string]worldFixture{
			"brain": {
				Agents: []string{"curator"},
				Automations: map[string]intmanifest.Automation{
					"inbox": {
						On: intmanifest.Trigger{FS: &intmanifest.FSTrigger{
							Path:     "./inbox",
							Patterns: []string{"*.md", "  "},
						}},
						Agent:  "curator",
						Prompt: "go",
					},
				},
			},
		},
	})
	expectIssue(t, ruleAutomations(in), LevelError, "non-empty glob")
}

// ── (8) Body XOR ────────────────────────────────────────────────────

func TestAutomations_Body_BothRejected(t *testing.T) {
	in := automationInput(t, t.TempDir(), automationFixture{
		Worlds: map[string]worldFixture{
			"brain": {
				Agents: []string{"editor"},
				Automations: map[string]intmanifest.Automation{
					"x": {
						On:      intmanifest.Trigger{Cron: "0 6 * * *"},
						Agent:   "editor",
						Prompt:  "inline",
						Command: "command/x",
					},
				},
			},
		},
	})
	expectIssue(t, ruleAutomations(in), LevelError, "exactly one body")
}

func TestAutomations_Body_NeitherRejected(t *testing.T) {
	in := automationInput(t, t.TempDir(), automationFixture{
		Worlds: map[string]worldFixture{
			"brain": {
				Agents: []string{"editor"},
				Automations: map[string]intmanifest.Automation{
					"x": {
						On:    intmanifest.Trigger{Cron: "0 6 * * *"},
						Agent: "editor",
					},
				},
			},
		},
	})
	expectIssue(t, ruleAutomations(in), LevelError, "body missing")
}

// ── (9) Command ref ─────────────────────────────────────────────────

func TestAutomations_CommandRef_BadShapeRejected(t *testing.T) {
	in := automationInput(t, t.TempDir(), automationFixture{
		Worlds: map[string]worldFixture{
			"brain": {
				Agents: []string{"editor"},
				Automations: map[string]intmanifest.Automation{
					"x": {
						On:      intmanifest.Trigger{Cron: "0 6 * * *"},
						Agent:   "editor",
						Command: "skill/foo", // wrong scheme
					},
				},
			},
		},
	})
	expectIssue(t, ruleAutomations(in), LevelError, "must use the `command/<name>` form")
}

func TestAutomations_CommandRef_MissingFileRejected(t *testing.T) {
	in := automationInput(t, t.TempDir(), automationFixture{
		Worlds: map[string]worldFixture{
			"brain": {
				Agents: []string{"editor"},
				Automations: map[string]intmanifest.Automation{
					"x": {
						On:      intmanifest.Trigger{Cron: "0 6 * * *"},
						Agent:   "editor",
						Command: "command/morning-brief",
					},
				},
			},
		},
	})
	expectIssue(t, ruleAutomations(in), LevelError, "command file not found")
}

func TestAutomations_CommandRef_FoundFileAccepted(t *testing.T) {
	root := t.TempDir()
	mkdir(t, root, filepath.Join("spwn", "commands"))
	must(t, os.WriteFile(filepath.Join(root, "spwn", "commands", "morning-brief.md"), []byte("body"), 0o644))

	in := automationInput(t, root, automationFixture{
		Worlds: map[string]worldFixture{
			"brain": {
				Agents: []string{"editor"},
				Automations: map[string]intmanifest.Automation{
					"morning-brief": {
						On:      intmanifest.Trigger{Cron: "0 6 * * *"},
						Agent:   "editor",
						Command: "command/morning-brief",
					},
				},
			},
		},
	})
	for _, iss := range ruleAutomations(in) {
		if iss.Level == LevelError {
			t.Errorf("happy path produced unexpected error: %+v", iss)
		}
	}
}

// ── (10) Catchup mode ───────────────────────────────────────────────

func TestAutomations_Catchup_UnknownModeRejected(t *testing.T) {
	in := automationInput(t, t.TempDir(), automationFixture{
		Worlds: map[string]worldFixture{
			"brain": {
				Agents: []string{"editor"},
				Automations: map[string]intmanifest.Automation{
					"x": {
						On:      intmanifest.Trigger{Cron: "0 6 * * *"},
						Agent:   "editor",
						Prompt:  "go",
						Catchup: "rewind",
					},
				},
			},
		},
	})
	expectIssue(t, ruleAutomations(in), LevelError, "unknown catchup mode")
}

func TestAutomations_Catchup_OnFsLogsInfo(t *testing.T) {
	root := t.TempDir()
	mkdir(t, root, "inbox")
	in := automationInput(t, root, automationFixture{
		Worlds: map[string]worldFixture{
			"brain": {
				Agents: []string{"curator"},
				Automations: map[string]intmanifest.Automation{
					"x": {
						On:      intmanifest.Trigger{FS: &intmanifest.FSTrigger{Path: "./inbox"}},
						Agent:   "curator",
						Prompt:  "go",
						Catchup: "collapse", // legal value, but no effect on fs
					},
				},
			},
		},
	})
	expectIssue(t, ruleAutomations(in), LevelInfo, "catchup is cron-only")
}

// ── (11) Agent membership ───────────────────────────────────────────

func TestAutomations_Agent_NotInWorldRejected(t *testing.T) {
	in := automationInput(t, t.TempDir(), automationFixture{
		Worlds: map[string]worldFixture{
			"brain": {
				Agents: []string{"editor"},
				Automations: map[string]intmanifest.Automation{
					"x": {
						On:     intmanifest.Trigger{Cron: "0 6 * * *"},
						Agent:  "ghostwriter", // not in agents
						Prompt: "go",
					},
				},
			},
		},
	})
	expectIssue(t, ruleAutomations(in), LevelError, "is not in world")
}

func TestAutomations_Agent_MissingRejected(t *testing.T) {
	in := automationInput(t, t.TempDir(), automationFixture{
		Worlds: map[string]worldFixture{
			"brain": {
				Agents: []string{"editor"},
				Automations: map[string]intmanifest.Automation{
					"x": {
						On:     intmanifest.Trigger{Cron: "0 6 * * *"},
						Prompt: "go",
					},
				},
			},
		},
	})
	expectIssue(t, ruleAutomations(in), LevelError, "must declare an `agent:`")
}

// ── Happy path — full valid automation produces zero errors ─────────

func TestAutomations_FullValid_ZeroErrors(t *testing.T) {
	root := t.TempDir()
	mkdir(t, root, "inbox")
	mkdir(t, root, filepath.Join("spwn", "commands"))
	must(t, os.WriteFile(filepath.Join(root, "spwn", "commands", "process.md"), []byte("body"), 0o644))

	in := automationInput(t, root, automationFixture{
		Worlds: map[string]worldFixture{
			"brain": {
				Agents: []string{"editor", "curator"},
				Automations: map[string]intmanifest.Automation{
					"morning-brief": {
						On:      intmanifest.Trigger{Cron: "0 6 * * *"},
						Agent:   "editor",
						Prompt:  "Review the last 24h.",
						Catchup: "collapse",
					},
					"inbox-pull": {
						On: intmanifest.Trigger{FS: &intmanifest.FSTrigger{
							Path:      "./inbox",
							Events:    []string{"create"},
							Recursive: true,
							Debounce:  intmanifest.Duration(10 * time.Second),
							Patterns:  []string{"*.md"},
						}},
						Agent:   "curator",
						Command: "command/process",
					},
				},
			},
		},
	})

	for _, iss := range ruleAutomations(in) {
		if iss.Level == LevelError {
			t.Errorf("happy path produced unexpected error: %+v", iss)
		}
	}
}

// ── helpers ─────────────────────────────────────────────────────────

type automationFixture struct {
	Worlds map[string]worldFixture
}

type worldFixture struct {
	Agents      []string
	Automations map[string]intmanifest.Automation
}

// automationInput builds a validate.Input rooted at the given disk
// directory with the given world declarations. Mirrors minimalInput
// from validate_edge_test.go but skips the agent.yaml scaffolding —
// the automations rule only reads the manifest, not agent dirs.
func automationInput(t *testing.T, root string, fx automationFixture) Input {
	t.Helper()
	worlds := map[string]intmanifest.World{}
	for wname, w := range fx.Worlds {
		worlds[wname] = intmanifest.World{
			Agents:      w.Agents,
			Workspaces:  []string{"."},
			Automations: w.Automations,
		}
	}
	return Input{
		Root: root,
		Manifest: &intmanifest.Manifest{
			Version: intmanifest.CurrentVersion,
			Name:    "automations-test",
			Worlds:  worlds,
		},
	}
}

// expectIssue asserts that the given issues contain exactly one entry
// at the expected level whose Message contains needle. Used to keep
// the test bodies short — the rule emits Hint text we don't want to
// pin in goldens.
func expectIssue(t *testing.T, issues []Issue, level Level, needle string) {
	t.Helper()
	for _, iss := range issues {
		if iss.Level == level && strings.Contains(iss.Message, needle) {
			return
		}
	}
	t.Fatalf("expected %s issue containing %q, got: %+v", level, needle, issues)
}

func mkdir(t *testing.T, root, rel string) {
	t.Helper()
	must(t, os.MkdirAll(filepath.Join(root, rel), 0o755))
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

package manifest

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Parser tests for the automations field on a world. The validation
// layer (packages/project/internal/validate) owns the cross-field
// rules — here we only assert the YAML round-trips into the struct
// shape downstream code expects.

// ── Cron variant ────────────────────────────────────────────────────

func TestAutomations_CronInlinePrompt(t *testing.T) {
	m := loadAutomationsFixture(t, `version: 1
name: test
worlds:
  brain:
    agents: [editor]
    workspaces: [.]
    automations:
      morning-brief:
        on:
          cron: "0 6 * * *"
        agent: editor
        prompt: "Cluster yesterday's signal."
`)

	autos := m.Worlds["brain"].Automations
	if len(autos) != 1 {
		t.Fatalf("expected 1 automation, got %d", len(autos))
	}
	a := autos["morning-brief"]
	if a.On.Cron != "0 6 * * *" {
		t.Errorf("cron = %q, want %q", a.On.Cron, "0 6 * * *")
	}
	if a.On.FS != nil {
		t.Errorf("fs should be nil for a cron automation, got %+v", a.On.FS)
	}
	if a.Agent != "editor" {
		t.Errorf("agent = %q, want %q", a.Agent, "editor")
	}
	if a.Prompt != "Cluster yesterday's signal." {
		t.Errorf("prompt = %q", a.Prompt)
	}
	// ApplyDefaults should have stamped catchup=collapse for cron.
	if a.Catchup != "collapse" {
		t.Errorf("catchup default = %q, want %q", a.Catchup, "collapse")
	}
}

func TestAutomations_CronCommandRef(t *testing.T) {
	m := loadAutomationsFixture(t, `version: 1
name: test
worlds:
  brain:
    agents: [editor]
    workspaces: [.]
    automations:
      morning-brief:
        on: { cron: "0 6 * * *" }
        agent: editor
        command: command/morning-brief
`)

	a := m.Worlds["brain"].Automations["morning-brief"]
	if a.Command != "command/morning-brief" {
		t.Errorf("command = %q", a.Command)
	}
	if a.Prompt != "" {
		t.Errorf("prompt should be empty, got %q", a.Prompt)
	}
}

func TestAutomations_CatchupSkip(t *testing.T) {
	m := loadAutomationsFixture(t, `version: 1
name: test
worlds:
  brain:
    agents: [editor]
    workspaces: [.]
    automations:
      brief:
        on: { cron: "0 6 * * *" }
        agent: editor
        prompt: "go"
        catchup: skip
`)

	a := m.Worlds["brain"].Automations["brief"]
	// Explicit catchup must be preserved by ApplyDefaults — the
	// default substitution only fires for empty strings.
	if a.Catchup != "skip" {
		t.Errorf("catchup = %q, want %q (no default override on explicit value)", a.Catchup, "skip")
	}
}

// ── FS variant ──────────────────────────────────────────────────────

func TestAutomations_FSAllDefaults(t *testing.T) {
	m := loadAutomationsFixture(t, `version: 1
name: test
worlds:
  brain:
    agents: [curator]
    workspaces: [.]
    automations:
      inbox:
        on:
          fs:
            path: ./inbox
        agent: curator
        prompt: "new file"
`)

	a := m.Worlds["brain"].Automations["inbox"]
	if a.On.FS == nil {
		t.Fatalf("fs trigger nil")
	}
	fs := a.On.FS
	if fs.Path != "./inbox" {
		t.Errorf("path = %q", fs.Path)
	}
	// Defaults: events=[create], debounce=1s, recursive=false, patterns=[].
	if len(fs.Events) != 1 || fs.Events[0] != "create" {
		t.Errorf("events default = %v, want [create]", fs.Events)
	}
	if fs.Debounce.AsDuration() != 1*time.Second {
		t.Errorf("debounce default = %s, want 1s", fs.Debounce.AsDuration())
	}
	if fs.Recursive {
		t.Errorf("recursive default = true, want false")
	}
	if len(fs.Patterns) != 0 {
		t.Errorf("patterns default = %v, want []", fs.Patterns)
	}
	// FS triggers don't get a catchup default.
	if a.Catchup != "" {
		t.Errorf("catchup on fs trigger = %q, want \"\"", a.Catchup)
	}
}

func TestAutomations_FSExplicitOverrides(t *testing.T) {
	m := loadAutomationsFixture(t, `version: 1
name: test
worlds:
  brain:
    agents: [curator]
    workspaces: [.]
    automations:
      inbox:
        on:
          fs:
            path: ./inbox
            events: [create, write]
            recursive: true
            debounce: 10s
            patterns: ["*.md", "*.txt"]
        agent: curator
        prompt: "{{ .Event.Path }}"
`)

	fs := m.Worlds["brain"].Automations["inbox"].On.FS
	if !fs.Recursive {
		t.Errorf("recursive = false, want true")
	}
	if fs.Debounce.AsDuration() != 10*time.Second {
		t.Errorf("debounce = %s, want 10s", fs.Debounce.AsDuration())
	}
	wantEvents := []string{"create", "write"}
	if len(fs.Events) != 2 || fs.Events[0] != wantEvents[0] || fs.Events[1] != wantEvents[1] {
		t.Errorf("events = %v, want %v", fs.Events, wantEvents)
	}
	wantPatterns := []string{"*.md", "*.txt"}
	if len(fs.Patterns) != 2 || fs.Patterns[0] != wantPatterns[0] || fs.Patterns[1] != wantPatterns[1] {
		t.Errorf("patterns = %v, want %v", fs.Patterns, wantPatterns)
	}
}

// ── Duration codec ──────────────────────────────────────────────────

func TestDuration_ParsesShortForm(t *testing.T) {
	cases := []struct {
		input string
		want  time.Duration
	}{
		{"100ms", 100 * time.Millisecond},
		{"1s", 1 * time.Second},
		{"30s", 30 * time.Second},
		{"5m", 5 * time.Minute},
		{"1h", 1 * time.Hour},
		{"1h30m", 90 * time.Minute},
	}
	for _, c := range cases {
		t.Run(c.input, func(t *testing.T) {
			m := loadAutomationsFixture(t, `version: 1
name: test
worlds:
  brain:
    agents: [a]
    workspaces: [.]
    automations:
      x:
        on: { fs: { path: ./x, debounce: `+c.input+` } }
        agent: a
        prompt: "p"
`)
			got := m.Worlds["brain"].Automations["x"].On.FS.Debounce.AsDuration()
			if got != c.want {
				t.Errorf("debounce %q = %s, want %s", c.input, got, c.want)
			}
		})
	}
}

func TestDuration_RejectsGarbage(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "spwn.yaml")
	content := `version: 1
name: test
worlds:
  brain:
    agents: [a]
    workspaces: [.]
    automations:
      x:
        on: { fs: { path: ./x, debounce: "not a duration" } }
        agent: a
        prompt: "p"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadPath(path); err == nil {
		t.Fatalf("expected parse error on bogus duration, got nil")
	}
}

func TestDuration_MarshalRoundTrip(t *testing.T) {
	// Round-trip property: parsing a duration and re-marshalling must
	// produce a string the parser accepts. Catches regressions where
	// MarshalYAML emits raw nanoseconds (which yaml.v3 would re-parse
	// as int, breaking author readability).
	d := Duration(90 * time.Minute)
	emitted, err := d.MarshalYAML()
	if err != nil {
		t.Fatalf("MarshalYAML: %v", err)
	}
	s, ok := emitted.(string)
	if !ok {
		t.Fatalf("MarshalYAML returned %T, want string", emitted)
	}
	parsed, err := time.ParseDuration(s)
	if err != nil {
		t.Fatalf("re-parse %q: %v", s, err)
	}
	if parsed != 90*time.Minute {
		t.Errorf("round-trip = %s, want 1h30m0s", parsed)
	}
}

// ── Empty / absent ──────────────────────────────────────────────────

func TestAutomations_AbsentKey(t *testing.T) {
	m := loadAutomationsFixture(t, `version: 1
name: test
worlds:
  brain:
    agents: [a]
    workspaces: [.]
`)
	if len(m.Worlds["brain"].Automations) != 0 {
		t.Errorf("absent automations should be empty, got %v", m.Worlds["brain"].Automations)
	}
}

func TestAutomations_EmptyMap(t *testing.T) {
	m := loadAutomationsFixture(t, `version: 1
name: test
worlds:
  brain:
    agents: [a]
    workspaces: [.]
    automations: {}
`)
	if len(m.Worlds["brain"].Automations) != 0 {
		t.Errorf("empty map should produce empty automations, got %v", m.Worlds["brain"].Automations)
	}
}

// ── Multiple automations / multiple worlds ──────────────────────────

func TestAutomations_MultipleInOneWorld(t *testing.T) {
	m := loadAutomationsFixture(t, `version: 1
name: test
worlds:
  brain:
    agents: [editor, curator]
    workspaces: [.]
    automations:
      morning-brief:
        on: { cron: "0 6 * * *" }
        agent: editor
        prompt: "p1"
      inbox-pull:
        on:
          fs:
            path: ./inbox
        agent: curator
        prompt: "p2"
`)
	autos := m.Worlds["brain"].Automations
	if len(autos) != 2 {
		t.Fatalf("expected 2 automations, got %d", len(autos))
	}
	if autos["morning-brief"].On.Cron == "" {
		t.Errorf("morning-brief should have cron")
	}
	if autos["inbox-pull"].On.FS == nil {
		t.Errorf("inbox-pull should have fs")
	}
}

func TestAutomations_AcrossMultipleWorlds(t *testing.T) {
	m := loadAutomationsFixture(t, `version: 1
name: test
worlds:
  brain:
    agents: [editor]
    workspaces: [.]
    automations:
      brief: { on: { cron: "0 6 * * *" }, agent: editor, prompt: "p" }
  scratch:
    agents: [explorer]
    workspaces: [.]
    automations:
      scout: { on: { cron: "0 12 * * *" }, agent: explorer, prompt: "p" }
`)
	if len(m.Worlds["brain"].Automations) != 1 {
		t.Errorf("brain should have 1 automation")
	}
	if len(m.Worlds["scratch"].Automations) != 1 {
		t.Errorf("scratch should have 1 automation")
	}
}

// ── helpers ─────────────────────────────────────────────────────────

func loadAutomationsFixture(t *testing.T, content string) *Manifest {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "spwn.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	m, err := LoadPath(path)
	if err != nil {
		t.Fatalf("LoadPath: %v", err)
	}
	return m
}

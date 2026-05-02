package automation

import (
	"strings"
	"testing"
	"time"
)

// ── prompt → identity (no templating) ───────────────────────────────

func TestRender_PlainPromptUnchanged(t *testing.T) {
	got, err := renderPrompt("hello world", "", FireSource{Now: time.Now()})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if got != "hello world" {
		t.Errorf("got %q", got)
	}
}

// ── command: ref placeholder (Phase 2 stub) ─────────────────────────

func TestRender_CommandRefPlaceholder(t *testing.T) {
	// Phase 2 doesn't read spwn/commands/<n>.md from disk yet; the
	// caller (engine) still has to fire so we can receipt-log the
	// path. The placeholder is explicit so QA finds it instantly.
	got, err := renderPrompt("", "command/morning-brief", FireSource{Now: time.Now()})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(got, "morning-brief") {
		t.Errorf("placeholder missing command name: %q", got)
	}
}

func TestRender_NeitherBodyErrors(t *testing.T) {
	if _, err := renderPrompt("", "", FireSource{Now: time.Now()}); err == nil {
		t.Error("expected error when neither prompt nor command set")
	}
}

// ── cron template variables ─────────────────────────────────────────

func TestRender_NowVariable(t *testing.T) {
	now := mustParse(t, "2026-05-02T06:00:00Z")
	got, err := renderPrompt("tick at {{ .Now }}", "", FireSource{Now: now})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(got, "2026-05-02") {
		t.Errorf("rendered prompt missing date: %q", got)
	}
}

func TestRender_DateHelper(t *testing.T) {
	now := mustParse(t, "2026-05-02T06:00:00Z")
	got, err := renderPrompt(`brief for {{ .Now | date "2006-01-02" }}`, "", FireSource{Now: now})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if got != "brief for 2026-05-02" {
		t.Errorf("got %q", got)
	}
}

// ── catchup template fields ─────────────────────────────────────────

func TestRender_MissedAndLastFired(t *testing.T) {
	src := FireSource{
		Now:       mustParse(t, "2026-05-03T08:14:00Z"),
		Reason:    "catchup",
		Missed:    2,
		LastFired: mustParse(t, "2026-05-01T06:00:00Z"),
	}
	body := `Brief.{{ if .Missed }} ({{ .Missed }} missed since {{ .LastFired | date "2006-01-02" }}){{ end }}`
	got, err := renderPrompt(body, "", src)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	want := "Brief. (2 missed since 2026-05-01)"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRender_OnTimeSuppressesMissedClause(t *testing.T) {
	src := FireSource{
		Now:    mustParse(t, "2026-05-03T06:00:00Z"),
		Reason: "on-time",
		// Missed = 0; LastFired zero.
	}
	body := `Brief.{{ if .Missed }} ({{ .Missed }} missed){{ end }}`
	got, err := renderPrompt(body, "", src)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if got != "Brief." {
		t.Errorf("got %q (the {{ if .Missed }} guard should suppress)", got)
	}
}

// ── fs event variables (Phase 3 — ensure no crash on missing) ───────

func TestRender_EventNilSafe(t *testing.T) {
	// Cron fires set EventPaths empty. A template that defensively
	// references {{ .Event.Path }} should produce empty string, not
	// panic — keeps the templates of mixed-trigger projects simple.
	body := `path is {{ if .Event }}{{ .Event.Path }}{{ else }}none{{ end }}`
	got, err := renderPrompt(body, "", FireSource{Kind: "cron", Now: time.Now()})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if got != "path is none" {
		t.Errorf("got %q", got)
	}
}

func TestRender_FSEvent(t *testing.T) {
	src := FireSource{
		Kind:       "fs",
		Now:        time.Now(),
		EventPaths: []string{"/inbox/foo.md", "/inbox/bar.md"},
		EventKind:  "create",
	}
	body := "kind={{ .Event.Kind }} first={{ .Event.Name }} count={{ len .Event.Paths }}"
	got, err := renderPrompt(body, "", src)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	want := "kind=create first=foo.md count=2"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// ── error paths ─────────────────────────────────────────────────────

func TestRender_BadTemplateSyntaxErrors(t *testing.T) {
	if _, err := renderPrompt("hi {{ unclosed", "", FireSource{Now: time.Now()}); err == nil {
		t.Error("expected parse error")
	}
}

func TestRender_BadTemplateExecErrors(t *testing.T) {
	// Reference an unknown function — execute-time error.
	if _, err := renderPrompt(`{{ .Now | nonexistent }}`, "", FireSource{Now: time.Now()}); err == nil {
		t.Error("expected execute error")
	}
}

// ── baseName helper (white-box) ─────────────────────────────────────

func TestBaseName(t *testing.T) {
	cases := map[string]string{
		"foo.md":           "foo.md",
		"/inbox/foo.md":    "foo.md",
		"/a/b/c/":          "c",
		"":                 "",
		"single":           "single",
		"/with/trailing//": "trailing",
	}
	for in, want := range cases {
		if got := baseName(in); got != want {
			t.Errorf("baseName(%q) = %q, want %q", in, got, want)
		}
	}
}

package automation

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
	"time"
)

// renderPrompt resolves a pre-loaded body into the final string the
// dispatcher delivers to the agent. The Engine has already resolved
// `command:` refs by the time this is called, so body is always the
// raw template text (whether from `prompt:` or from a command file).
//
// Returns an error if body is empty or templating fails — the caller
// writes a receipt with OK=false and a "render: …" error rather than
// dispatching garbage.
//
// Templating is Go's text/template with a small helper set:
//
//   - {{ .Now }}              — wall time of the fire
//   - {{ .Now | date "..." }} — strftime-style format
//   - {{ .Missed }}           — catch-up slot count (0 for on-time)
//   - {{ .LastFired }}        — previous successful fire's scheduled
//   - {{ .Scheduled }}        — the slot this fire covers
//   - {{ .Reason }}           — "on-time" / "catchup" / fs labels
//   - {{ .Event.Path }}       — fs only; first event path
//   - {{ .Event.Name }}       — fs only; basename of Path
//   - {{ .Event.Paths }}      — fs only; full list
//   - {{ .Event.Kind }}       — fs only; create | write | rename
//
// The signature retains the trailing unused `command` arg for
// backward-compat with the Phase 2 tests that pass it; the engine no
// longer relies on it.
func renderPrompt(prompt, command string, src FireSource) (string, error) {
	_ = command // retained for test signature; Engine resolves refs upstream
	body := prompt
	if body == "" {
		return "", fmt.Errorf("automation has neither prompt nor command body")
	}

	tmpl, err := template.New("automation").
		Option("missingkey=zero").
		Funcs(template.FuncMap{
			"date": dateFormat,
		}).
		Parse(body)
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, renderContextFor(src)); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}
	return buf.String(), nil
}

// renderContext is the data the prompt template sees. Public for
// future external renderers (e.g. a dry-run preview command); the
// engine itself constructs it via renderContextFor.
type renderContext struct {
	Now       time.Time
	Scheduled time.Time
	Missed    int
	LastFired time.Time
	Reason    string
	Event     *renderEvent
}

type renderEvent struct {
	Path  string
	Name  string
	Paths []string
	Kind  string
}

func renderContextFor(src FireSource) renderContext {
	c := renderContext{
		Now:       src.Now,
		Scheduled: src.Scheduled,
		Missed:    src.Missed,
		LastFired: src.LastFired,
		Reason:    src.Reason,
	}
	if src.Kind == "fs" && len(src.EventPaths) > 0 {
		c.Event = &renderEvent{
			Path:  src.EventPaths[0],
			Name:  baseName(src.EventPaths[0]),
			Paths: src.EventPaths,
			Kind:  src.EventKind,
		}
	}
	return c
}

// dateFormat is the {{ .Now | date "2006-01-02" }} helper. Accepts
// the same layout strings as time.Time.Format — calling it "date"
// matches sprig's idiom, which authors are most likely to know.
func dateFormat(layout string, t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(layout)
}

// baseName is filepath.Base without importing filepath (which would
// drag a tiny extra closure into render.go's import set). Trims the
// trailing slash and returns the last path component.
func baseName(p string) string {
	for len(p) > 0 && p[len(p)-1] == '/' {
		p = p[:len(p)-1]
	}
	if i := strings.LastIndex(p, "/"); i >= 0 {
		return p[i+1:]
	}
	return p
}

package automation

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
	"time"
)

// renderPrompt resolves an automation's body into the final string
// the dispatcher delivers to the agent. Returns an error if the body
// is empty (validation should have caught it) or if templating
// fails — the caller writes a receipt with OK=false and a "render: …"
// error message rather than dispatching garbage.
//
// Templating is Go's text/template with a small helper set:
//
//   - {{ .Now }}           — wall time of the fire
//   - {{ .Now | date "..." }} — strftime-style format (Go-layout under the hood)
//   - {{ .Missed }}        — count of catch-up slots collapsed (0 for on-time)
//   - {{ .LastFired }}     — previous successful fire's scheduled time
//   - {{ .Scheduled }}     — the slot this fire covers
//   - {{ .Reason }}        — "on-time" / "catchup" / fs-event labels
//   - {{ .Event.Path }}    — fs only (Phase 3); first event path
//   - {{ .Event.Name }}    — fs only; basename of Path
//   - {{ .Event.Paths }}   — fs only; full list (debounce coalesces)
//   - {{ .Event.Kind }}    — fs only; create | write | rename
//
// Phase 2 only populates the cron fields; Event is nil and the helper
// won't crash if a template references it (text/template's missingkey
// behaviour is handled by Option below).
//
// The template fixture for `command:` refs is read by the caller and
// passed in via body — this function doesn't touch the filesystem.
func renderPrompt(prompt, command string, src FireSource) (string, error) {
	body := prompt
	if body == "" && command != "" {
		// command: ref. Phase 2 stub — Phase 4 will load
		// spwn/commands/<name>.md from disk. For now, surface a
		// clear-enough placeholder so the engine can still fire and
		// integration tests catch the missing-file path.
		// (Fully wired once the project Root is plumbed through.)
		body = fmt.Sprintf("[command/%s — body loading deferred to phase 4]", strings.TrimPrefix(command, "command/"))
	}
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

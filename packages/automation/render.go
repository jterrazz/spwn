package automation

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"text/template"
	"time"
)

var (
	templateCache   = make(map[string]*template.Template)
	templateCacheMu sync.RWMutex
)

var templateFuncs = template.FuncMap{
	"date": dateFormat,
}

func getOrParseTemplate(body string) (*template.Template, error) {
	key := templateKey(body)
	templateCacheMu.RLock()
	if t, ok := templateCache[key]; ok {
		templateCacheMu.RUnlock()
		return t, nil
	}
	templateCacheMu.RUnlock()

	t, err := template.New("automation").
		Option("missingkey=zero").
		Funcs(templateFuncs).
		Parse(body)
	if err != nil {
		return nil, err
	}
	templateCacheMu.Lock()
	templateCache[key] = t
	templateCacheMu.Unlock()
	return t, nil
}

func templateKey(body string) string {
	sum := sha256.Sum256([]byte(body))
	return hex.EncodeToString(sum[:])
}

func renderPrompt(prompt, command string, src FireSource) (string, error) {
	_ = command
	body := prompt
	if body == "" {
		return "", fmt.Errorf("automation has neither prompt nor command body")
	}

	tmpl, err := getOrParseTemplate(body)
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, renderContextFor(src)); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}
	return buf.String(), nil
}

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

func dateFormat(layout string, t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(layout)
}

func baseName(p string) string {
	for len(p) > 0 && p[len(p)-1] == '/' {
		p = p[:len(p)-1]
	}
	if i := strings.LastIndex(p, "/"); i >= 0 {
		return p[i+1:]
	}
	return p
}

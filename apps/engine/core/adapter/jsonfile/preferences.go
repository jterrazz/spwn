package jsonfile

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"path"
	"slices"

	"github.com/jterrazz/studio-station/apps/engine/core/contract"
	"github.com/jterrazz/studio-station/apps/engine/core/store"
)

const (
	FileName = "preferences.json"
	Version  = 1
)

type Preferences struct {
	files  Files
	dir    string
	groups []string
}

var (
	_ contract.Preferences = (*Preferences)(nil)
)

func NewPreferences(files Files, dir string, groups []string) *Preferences {
	// invariant: groups are every group station knows, in the order the file
	// writes them; a key no group names is dropped at the next write.
	return &Preferences{files: files, dir: dir, groups: slices.Clone(groups)}
}

func (p *Preferences) File() string { return path.Join(p.dir, FileName) }

func (p *Preferences) Read(group string, v any) error {
	doc, err := p.load()
	if err != nil {
		return err
	}
	raw, ok := doc[group]
	if !ok {
		return nil
	}
	if err := json.Unmarshal(raw, v); err != nil {
		return &store.UnreadableError{Path: p.File(), Cause: err}
	}
	return nil
}

func (p *Preferences) Write(group string, v any) error {
	doc, err := p.load()
	if err != nil {
		return err
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if empty(raw) {
		delete(doc, group)
	} else {
		doc[group] = raw
	}
	return p.save(doc)
}

func (p *Preferences) load() (map[string]json.RawMessage, error) {
	doc := map[string]json.RawMessage{}
	data, err := p.files.ReadFile(p.File())
	if errors.Is(err, fs.ErrNotExist) {
		return doc, nil
	}
	if err != nil {
		return nil, &store.UnreadableError{Path: p.File(), Cause: err}
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, &store.UnreadableError{Path: p.File(), Cause: err, Line: store.LineOf(data, err)}
	}
	var version int
	if raw, ok := doc["v"]; ok {
		if err := json.Unmarshal(raw, &version); err != nil {
			return nil, &store.UnreadableError{Path: p.File(), Cause: err}
		}
	}
	if err := store.Current(version, Version); err != nil {
		return nil, &store.UnreadableError{Path: p.File(), Cause: err}
	}
	return doc, nil
}

func (p *Preferences) save(doc map[string]json.RawMessage) error {
	var out bytes.Buffer
	out.WriteString("{")
	first := true
	for _, g := range p.groups {
		raw, ok := doc[g]
		if !ok {
			continue
		}
		if !first {
			out.WriteString(",")
		}
		first = false
		key, _ := json.Marshal(g)
		out.WriteString("\n  ")
		out.Write(key)
		out.WriteString(": ")
		if err := json.Indent(&out, raw, "  ", "  "); err != nil {
			return err
		}
	}
	if !first {
		out.WriteString("\n")
	}
	out.WriteString("}\n")
	if err := p.files.MkdirAll(p.dir, 0o700); err != nil {
		return &store.UnwritableError{Path: p.File(), Cause: err}
	}
	if err := p.files.WriteFile(p.File(), out.Bytes(), 0o600); err != nil {
		return &store.UnwritableError{Path: p.File(), Cause: err}
	}
	return nil
}

func empty(raw []byte) bool {
	switch string(raw) {
	case "null", `""`, "0", "false", "{}", "[]":
		return true
	}
	return false
}

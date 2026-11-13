package fake

import (
	"context"
	"encoding/json"
	"maps"
	"slices"
	"sync"

	"github.com/jterrazz/studio-station/apps/engine/core/contract"
	"github.com/jterrazz/studio-station/apps/engine/core/store"
)

type Preferences struct {
	// invariant: the preferences of one machine, kept in groups handed by the
	// caller (a Mac of the spec world) or its own; the lock is held while they
	// are read or written.
	kept  map[string]json.RawMessage
	owner string
	mu    sync.Locker
	// value: the Mac's files as text, by name, when the preferences are a
	// world Mac's: preferences.json as edited by hand, settings.json as v2
	// left it.
	files map[string]string
	// value: a group's retired name, and the name it is read under now.
	renamed map[string]string
}

const (
	shownFile   = "~/.config/station/preferences.json"
	editedFile  = "preferences.json"
	retiredFile = "settings.json"
)

var (
	_ contract.Preferences = (*Preferences)(nil)
	_ store.Migrator       = (*Preferences)(nil)
)

func NewPreferences() *Preferences {
	return PreferencesOver(map[string]json.RawMessage{}, "this machine", &sync.Mutex{})
}

func PreferencesOver(groups map[string]json.RawMessage, owner string, mu sync.Locker) *Preferences {
	return &Preferences{kept: groups, owner: owner, mu: mu}
}

func (p *Preferences) groups() map[string]json.RawMessage { return p.kept }

func (p *Preferences) Over(files map[string]string) *Preferences {
	p.files = files
	return p
}

func (p *Preferences) Renaming(retired, now string) *Preferences {
	if p.renamed == nil {
		p.renamed = map[string]string{}
	}
	p.renamed[retired] = now
	return p
}

// invariant: a file edited by hand is read before any group: once it reads,
// its groups are the preferences; one that does not read stops every call
// and is left byte for byte as it is. A group under its retired name is
// kept under its new one.
func (p *Preferences) load() error {
	defer p.rename()
	text, ok := p.files[editedFile]
	if !ok {
		return nil
	}
	doc := map[string]json.RawMessage{}
	if err := json.Unmarshal([]byte(text), &doc); err != nil {
		return &store.UnreadableError{Path: p.File(), Cause: err, Line: store.LineOf([]byte(text), err)}
	}
	clear(p.kept)
	maps.Copy(p.kept, doc)
	delete(p.files, editedFile)
	return nil
}

func (p *Preferences) rename() {
	for retired, now := range p.renamed {
		raw, ok := p.kept[retired]
		if !ok {
			continue
		}
		if _, kept := p.kept[now]; !kept {
			p.kept[now] = raw
		}
		delete(p.kept, retired)
	}
}

func (p *Preferences) MoveRetired(context.Context) (bool, error) {
	defer p.lock()()
	text, ok := p.files[retiredFile]
	if !ok || len(p.kept) > 0 {
		return false, nil
	}
	doc := map[string]json.RawMessage{}
	if err := json.Unmarshal([]byte(text), &doc); err != nil {
		return false, &store.UnreadableError{Path: retiredFile, Cause: err, Line: store.LineOf([]byte(text), err)}
	}
	maps.Copy(p.kept, doc)
	delete(p.files, retiredFile)
	return true, nil
}

func (p *Preferences) lock() func() {
	p.mu.Lock()
	return p.mu.Unlock
}

func (p *Preferences) File() string {
	// value: where the preferences are kept, as a page names it.
	if p.files != nil {
		return shownFile
	}
	return "the preferences of " + p.owner
}

func (p *Preferences) Read(group string, v any) error {
	defer p.lock()()
	if err := p.load(); err != nil {
		return err
	}
	raw, ok := p.groups()[group]
	if !ok {
		return nil
	}
	return json.Unmarshal(raw, v)
}

func (p *Preferences) Write(group string, v any) error {
	defer p.lock()()
	if err := p.load(); err != nil {
		return err
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if slices.Contains([]string{"null", `""`, "0", "false", "{}", "[]"}, string(raw)) {
		delete(p.groups(), group)
		return nil
	}
	p.groups()[group] = raw
	return nil
}

func (p *Preferences) Migrate(context.Context) error { return nil }

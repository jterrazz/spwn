package fake

import (
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
	// world Mac's: preferences.json as edited by hand.
	files map[string]string
}

const (
	shownFile  = "~/.config/station/preferences.json"
	editedFile = "preferences.json"
)

var (
	_ contract.Preferences = (*Preferences)(nil)
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

// invariant: a file edited by hand is read before any group: once it reads,
// its groups are the preferences; one that does not read stops every call
// and is left byte for byte as it is.
func (p *Preferences) load() error {
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

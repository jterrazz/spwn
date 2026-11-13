package claudecode

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/agents"
	"github.com/jterrazz/studio-station/apps/engine/agents/contract"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

const ID agents.App = "claudecode"

const (
	Program = "claude"                  // value: the app's name in the catalogue, the program its agent runs as.
	File    = "~/.claude/settings.json" // value: where its hook lives, as the owner reads it.
)

const CheckedAgainst = "Claude Code 2.0" // value: the version whose hook payloads the recorded ones are.

type Files interface {
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte, perm fs.FileMode) error
	MkdirAll(path string, perm fs.FileMode) error
	Rename(from, to string) error
}

type Hooks struct {
	// invariant: station's hook is one command entry under each event of
	// the hooks key of ~/.claude/settings.json; every other key, and every
	// other entry under hooks, is the owner's and is written back as read.
	files    Files
	settings string
	now      func() time.Time
}

var _ contract.Hooks = (*Hooks)(nil)

func New(files Files, home string, now func() time.Time) *Hooks {
	return &Hooks{files: files, settings: path.Join(home, ".claude", "settings.json"), now: now}
}

func (*Hooks) App() agents.App { return ID }

func (*Hooks) Launch() []string { return []string{Program} }

func command() string {
	return strings.Join(append(slices.Clone(catalog.AgentRecord), string(ID)), " ")
}

func events() []string {
	return []string{"SessionStart", "UserPromptSubmit", "PreToolUse", "PostToolUse", "Notification", "Stop", "SessionEnd"}
}

func matchesUses(event string) bool { return event == "PreToolUse" || event == "PostToolUse" }

type hook struct {
	Type    string `json:"type"`
	Command string `json:"command"`
}

func (h *Hooks) load() (doc map[string]json.RawMessage, groups map[string][]map[string]json.RawMessage, err error) {
	doc = map[string]json.RawMessage{}
	data, err := h.files.ReadFile(h.settings)
	switch {
	case errors.Is(err, fs.ErrNotExist):
	case err != nil:
		return nil, nil, err
	default:
		if err := json.Unmarshal(data, &doc); err != nil {
			return nil, nil, errors.New(h.settings + " is not JSON: " + err.Error())
		}
	}
	groups = map[string][]map[string]json.RawMessage{}
	if raw, ok := doc["hooks"]; ok {
		if err := json.Unmarshal(raw, &groups); err != nil {
			return nil, nil, errors.New(h.settings + "'s hooks are not a map of events: " + err.Error())
		}
	}
	return doc, groups, nil
}

func ours(group map[string]json.RawMessage) bool {
	var hooks []hook
	_ = json.Unmarshal(group["hooks"], &hooks)
	return slices.ContainsFunc(hooks, func(x hook) bool { return x.Command == command() })
}

func (h *Hooks) save(doc map[string]json.RawMessage, groups map[string][]map[string]json.RawMessage) error {
	for event, list := range groups {
		if len(list) == 0 {
			delete(groups, event)
		}
	}
	if len(groups) == 0 {
		delete(doc, "hooks")
	} else {
		raw, err := json.Marshal(groups)
		if err != nil {
			return err
		}
		doc["hooks"] = raw
	}
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	if err := h.files.MkdirAll(path.Dir(h.settings), 0o700); err != nil {
		return err
	}
	staged := h.settings + ".station"
	if err := h.files.WriteFile(staged, append(data, '\n'), 0o600); err != nil {
		return err
	}
	return h.files.Rename(staged, h.settings)
}

func (h *Hooks) Install(context.Context) error {
	doc, groups, err := h.load()
	if err != nil {
		return err
	}
	changed := false
	for _, event := range events() {
		if slices.ContainsFunc(groups[event], ours) {
			continue
		}
		entry, _ := json.Marshal([]hook{{Type: "command", Command: command()}})
		group := map[string]json.RawMessage{"hooks": entry}
		if matchesUses(event) {
			group["matcher"] = json.RawMessage(`"*"`)
		}
		groups[event], changed = append(groups[event], group), true
	}
	if !changed {
		return nil
	}
	return h.save(doc, groups)
}

func (h *Hooks) Inspect(context.Context) (bool, error) {
	_, groups, err := h.load()
	if err != nil {
		return false, err
	}
	for _, event := range events() {
		if !slices.ContainsFunc(groups[event], ours) {
			return false, nil
		}
	}
	return true, nil
}

func (h *Hooks) Remove(context.Context) error {
	doc, groups, err := h.load()
	if err != nil {
		return err
	}
	changed := false
	for event, list := range groups {
		kept := list[:0]
		for _, group := range list {
			if !ours(group) {
				kept = append(kept, group)
				continue
			}
			changed = true
			var hooks []json.RawMessage
			_ = json.Unmarshal(group["hooks"], &hooks)
			hooks = slices.DeleteFunc(hooks, func(raw json.RawMessage) bool {
				var x hook
				return json.Unmarshal(raw, &x) == nil && x.Command == command()
			})
			if len(hooks) > 0 {
				group["hooks"], _ = json.Marshal(hooks)
				kept = append(kept, group)
			}
		}
		groups[event] = kept
	}
	if !changed {
		return nil
	}
	return h.save(doc, groups)
}

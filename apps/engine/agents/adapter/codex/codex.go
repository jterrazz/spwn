package codex

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/agents"
	"github.com/jterrazz/studio-station/apps/engine/agents/contract"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

const ID agents.App = "codex"

const (
	Program = "codex"                // value: the app's name in the catalogue, the program its agent runs as.
	File    = "~/.codex/config.toml" // value: where its hook lives, as the owner reads it.
)

const CheckedAgainst = "codex-cli 0.46" // value: the version whose notify payload the recorded one is.

type Files interface {
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte, perm fs.FileMode) error
	MkdirAll(path string, perm fs.FileMode) error
	Rename(from, to string) error
}

type Hooks struct {
	// invariant: station's hook is the top-level notify key of
	// ~/.codex/config.toml; every other line of the file is the owner's and
	// is written back as read. A notify the owner set to another program is
	// theirs: station does not replace it.
	files Files
	file  string
	now   func() time.Time
}

var _ contract.Hooks = (*Hooks)(nil)

func New(files Files, home string, now func() time.Time) *Hooks {
	return &Hooks{files: files, file: path.Join(home, ".codex", "config.toml"), now: now}
}

func (*Hooks) App() agents.App { return ID }

func (*Hooks) Launch() []string { return []string{Program} }

func notify() []string { return append(append([]string{}, catalog.AgentRecord...), string(ID)) }

func line() string {
	quoted := make([]string, 0, len(notify()))
	for _, w := range notify() {
		quoted = append(quoted, strconv.Quote(w))
	}
	return "notify = [" + strings.Join(quoted, ", ") + "]"
}

func squeezed(s string) string { return strings.Join(strings.Fields(s), "") }

// value: the lines of the file, and where its top-level notify key spans
// (from, to excluded), or -1 when it has none.
func (h *Hooks) load() (lines []string, from, to int, err error) {
	data, err := h.files.ReadFile(h.file)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, -1, -1, nil
	}
	if err != nil {
		return nil, -1, -1, err
	}
	lines = strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
	for i := range lines {
		trimmed := strings.TrimSpace(lines[i])
		if strings.HasPrefix(trimmed, "[") {
			break
		}
		key, _, found := strings.Cut(trimmed, "=")
		if !found || strings.TrimSpace(key) != "notify" {
			continue
		}
		end, depth := i, strings.Count(trimmed, "[")-strings.Count(trimmed, "]")
		for depth > 0 && end+1 < len(lines) {
			end++
			depth += strings.Count(lines[end], "[") - strings.Count(lines[end], "]")
		}
		return lines, i, end + 1, nil
	}
	return lines, -1, -1, nil
}

func isOurs(lines []string, from, to int) bool {
	return from >= 0 && squeezed(strings.Join(lines[from:to], " ")) == squeezed(line())
}

func (h *Hooks) save(lines []string) error {
	if err := h.files.MkdirAll(path.Dir(h.file), 0o700); err != nil {
		return err
	}
	text := strings.Join(lines, "\n")
	if text != "" {
		text += "\n"
	}
	staged := h.file + ".station"
	if err := h.files.WriteFile(staged, []byte(text), 0o600); err != nil {
		return err
	}
	return h.files.Rename(staged, h.file)
}

func (h *Hooks) Install(context.Context) error {
	lines, from, to, err := h.load()
	switch {
	case err != nil:
		return err
	case isOurs(lines, from, to):
		return nil
	case from >= 0:
		return errors.New(h.file + " already notifies another program: station's hook would replace it")
	}
	return h.save(append([]string{line()}, lines...))
}

func (h *Hooks) Inspect(context.Context) (bool, error) {
	lines, from, to, err := h.load()
	return err == nil && isOurs(lines, from, to), err
}

func (h *Hooks) Remove(context.Context) error {
	lines, from, to, err := h.load()
	if err != nil || !isOurs(lines, from, to) {
		return err
	}
	return h.save(append(lines[:from:from], lines[to:]...))
}

type payload struct {
	Type   string   `json:"type"`
	Thread string   `json:"thread-id"`
	Turn   string   `json:"turn-id"`
	Cwd    string   `json:"cwd"`
	Inputs []string `json:"input-messages"`
	Last   string   `json:"last-assistant-message"`
}

func (h *Hooks) Event(_ context.Context, data []byte) (agents.Event, bool) {
	var p payload
	if json.Unmarshal(data, &p) != nil || p.Type != "agent-turn-complete" {
		return agents.Event{}, false
	}
	session := p.Thread
	if session == "" {
		session = p.Turn
	}
	if session == "" {
		return agents.Event{}, false
	}
	e := agents.Event{Kind: agents.TurnDone, App: ID, Session: session, Folder: p.Cwd, At: h.now(), Message: p.Last}
	if len(p.Inputs) > 0 {
		e.Prompt = p.Inputs[0]
	}
	return e, true
}

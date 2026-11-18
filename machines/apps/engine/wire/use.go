package wire

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"path/filepath"
	"regexp"
	"time"

	"github.com/spf13/cobra"

	model "github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
	usecmd "github.com/jterrazz/studio-station/apps/engine/surface/cli/use"
)

// value: each terminal's choice, one file per STATION_TERMINAL under
// <state>/use (SPEC §8.2).
type terminalUses struct {
	fs       local.FS
	clock    local.Clock
	dir      string
	terminal string
}

// invariant: the id names a file, so one that is not a plain word is no
// block: it never reaches outside the folder.
var terminalID = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

func (m machine) uses() *terminalUses {
	id := m.vars["STATION_TERMINAL"]
	if !terminalID.MatchString(id) {
		id = ""
	}
	return &terminalUses{fs: m.sys.FS, clock: m.sys.Clock, dir: filepath.Join(m.paths.State, "use"), terminal: id}
}

func (u *terminalUses) Block() bool    { return u.terminal != "" }
func (u *terminalUses) Now() time.Time { return u.clock.Now() }

func (u *terminalUses) file() string { return filepath.Join(u.dir, u.terminal) }

func (u *terminalUses) Read() (kit.Choice, bool) {
	if !u.Block() {
		return kit.Choice{}, false
	}
	c, ok := u.read(u.file())
	if ok && u.Now().Sub(c.Used) >= kit.ForgottenAfter {
		_ = u.fs.Remove(u.file())
		return kit.Choice{}, false
	}
	return c, ok
}

func (u *terminalUses) read(path string) (kit.Choice, bool) {
	data, err := u.fs.ReadFile(path)
	if err != nil {
		return kit.Choice{}, false
	}
	var c kit.Choice
	if json.Unmarshal(data, &c) != nil {
		return kit.Choice{}, false
	}
	return c, true
}

func (u *terminalUses) Write(c kit.Choice) error {
	if !u.Block() {
		return nil
	}
	if c.Machine == "" && c.Previous == "" {
		if err := u.fs.Remove(u.file()); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		return nil
	}
	c.Used = u.Now()
	return u.write(u.file(), c)
}

func (u *terminalUses) write(path string, c kit.Choice) error {
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}
	if err := u.fs.MkdirAll(u.dir, 0o700); err != nil {
		return err
	}
	return u.fs.WriteFile(path, append(data, '\n'), 0o600)
}

func (u *terminalUses) Gone(machine string, at time.Time) {
	entries, err := u.fs.ReadDir(u.dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		path := filepath.Join(u.dir, e.Name())
		c, ok := u.read(path)
		if !ok {
			continue
		}
		if c.Previous == machine {
			c.Previous = ""
		}
		if c.Machine == machine {
			c.Gone = at
		}
		_ = u.write(path, c)
	}
}

// value: whether a machine of that name is known: this Mac's records, else
// the other Macs' lists; a Mac that does not answer leaves it untold.
func (m machine) known(ctx context.Context, name string) (bool, error) {
	if _, known := m.locate(name); known {
		return true, nil
	}
	_, err := m.holder(ctx, name)
	var unknown *model.UnknownMachineError
	if errors.As(err, &unknown) {
		return false, nil
	}
	return err == nil, err
}

func useCommand(env kit.Env, m machine) *cobra.Command {
	return usecmd.Command(usecmd.Deps{ThisMachine: m.where, Names: m.machines(false).Names}, env)
}

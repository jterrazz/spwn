package tailscale

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"path"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/settings"
	"github.com/jterrazz/studio-station/apps/engine/settings/contract"
)

type Files interface {
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte, perm fs.FileMode) error
	MkdirAll(path string, perm fs.FileMode) error
	Remove(path string) error
	Chmod(path string, perm fs.FileMode) error
	Readlink(path string) (string, error)
}

type Options struct {
	// value: the user's bin, where the command goes, and how it is shown.
	Bin, Shown string
	// value: the App Store app's CLI inside its bundle; "" for any other
	// variant, which brings its own command.
	AppStoreCLI func() string
	LookPath    func(name string) (string, bool)
}

type CLI struct {
	files Files
	o     Options
}

var _ contract.Applier = (*CLI)(nil)

func New(files Files, o Options) *CLI { return &CLI{files: files, o: o} }

func (*CLI) Kind() catalog.EffectKind { return catalog.ProcTailscaleCLI.Kind() }

func (c *CLI) at() string { return path.Join(c.o.Bin, catalog.TailnetCLI) }

func script(cli string) []byte { return []byte("#!/bin/sh\nexec " + cli + " \"$@\"\n") }

// value: what stands where the command goes: nothing, a link, station's
// script, or a file of someone else's.
type found int

const (
	nothing found = iota
	link
	ours
	theirs
)

// invariant: the App Store app's CLI answers only when its bundle's
// executable runs directly (a symlink to it crashes), so PATH gets a
// two-line script, and only where nothing else answers: a symlink is
// replaced, the script's own bytes are adopted, another tailscale on PATH
// or a file station did not write is never touched.
func (c *CLI) read(cli string) (what found, data []byte) {
	at := c.at()
	if _, err := c.files.Readlink(at); err == nil {
		return link, nil
	}
	data, err := c.files.ReadFile(at)
	switch {
	case err != nil:
		return nothing, nil
	case bytes.Equal(data, script(cli)):
		return ours, data
	}
	return theirs, data
}

func (c *CLI) Observe(_ context.Context, _ catalog.Setting) (settings.Observation, error) {
	o := settings.Observation{Where: c.o.Shown}
	cli := c.o.AppStoreCLI()
	if cli == "" {
		o.State, o.Detail = catalog.StateOK, "the app brings its own command"
		return o, nil
	}
	what, data := c.read(cli)
	o.Digest = string(data)
	switch what {
	case ours:
		o.State, o.Detail = catalog.StateOK, c.o.Shown
	case theirs:
		o.State, o.Detail = catalog.StateOK, "a file station did not write, left alone"
	case link:
		o.State, o.Detail = catalog.StateAbsent, "a link to the app crashes: station writes a script"
	case nothing:
		o.State = catalog.StateAbsent
		if other, ok := c.o.LookPath(catalog.TailnetCLI); ok && other != c.at() {
			o.State, o.Detail = catalog.StateOK, "another tailscale answers: "+other
		}
	}
	return o, nil
}

func (c *CLI) Apply(ctx context.Context, it catalog.Setting, _ map[string]string) (settings.Change, error) {
	o, err := c.Observe(ctx, it)
	if err != nil || o.Present() {
		detail := o.Detail
		if detail == c.o.Shown {
			detail = "already there"
		}
		return settings.Change{Steps: []settings.ChangeStep{{Label: c.o.Shown, State: catalog.StateOK, Detail: detail}}}, err
	}
	at := c.at()
	if err := c.files.Remove(at); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return settings.Change{}, err
	}
	if err := c.files.MkdirAll(c.o.Bin, 0o755); err != nil {
		return settings.Change{}, err
	}
	if err := c.files.WriteFile(at, script(c.o.AppStoreCLI()), 0o755); err != nil {
		return settings.Change{}, err
	}
	if err := c.files.Chmod(at, 0o755); err != nil {
		return settings.Change{}, err
	}
	return settings.Change{Steps: []settings.ChangeStep{{Label: c.o.Shown, State: catalog.StateOK, Detail: "written"}}}, nil
}

func (c *CLI) Undo(_ context.Context, _ catalog.Setting) (settings.Change, error) {
	if cli := c.o.AppStoreCLI(); cli != "" {
		if what, _ := c.read(cli); what == ours {
			if err := c.files.Remove(c.at()); err != nil {
				return settings.Change{}, err
			}
			return settings.Change{Steps: []settings.ChangeStep{{Label: c.o.Shown, State: catalog.StateOK, Detail: "removed"}}}, nil
		}
	}
	return settings.Change{Steps: []settings.ChangeStep{{Label: c.o.Shown, State: catalog.StateOK, Detail: "nothing of station's to remove"}}}, nil
}

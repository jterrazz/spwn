package zshblock

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io/fs"
	"strings"

	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/settings"
	"github.com/jterrazz/studio-station/src/settings/contract"
)

type Files interface {
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte, perm fs.FileMode) error
	Stat(path string) (fs.FileInfo, error)
}

// value: the owner's ~/.zshrc, read whole, and edited whole under the lock
// every writer of it takes: edit gets its text and answers the text to
// write.

type Zshrc interface {
	Read(ctx context.Context) (string, error)
	Edit(ctx context.Context, edit func(text string) (string, error)) error
}

type Options struct {
	// value: the machine the blocks' variables name.
	Home string
	OS   catalog.OS
	Arch catalog.Arch
}

type Applier struct {
	zshrc Zshrc
	o     Options
}

var _ contract.Applier = (*Applier)(nil)

func New(zshrc Zshrc, o Options) *Applier { return &Applier{zshrc: zshrc, o: o} }

func (*Applier) Kind() catalog.EffectKind { return catalog.EffectShell }

type rendered struct{ id, body string }

func (a *Applier) of(it catalog.Setting) ([]rendered, error) {
	s, ok := it.Effect.(catalog.ShellBlocks)
	if !ok || len(s.Blocks) == 0 {
		return nil, errors.New(it.Name + " is not shell blocks station writes")
	}
	var out []rendered
	for _, b := range s.Blocks {
		body, err := catalog.Item{}.RenderShell(b, catalog.ShellVars{Home: a.o.Home, OS: a.o.OS, Arch: a.o.Arch})
		if err != nil {
			return nil, err
		}
		out = append(out, rendered{b.ID, body})
	}
	return out, nil
}

func name(id string) string { return "~/.zshrc [" + id + "]" }

func where(blocks []rendered) string {
	ids := make([]string, len(blocks))
	for i, b := range blocks {
		ids[i] = b.id
	}
	return "~/.zshrc [" + strings.Join(ids, ", ") + "]"
}

func (a *Applier) Observe(ctx context.Context, it catalog.Setting) (settings.Observation, error) {
	blocks, err := a.of(it)
	if err != nil {
		return settings.Observation{}, err
	}
	text, err := a.zshrc.Read(ctx)
	if err != nil {
		return settings.Observation{}, err
	}
	// invariant: the fingerprint is the text between the markers of each
	// block there, NUL between two, so a lone block's is its text's alone,
	// the fingerprint the setting's file of functions once had.
	o := settings.Observation{Where: where(blocks)}
	var found, same int
	var diffs, there []string
	for _, b := range blocks {
		have, state := settings.PackageBlock(text, b.id)
		if state != settings.BlockAbsent {
			found++
			there = append(there, have)
		}
		if state == settings.BlockHere && have == b.body {
			same++
			continue
		}
		diffs = append(diffs, settings.UnifiedDiff(b.body, have, "station's "+name(b.id), name(b.id)))
	}
	o.Diff = strings.Join(diffs, "")
	switch {
	case found == 0:
		o.State = catalog.StateAbsent
	case same == len(blocks):
		o.State, o.Detail = catalog.StateOK, o.Where
	default:
		o.State, o.Detail = catalog.StateDegraded, "differs from station's: station settings apply "+it.Name
	}
	if found > 0 {
		sum := sha256.Sum256([]byte(strings.Join(there, "\x00")))
		o.Digest = "sha256:" + hex.EncodeToString(sum[:])
	}
	return o, nil
}

func (a *Applier) Apply(ctx context.Context, it catalog.Setting, _ map[string]string) (settings.Change, error) {
	blocks, err := a.of(it)
	if err != nil {
		return settings.Change{}, err
	}
	err = a.zshrc.Edit(ctx, func(text string) (string, error) {
		for _, b := range blocks {
			text, _ = settings.UpsertPackageBlock(text, b.id, b.body)
		}
		return text, nil
	})
	if err != nil {
		return settings.Change{}, err
	}
	return settings.Change{Steps: []settings.ChangeStep{{Label: where(blocks), State: catalog.StateOK, Detail: "written"}}}, nil
}

func (a *Applier) Undo(ctx context.Context, it catalog.Setting) (settings.Change, error) {
	// invariant: a block leaves ~/.zshrc only while it is still exactly
	// station's: off never destroys a change of the owner's.
	blocks, err := a.of(it)
	if err != nil {
		return settings.Change{}, err
	}
	var steps []settings.ChangeStep
	err = a.zshrc.Edit(ctx, func(text string) (string, error) {
		steps = nil
		for _, b := range blocks {
			have, state := settings.PackageBlock(text, b.id)
			switch {
			case state == settings.BlockAbsent:
				continue
			case state != settings.BlockHere || have != b.body:
				steps = append(steps, settings.ChangeStep{Label: name(b.id), State: catalog.StateOK, Detail: "yours, kept"})
				continue
			}
			text, _ = settings.RemovePackageBlock(text, b.id)
			steps = append(steps, settings.ChangeStep{Label: name(b.id), State: catalog.StateOK, Detail: "removed"})
		}
		return text, nil
	})
	if err != nil || len(steps) == 0 {
		return settings.Change{}, err
	}
	return settings.Change{Steps: steps}, nil
}

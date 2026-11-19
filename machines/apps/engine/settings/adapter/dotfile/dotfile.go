package dotfile

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io/fs"
	"path"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
	"github.com/jterrazz/studio-station/apps/engine/settings"
	"github.com/jterrazz/studio-station/apps/engine/settings/contract"
)

type Files interface {
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte, perm fs.FileMode) error
	MkdirAll(path string, perm fs.FileMode) error
	Chmod(path string, perm fs.FileMode) error
	Remove(path string) error
}

type Runner interface {
	Output(ctx context.Context, c local.Cmd) (local.Output, error)
}

type Options struct {
	Home string
	// value: station's own copies of the files, the embedded dotfiles.
	Ours fs.FS
}

type Applier struct {
	files Files
	run   Runner
	o     Options
}

var _ contract.Applier = (*Applier)(nil)

func New(files Files, run Runner, o Options) *Applier { return &Applier{files: files, run: run, o: o} }

func (*Applier) Kind() catalog.EffectKind { return catalog.EffectDotfile }

func of(it catalog.Setting) (catalog.Dotfiles, error) {
	d, ok := it.Effect.(catalog.Dotfiles)
	if !ok || len(d.Files) == 0 {
		return catalog.Dotfiles{}, errors.New(it.Name + " is not files station writes")
	}
	return d, nil
}

func (a *Applier) Observe(_ context.Context, it catalog.Setting) (settings.Observation, error) {
	// invariant: the fingerprint covers each file of the setting that is
	// there, in its order: an edit or a removal of one station wrote moves
	// it, and a file the setting gains, not there yet, does not, so a
	// setting otherwise as station last wrote it reads as station's to write.
	d, err := of(it)
	if err != nil {
		return settings.Observation{}, err
	}
	o := settings.Observation{Where: d.Files[0].To}
	sum := sha256.New()
	present, differ := 0, false
	var diffs strings.Builder
	for _, c := range d.Files {
		want, err := fs.ReadFile(a.o.Ours, c.From)
		if err != nil {
			return settings.Observation{}, err
		}
		have, err := a.files.ReadFile(expand(a.o.Home, c.To))
		switch {
		case err == nil:
			present++
			sum.Write([]byte(c.To + "\x00"))
			sum.Write(have)
		case errors.Is(err, fs.ErrNotExist):
		default:
			return settings.Observation{}, err
		}
		if !bytes.Equal(want, have) {
			differ = true
			diffs.WriteString(settings.UnifiedDiff(string(want), string(have), "station's "+c.To, c.To))
		}
	}
	o.Digest, o.Diff = "sha256:"+hex.EncodeToString(sum.Sum(nil)), diffs.String()
	switch {
	case present == len(d.Files) && differ:
		o.State, o.Detail = catalog.StateDegraded, "differs from station's: station settings apply "+it.Name
	case present == len(d.Files):
		o.State, o.Detail = catalog.StateOK, d.Files[0].To
	case present == 0:
		o.State = catalog.StateAbsent
	default:
		o.State, o.Detail = catalog.StateDegraded, "some files are missing"
	}
	return o, nil
}

func (a *Applier) Apply(ctx context.Context, it catalog.Setting, _ map[string]string) (settings.Change, error) {
	d, err := of(it)
	if err != nil {
		return settings.Change{}, err
	}
	var change settings.Change
	for _, c := range d.Files {
		data, err := fs.ReadFile(a.o.Ours, c.From)
		if err != nil {
			return change, err
		}
		mode := c.Mode
		if mode == 0 {
			mode = 0o600
		}
		full := expand(a.o.Home, c.To)
		if err := a.files.MkdirAll(path.Dir(full), 0o750); err != nil {
			return change, err
		}
		if err := a.files.WriteFile(full, data, mode); err != nil {
			return change, err
		}
		if err := a.files.Chmod(full, mode); err != nil {
			return change, err
		}
		change.Steps = append(change.Steps, settings.ChangeStep{Label: c.To, State: catalog.StateOK, Detail: "written"})
	}
	if len(d.Reload) > 0 {
		change.Steps = append(change.Steps, a.reload(ctx, d.Reload)...)
	}
	return change, nil
}

func (a *Applier) Undo(_ context.Context, it catalog.Setting) (settings.Change, error) {
	// invariant: Undo removes a file only while it is still exactly station's:
	// off never destroys a change of the owner's.
	d, err := of(it)
	if err != nil {
		return settings.Change{}, err
	}
	var change settings.Change
	for _, c := range d.Files {
		want, err := fs.ReadFile(a.o.Ours, c.From)
		if err != nil {
			return change, err
		}
		full := expand(a.o.Home, c.To)
		have, err := a.files.ReadFile(full)
		switch {
		case errors.Is(err, fs.ErrNotExist):
			continue
		case err != nil:
			return change, err
		case !bytes.Equal(want, have):
			change.Steps = append(change.Steps, settings.ChangeStep{Label: c.To, State: catalog.StateOK, Detail: "yours, kept"})
			continue
		}
		if err := a.files.Remove(full); err != nil {
			return change, err
		}
		change.Steps = append(change.Steps, settings.ChangeStep{Label: c.To, State: catalog.StateOK, Detail: "removed"})
	}
	return change, nil
}

const reloadBudget = 5 * time.Second

// outside: a reload that fails is no failure: tmux with no server running
// exits non-zero, and the new file is read at the next start.
func (a *Applier) reload(ctx context.Context, argv []string) []settings.ChangeStep {
	args := make([]string, len(argv)-1)
	for i, arg := range argv[1:] {
		args[i] = expand(a.o.Home, arg)
	}
	out, err := a.run.Output(ctx, local.Cmd{Name: argv[0], Args: args, Timeout: reloadBudget})
	if err != nil || out.ExitCode != 0 {
		return nil
	}
	return []settings.ChangeStep{{Label: argv[0], State: catalog.StateOK, Detail: "reloaded"}}
}

func expand(home, p string) string {
	if p == "~" {
		return home
	}
	if rest, ok := strings.CutPrefix(p, "~/"); ok {
		return path.Join(home, rest)
	}
	return p
}

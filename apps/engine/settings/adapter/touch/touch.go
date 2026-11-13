package touch

import (
	"context"
	"errors"
	"io/fs"
	"path"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/settings"
	"github.com/jterrazz/studio-station/apps/engine/settings/contract"
)

type Files interface {
	Stat(path string) (fs.FileInfo, error)
	WriteFile(path string, data []byte, perm fs.FileMode) error
	MkdirAll(path string, perm fs.FileMode) error
	Remove(path string) error
}

type Applier struct {
	files Files
	home  string
}

var _ contract.Applier = (*Applier)(nil)

func New(files Files, home string) *Applier { return &Applier{files: files, home: home} }

func (*Applier) Kind() catalog.EffectKind { return catalog.EffectTouch }

func (a *Applier) at(it catalog.Setting) (shown, full string, err error) {
	t, ok := it.Effect.(catalog.Touch)
	if !ok {
		return "", "", errors.New(it.Name + " is not a file station creates")
	}
	return t.Path, expand(a.home, t.Path), nil
}

func (a *Applier) Observe(_ context.Context, it catalog.Setting) (settings.Observation, error) {
	shown, full, err := a.at(it)
	if err != nil {
		return settings.Observation{}, err
	}
	o := settings.Observation{State: catalog.StateAbsent, Where: shown}
	if _, err := a.files.Stat(full); err == nil {
		o.State, o.Detail, o.Digest = catalog.StateOK, shown, "present"
		return o, nil
	}
	o.Before, o.After = []settings.KeyValue{{Key: shown, Value: "absent"}}, []settings.KeyValue{{Key: shown, Value: "present"}}
	return o, nil
}

func (a *Applier) Apply(_ context.Context, it catalog.Setting, _ map[string]string) (settings.Change, error) {
	shown, full, err := a.at(it)
	if err != nil {
		return settings.Change{}, err
	}
	if _, err := a.files.Stat(full); err == nil {
		return settings.Change{Steps: []settings.ChangeStep{{Label: shown, State: catalog.StateOK, Detail: "already there"}}}, nil
	}
	if err := a.files.MkdirAll(path.Dir(full), 0o750); err != nil {
		return settings.Change{}, err
	}
	if err := a.files.WriteFile(full, nil, 0o600); err != nil {
		return settings.Change{}, err
	}
	return settings.Change{Steps: []settings.ChangeStep{{Label: shown, State: catalog.StateOK, Detail: "created"}}}, nil
}

func (a *Applier) Undo(_ context.Context, it catalog.Setting) (settings.Change, error) {
	shown, full, err := a.at(it)
	if err != nil {
		return settings.Change{}, err
	}
	if err := a.files.Remove(full); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return settings.Change{}, err
	}
	return settings.Change{Steps: []settings.ChangeStep{{Label: shown, State: catalog.StateOK, Detail: "removed"}}}, nil
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

package jsonfile

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"path"
	"sort"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/core/store"
	"github.com/jterrazz/studio-station/apps/engine/settings"
	"github.com/jterrazz/studio-station/apps/engine/settings/contract"
)

const Version = 1

type Files interface {
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte, perm fs.FileMode) error
	MkdirAll(path string, perm fs.FileMode) error
	ReadDir(path string) ([]fs.DirEntry, error)
	Remove(path string) error
}

type Applied struct {
	files Files
	dir   string
}

var (
	_ contract.AppliedStore = (*Applied)(nil)
	_ store.Migrator        = (*Applied)(nil)
)

func NewApplied(files Files, dir string) *Applied { return &Applied{files: files, dir: dir} }

type file struct {
	settings.Applied
	V int `json:"v,omitempty"`
}

func (a *Applied) file(setting string) string { return path.Join(a.dir, setting+".json") }

func (a *Applied) read(at string) (settings.Applied, error) {
	data, err := a.files.ReadFile(at)
	if err != nil {
		return settings.Applied{}, err
	}
	var f file
	if err := json.Unmarshal(data, &f); err != nil {
		return settings.Applied{}, &store.UnreadableError{Path: at, Cause: err}
	}
	if _, err := store.Current(f.V, Version); err != nil {
		return settings.Applied{}, &store.UnreadableError{Path: at, Cause: err}
	}
	return f.Applied, nil
}

func (a *Applied) Get(_ context.Context, setting string) (settings.Applied, error) {
	got, err := a.read(a.file(setting))
	if errors.Is(err, fs.ErrNotExist) {
		return settings.Applied{}, store.ErrNotFound
	}
	return got, err
}

func (a *Applied) Put(_ context.Context, applied settings.Applied) error {
	if err := a.files.MkdirAll(a.dir, 0o700); err != nil {
		return &store.UnwritableError{Path: a.dir, Cause: err}
	}
	data, err := json.MarshalIndent(file{Applied: applied}, "", "  ")
	if err != nil {
		return err
	}
	if err := a.files.WriteFile(a.file(applied.Setting), append(data, '\n'), 0o600); err != nil {
		return &store.UnwritableError{Path: a.file(applied.Setting), Cause: err}
	}
	return nil
}

func (a *Applied) List(context.Context) ([]settings.Applied, error) {
	entries, err := a.files.ReadDir(a.dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, &store.UnreadableError{Path: a.dir, Cause: err}
	}
	var out []settings.Applied
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		got, err := a.read(path.Join(a.dir, e.Name()))
		if err != nil {
			return nil, err
		}
		out = append(out, got)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Setting < out[j].Setting })
	return out, nil
}

func (a *Applied) Delete(_ context.Context, setting string) error {
	if err := a.files.Remove(a.file(setting)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

func (a *Applied) Migrate(context.Context) error {
	entries, err := a.files.ReadDir(a.dir)
	if err != nil {
		return store.OnlyNewer(err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		_, err = a.read(path.Join(a.dir, e.Name()))
		if store.OnlyNewer(err) != nil {
			return err
		}
	}
	return nil
}

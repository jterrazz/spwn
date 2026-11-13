package files

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"path/filepath"

	"github.com/jterrazz/studio-station/apps/engine/machinefile/contract"
)

type writable interface {
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte, perm fs.FileMode) error
	MkdirAll(path string, perm fs.FileMode) error
}

// value: the owner's Machinefiles folder, `~/.config/station/machinefiles`,
// which a Machinefile named by its path is copied into.

type Owned struct {
	fs  writable
	dir string
}

func NewOwned(fsys writable, dir string) Owned { return Owned{fs: fsys, dir: dir} }

func (o Owned) Read(_ context.Context, path string) ([]byte, error) { return o.fs.ReadFile(path) }

func (o Owned) Keep(_ context.Context, name string, body []byte) error {
	at := filepath.Join(o.dir, name+".yaml")
	switch there, err := o.fs.ReadFile(at); {
	case err == nil && bytes.Equal(there, body):
		return nil
	case err == nil:
		return &contract.KeptError{Name: name, Path: at}
	case !errors.Is(err, fs.ErrNotExist):
		return err
	}
	if err := o.fs.MkdirAll(o.dir, 0o700); err != nil {
		return err
	}
	return o.fs.WriteFile(at, body, 0o600)
}

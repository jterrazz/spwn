package tailscale_test

import (
	"io/fs"
	"testing"

	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/settings/adapter/tailscale"
	"github.com/jterrazz/studio-station/src/settings/contract/contracttest"
)

type memory struct{ files map[string][]byte }

func (m *memory) ReadFile(p string) ([]byte, error) {
	if b, ok := m.files[p]; ok {
		return b, nil
	}
	return nil, &fs.PathError{Op: "open", Path: p, Err: fs.ErrNotExist}
}

func (m *memory) WriteFile(p string, b []byte, _ fs.FileMode) error {
	m.files[p] = append([]byte(nil), b...)
	return nil
}

func (*memory) MkdirAll(string, fs.FileMode) error { return nil }
func (*memory) Chmod(string, fs.FileMode) error    { return nil }

func (*memory) Readlink(p string) (string, error) {
	return "", &fs.PathError{Op: "readlink", Path: p, Err: fs.ErrInvalid}
}

func (m *memory) Remove(p string) error {
	if _, ok := m.files[p]; !ok {
		return &fs.PathError{Op: "remove", Path: p, Err: fs.ErrNotExist}
	}
	delete(m.files, p)
	return nil
}

func TestApplier(t *testing.T) {
	contracttest.Applier(t, func(*testing.T) contracttest.Opened {
		files := &memory{files: map[string][]byte{}}
		cli := tailscale.New(files, tailscale.Options{
			Bin: "/home/u/.local/bin", Shown: "~/.local/bin/tailscale",
			AppStoreCLI: func() string { return "/Applications/Tailscale.app/Contents/MacOS/Tailscale" },
			LookPath:    func(string) (string, bool) { return "", false },
		})
		return contracttest.Opened{Applier: cli, Item: catalog.Setting{Name: "tailscale-cli", Effect: catalog.ProcTailscaleCLI}, Edit: func(*testing.T) {
			files.files["/home/u/.local/bin/tailscale"] = []byte("#!/bin/sh\necho mine\n")
		}}
	})
}

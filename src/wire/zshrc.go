package wire

import (
	"context"
	"errors"
	"io/fs"

	"github.com/jterrazz/studio-station/src/platform/local"
)

// value: this machine's ~/.zshrc, read whole and written whole; one that is
// not there yet reads empty, and a write keeps the file's own mode.
type zshrcFile struct {
	fs   local.FS
	path string
}

func (z zshrcFile) Read(context.Context) (string, error) {
	data, err := z.fs.ReadFile(z.path)
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}
	return string(data), err
}

func (z zshrcFile) Write(_ context.Context, text string) error {
	mode := fs.FileMode(0o644)
	if info, err := z.fs.Stat(z.path); err == nil {
		mode = info.Mode().Perm()
	}
	return z.fs.WriteFile(z.path, []byte(text), mode)
}

func (m machine) zshrc() zshrcFile {
	return zshrcFile{fs: m.sys.FS, path: m.zshrcPath()}
}

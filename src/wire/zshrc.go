package wire

import (
	"context"
	"errors"
	"io/fs"
	"path/filepath"

	"github.com/jterrazz/studio-station/src/platform/local"
)

// value: this machine's ~/.zshrc, read whole; one that is not there yet
// reads empty.
type zshrcFile struct {
	fs   local.FS
	path string
	// value: the lock every writer of ~/.zshrc takes, the CLI's, the menu
	// bar's and the engine's alike, in station's state folder.
	lock string
}

func (z zshrcFile) Read(context.Context) (string, error) {
	data, err := z.fs.ReadFile(z.path)
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}
	return string(data), err
}

// invariant: an edit reads ~/.zshrc and writes it back under the one lock
// every station process takes for it, so two writers at once never drop
// each other's block; it lands whole, beside the file a link names, with
// the file's own mode (local.Rewrite), and only when edit changed it.
func (z zshrcFile) Edit(_ context.Context, edit func(text string) (string, error)) error {
	if err := z.fs.MkdirAll(filepath.Dir(z.lock), 0o700); err != nil {
		return err
	}
	release, err := local.Exclusive(z.lock)
	if err != nil {
		return err
	}
	defer release()
	return local.Rewrite(z.fs, z.path, func(old []byte) ([]byte, bool, error) {
		text, err := edit(string(old))
		return []byte(text), err == nil && text != string(old), err
	})
}

func (m machine) zshrc() zshrcFile {
	return zshrcFile{fs: m.sys.FS, path: m.zshrcPath(), lock: filepath.Join(m.paths.State, "zshrc.lock")}
}

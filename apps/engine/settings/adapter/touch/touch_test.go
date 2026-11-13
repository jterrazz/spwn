package touch_test

import (
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/settings/adapter/touch"
	"github.com/jterrazz/studio-station/apps/engine/settings/contract/contracttest"
)

type memory struct{ files map[string][]byte }

func newMemory() *memory { return &memory{files: map[string][]byte{}} }

func (m *memory) ReadFile(p string) ([]byte, error) {
	if b, ok := m.files[p]; ok {
		return b, nil
	}
	return nil, &fs.PathError{Op: "open", Path: p, Err: fs.ErrNotExist}
}

func (m *memory) Stat(p string) (fs.FileInfo, error) {
	if _, ok := m.files[p]; ok {
		return fstest.MapFS{"f": {}}.Stat("f")
	}
	return nil, &fs.PathError{Op: "stat", Path: p, Err: fs.ErrNotExist}
}

func (m *memory) WriteFile(p string, b []byte, _ fs.FileMode) error {
	m.files[p] = append([]byte(nil), b...)
	return nil
}

func (*memory) MkdirAll(string, fs.FileMode) error { return nil }
func (*memory) Chmod(string, fs.FileMode) error    { return nil }

func (m *memory) Remove(p string) error {
	if _, ok := m.files[p]; !ok {
		return &fs.PathError{Op: "remove", Path: p, Err: fs.ErrNotExist}
	}
	delete(m.files, p)
	return nil
}

func TestApplier(t *testing.T) {
	contracttest.Applier(t, func(*testing.T) contracttest.Opened {
		item := catalog.Setting{Name: "quiet-login", Effect: catalog.Touch{Path: "~/.hushlogin"}}
		return contracttest.Opened{Applier: touch.New(newMemory(), "/home/u"), Item: item}
	})
}

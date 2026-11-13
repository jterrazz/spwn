package gitaliases_test

import (
	"context"
	"io/fs"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/settings/adapter/gitaliases"
	"github.com/jterrazz/studio-station/apps/engine/settings/contract/contracttest"
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

const owners = "[user]\n\tname = Owner\n[alias]\n\tco = checkout\n"

func pkg() catalog.Setting {
	c, err := catalog.Builtin()
	if err != nil {
		panic(err)
	}
	it, _ := c.Setting(catalog.GitShortcutsSetting)
	return it
}

func TestApplier(t *testing.T) {
	contracttest.Applier(t, func(*testing.T) contracttest.Opened {
		files := &memory{files: map[string][]byte{"/home/u/.gitconfig": []byte(owners)}}
		edit := func(*testing.T) {
			text := string(files.files["/home/u/.gitconfig"])
			files.files["/home/u/.gitconfig"] = []byte(strings.Replace(text, "-m WIP", "-m mine", 1))
		}
		return contracttest.Opened{Applier: gitaliases.New(files, "/home/u"), Package: pkg(), Edit: edit}
	})
}

func TestTheOwnersOwnConfigStaysAroundStationsBlock(t *testing.T) {
	ctx := context.Background()
	files := &memory{files: map[string][]byte{"/home/u/.gitconfig": []byte(owners)}}
	a := gitaliases.New(files, "/home/u")

	if _, err := a.Apply(ctx, pkg(), nil); err != nil {
		t.Fatal(err)
	}
	applied := string(files.files["/home/u/.gitconfig"])
	if !strings.HasPrefix(applied, owners) || !strings.Contains(applied, "\tfeat = \"!f() { git add . && git commit -m \\\"feat: $*\\\"; }; f\"\n") {
		t.Fatalf("applied:\n%s", applied)
	}
	if _, err := a.Undo(ctx, pkg()); err != nil {
		t.Fatal(err)
	}
	if got := string(files.files["/home/u/.gitconfig"]); got != owners {
		t.Fatalf("undone:\n%s\nwant the owner's file as it was", got)
	}
}

package zshblock_test

import (
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/settings/adapter/zshblock"
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

func (m *memory) Stat(p string) (fs.FileInfo, error) {
	if _, ok := m.files[p]; !ok {
		return nil, &fs.PathError{Op: "stat", Path: p, Err: fs.ErrNotExist}
	}
	return fstest.MapFS{"f": {Data: m.files[p], Mode: 0o644}}.Stat("f")
}

const zshrc = "/home/u/.zshrc"

func TestApplier(t *testing.T) {
	contracttest.Applier(t, func(*testing.T) contracttest.Opened {
		files := &memory{files: map[string][]byte{}}
		item := catalog.Setting{Name: "greetings", Effect: catalog.ShellBlocks{Blocks: []catalog.ShellBlock{{ID: "station:greetings", Zsh: "greet() { echo hi; }\n"}}}}
		edit := func(*testing.T) {
			files.files[zshrc] = []byte("# >>> station:greetings >>>\ngreet() { echo mine; }\n# <<< station:greetings <<<\n")
		}
		return contracttest.Opened{Applier: zshblock.New(files, zshblock.Options{File: zshrc, Home: "/home/u", OS: catalog.OSMacOS, Arch: catalog.ArchARM64}), Item: item, Edit: edit}
	})
}

func TestApplyWritesTheExactBytesAndUndoGivesTheOwnersFileBack(t *testing.T) {
	owner := "export EDITOR=vim\n\n# >>> station >>>\nexport PATH=\"$HOME/.local/bin:$PATH\"\n# <<< station <<<\nalias gs='git status'\n"
	files := &memory{files: map[string][]byte{zshrc: []byte(owner)}}
	item := catalog.Setting{Name: "greetings", Effect: catalog.ShellBlocks{Blocks: []catalog.ShellBlock{
		{ID: "station:greetings", Zsh: "export HELLO={{ home }}\n"},
		{ID: "station:greetings/path", Zsh: "path=({{ home }}/bin $path)\n"},
	}}}
	applier := zshblock.New(files, zshblock.Options{File: zshrc, Home: "/home/u", OS: catalog.OSMacOS, Arch: catalog.ArchARM64})
	if _, err := applier.Apply(t.Context(), item, nil); err != nil {
		t.Fatal(err)
	}
	main := "export EDITOR=vim\n\n# >>> station >>>\nexport PATH=\"$HOME/.local/bin:$PATH\"\n# <<< station <<<\n"
	want := main +
		"# >>> station:greetings >>>\nexport HELLO=/home/u\n# <<< station:greetings <<<\n" +
		"# >>> station:greetings/path >>>\npath=(/home/u/bin $path)\n# <<< station:greetings/path <<<\n" +
		"alias gs='git status'\n"
	if got := string(files.files[zshrc]); got != want {
		t.Fatalf("applied\n%s\nwant\n%s", got, want)
	}
	if _, err := applier.Undo(t.Context(), item); err != nil {
		t.Fatal(err)
	}
	if got := string(files.files[zshrc]); got != owner {
		t.Fatalf("undone\n%s\nwant the owner's file\n%s", got, owner)
	}
}

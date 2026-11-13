package zshblock_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/settings/adapter/zshblock"
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

func (m *memory) Stat(p string) (fs.FileInfo, error) {
	if _, ok := m.files[p]; !ok {
		return nil, &fs.PathError{Op: "stat", Path: p, Err: fs.ErrNotExist}
	}
	return fstest.MapFS{"f": {Data: m.files[p], Mode: 0o644}}.Stat("f")
}

const zshrc = "/home/u/.zshrc"

// value: the owner's ~/.zshrc over memory, an edit written back only when
// it changed the text.
type zshrcIn struct{ m *memory }

func (z zshrcIn) Read(context.Context) (string, error) {
	b, err := z.m.ReadFile(zshrc)
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}
	return string(b), err
}

func (z zshrcIn) Edit(ctx context.Context, edit func(string) (string, error)) error {
	text, err := z.Read(ctx)
	if err != nil {
		return err
	}
	out, err := edit(text)
	if err == nil && out != text {
		err = z.m.WriteFile(zshrc, []byte(out), 0o644)
	}
	return err
}

func TestApplier(t *testing.T) {
	contracttest.Applier(t, func(*testing.T) contracttest.Opened {
		files := &memory{files: map[string][]byte{}}
		item := catalog.Setting{Name: "greetings", Effect: catalog.ShellBlocks{Blocks: []catalog.ShellBlock{{ID: "station:greetings", Zsh: "greet() { echo hi; }\n"}}}}
		edit := func(*testing.T) {
			files.files[zshrc] = []byte("# >>> station:greetings >>>\ngreet() { echo mine; }\n# <<< station:greetings <<<\n")
		}
		return contracttest.Opened{Applier: zshblock.New(zshrcIn{files}, zshblock.Options{Home: "/home/u", OS: catalog.OSMacOS, Arch: catalog.ArchARM64}), Item: item, Edit: edit}
	})
}

func TestApplyWritesTheExactBytesAndUndoGivesTheOwnersFileBack(t *testing.T) {
	owner := "export EDITOR=vim\n\n# >>> station >>>\nexport PATH=\"$HOME/.local/bin:$PATH\"\n# <<< station <<<\nalias gs='git status'\n"
	files := &memory{files: map[string][]byte{zshrc: []byte(owner)}}
	item := catalog.Setting{Name: "greetings", Effect: catalog.ShellBlocks{Blocks: []catalog.ShellBlock{
		{ID: "station:greetings", Zsh: "export HELLO={{ home }}\n"},
		{ID: "station:greetings/path", Zsh: "path=({{ home }}/bin $path)\n"},
	}}}
	applier := zshblock.New(zshrcIn{files}, zshblock.Options{Home: "/home/u", OS: catalog.OSMacOS, Arch: catalog.ArchARM64})
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

func TestALoneBlocksFingerprintIsItsTextsAlone(t *testing.T) {
	// invariant: a setting's record from the file of functions it once
	// wrote matches the block that file's bytes moved into.
	body := "greet() { echo hi; }\n"
	files := &memory{files: map[string][]byte{zshrc: []byte("mine\n# >>> station:greetings >>>\n" + body + "# <<< station:greetings <<<\n")}}
	item := catalog.Setting{Name: "greetings", Effect: catalog.ShellBlocks{Blocks: []catalog.ShellBlock{{ID: "station:greetings", Zsh: body}}}}
	o, err := zshblock.New(zshrcIn{files}, zshblock.Options{Home: "/home/u"}).Observe(t.Context(), item)
	sum := sha256.Sum256([]byte(body))
	if err != nil || o.Digest != "sha256:"+hex.EncodeToString(sum[:]) {
		t.Errorf("fingerprint %q, %v", o.Digest, err)
	}
}

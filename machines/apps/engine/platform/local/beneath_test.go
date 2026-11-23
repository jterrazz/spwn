package local

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

// invariant: a link below base, at the end of the path or on the way to
// it, is refused by every call, and what it links to keeps its mode.
func TestALinkBeneathTheBaseIsNeverFollowed(t *testing.T) {
	base, away := t.TempDir(), t.TempDir()
	if err := os.WriteFile(filepath.Join(away, "f"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(away, filepath.Join(base, "on-the-way")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(away, "f"), filepath.Join(base, "at-the-end")); err != nil {
		t.Fatal(err)
	}
	b := Beneath{}
	for name, err := range map[string]error{
		"lstat through":      func() error { _, err := b.Lstat(base, "on-the-way/f"); return err }(),
		"chmod through":      b.Chmod(base, "on-the-way/f", 0o644),
		"chmod at the end":   b.Chmod(base, "at-the-end", 0o644),
		"mkdir through":      b.Mkdir(base, "on-the-way/d", 0o700, Self()),
		"read through":       func() error { _, err := b.ReadFile(base, "on-the-way/f"); return err }(),
		"read at the end":    func() error { _, err := b.ReadFile(base, "at-the-end"); return err }(),
		"replace through":    b.Replace(base, "on-the-way/f", []byte("y"), 0o600, Self()),
		"replace at the end": b.Replace(base, "at-the-end", []byte("y"), 0o600, Self()),
		"remove through":     b.Remove(base, "on-the-way/f"),
		"remove at the end":  b.Remove(base, "at-the-end"),
		"a name that climbs": b.Chmod(base, "../x", 0o600),
	} {
		if err == nil {
			t.Errorf("%s: no refusal", name)
		}
	}
	if err := b.Chmod(base, "at-the-end", 0o600); !errors.Is(err, ErrLink) {
		t.Errorf("chmod of a link = %v; want ErrLink", err)
	}
	if e, err := b.Lstat(base, "at-the-end"); err != nil || e.Mode&fs.ModeSymlink == 0 {
		t.Errorf("Lstat of a link = %+v, %v; want the link itself", e, err)
	}
	if info, err := os.Stat(filepath.Join(away, "f")); err != nil || info.Mode().Perm() != 0o600 {
		t.Errorf("the target moved: %v, %v", info.Mode(), err)
	}
	if data, err := os.ReadFile(filepath.Join(away, "f")); err != nil || string(data) != "x" {
		t.Errorf("the target was written: %q, %v", data, err)
	}
	if e, err := b.Lstat(base, "at-the-end"); err != nil || e.Mode&fs.ModeSymlink == 0 {
		t.Errorf("the link was replaced: %+v, %v", e, err)
	}
	if _, err := os.Stat(filepath.Join(away, "d")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("a folder was made through the link: %v", err)
	}
}

func TestAFolderIsMadeWithItsModeAndOwner(t *testing.T) {
	base := t.TempDir()
	if err := (Beneath{}).Mkdir(base, "d", 0o700, Self()); err != nil {
		t.Fatal(err)
	}
	e, err := Beneath{}.Lstat(base, "d")
	if err != nil || !e.Mode.IsDir() || e.Mode.Perm() != 0o700 || e.Owner != Self() {
		t.Fatalf("Lstat = %+v, %v; want a folder at 0700, %+v's", e, err, Self())
	}
	if err := (Beneath{}).Mkdir(base, "d", 0o700, Self()); !errors.Is(err, fs.ErrExist) {
		t.Fatalf("a second Mkdir = %v; want ErrExist", err)
	}
}

// invariant: a replace keeps a file's mode and owner, and gives a new one
// perm and its owner.
func TestAReplaceKeepsAFilesModeAndMakesANewOneWithPerm(t *testing.T) {
	base, b := t.TempDir(), Beneath{}
	if err := b.Replace(base, "new", []byte("a"), 0o600, Self()); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Join(base, "new"), 0o400); err != nil {
		t.Fatal(err)
	}
	if err := b.Replace(base, "new", []byte("b"), 0o600, Self()); err != nil {
		t.Fatal(err)
	}
	e, err := b.Lstat(base, "new")
	data, _ := b.ReadFile(base, "new")
	if err != nil || e.Mode.Perm() != 0o400 || e.Owner != Self() || string(data) != "b" {
		t.Fatalf("Lstat = %+v, %v, data %q; want b at 0400, %+v's", e, err, data, Self())
	}
	if left, _ := os.ReadDir(base); len(left) != 1 {
		t.Fatalf("the folder holds %d entries; want the file alone", len(left))
	}
}

// invariant: a folder is removed only while it is empty; a file is.
func TestRemoveTakesAFileAndOnlyAnEmptyFolder(t *testing.T) {
	base := t.TempDir()
	if err := os.MkdirAll(filepath.Join(base, "d", "e"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, "d", "f"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	b := Beneath{}
	if err := b.Remove(base, "d"); err == nil {
		t.Fatal("a folder that holds files was removed")
	}
	for _, rel := range []string{"d/f", "d/e", "d"} {
		if err := b.Remove(base, rel); err != nil {
			t.Fatalf("Remove(%s) = %v", rel, err)
		}
	}
	if _, err := os.Lstat(filepath.Join(base, "d")); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("d is still there: %v", err)
	}
}

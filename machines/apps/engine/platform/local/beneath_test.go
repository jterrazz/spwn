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
		"chown at the end":   b.Chown(base, "at-the-end", Self()),
		"mkdir through":      b.Mkdir(base, "on-the-way/d", 0o700, Self()),
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

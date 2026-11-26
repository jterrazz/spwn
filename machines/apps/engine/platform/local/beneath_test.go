package local

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func opened(t *testing.T, base, rel string) *Folder {
	t.Helper()
	f, err := Beneath{}.Open(base, rel)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(f.Close)
	return f
}

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
	b, f := Beneath{}, opened(t, filepath.Dir(base), filepath.Base(base))
	for name, err := range map[string]error{
		"lstat through":      func() error { _, err := b.Lstat(base, "on-the-way/f"); return err }(),
		"open through":       func() error { _, err := b.Open(base, "on-the-way"); return err }(),
		"mkdir through":      b.Mkdir(base, "on-the-way/d", 0o700, Self()),
		"chmod at the end":   f.ChmodFile("at-the-end", os.Geteuid(), func(fs.FileMode) fs.FileMode { return 0o644 }),
		"read at the end":    func() error { _, _, err := f.ReadFile("at-the-end"); return err }(),
		"replace at the end": f.Replace("at-the-end", Seen{}, []byte("y"), 0o600, Self()),
		"remove at the end":  f.Remove("at-the-end", Seen{}),
		"a name that climbs": func() error { _, err := b.Open(base, "../x"); return err }(),
	} {
		if err == nil {
			t.Errorf("%s: no refusal", name)
		}
	}
	if _, err := b.Open(base, "on-the-way"); !errors.Is(err, ErrLink) {
		t.Errorf("open of a link = %v; want ErrLink", err)
	}
	if e, err := f.Lstat("at-the-end"); err != nil || e.Mode&fs.ModeSymlink == 0 {
		t.Errorf("the link was replaced: %+v, %v", e, err)
	}
	if info, err := os.Stat(filepath.Join(away, "f")); err != nil || info.Mode().Perm() != 0o600 {
		t.Errorf("the target moved: %v, %v", info.Mode(), err)
	}
	if data, err := os.ReadFile(filepath.Join(away, "f")); err != nil || string(data) != "x" {
		t.Errorf("the target was written: %q, %v", data, err)
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
	base := t.TempDir()
	f := opened(t, filepath.Dir(base), filepath.Base(base))
	if err := f.Replace("new", Seen{}, []byte("a"), 0o600, Self()); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Join(base, "new"), 0o400); err != nil {
		t.Fatal(err)
	}
	if err := f.Replace("new", Seen{Data: []byte("a"), There: true}, []byte("b"), 0o600, Self()); err != nil {
		t.Fatal(err)
	}
	data, e, err := f.ReadFile("new")
	if err != nil || e.Mode.Perm() != 0o400 || e.Owner != Self() || string(data) != "b" {
		t.Fatalf("ReadFile = %q, %+v, %v; want b at 0400, %+v's", data, e, err, Self())
	}
	if left, _ := os.ReadDir(base); len(left) != 1 {
		t.Fatalf("the folder holds %d entries; want the file alone", len(left))
	}
}

// invariant: a file is removed; a folder removes itself only while it is
// empty.
// invariant: a replace or a remove of a name that no longer holds what
// its writer read is refused, the name as the other writer left it.
func TestAWriteOverANameChangedSinceItsReadIsRefused(t *testing.T) {
	base := t.TempDir()
	f := opened(t, filepath.Dir(base), filepath.Base(base))
	if err := os.WriteFile(filepath.Join(base, "f"), []byte("theirs"), 0o600); err != nil {
		t.Fatal(err)
	}
	stale := Seen{Data: []byte("read"), There: true}
	for name, err := range map[string]error{
		"replace":             f.Replace("f", stale, []byte("ours"), 0o600, Self()),
		"remove":              f.Remove("f", stale),
		"replace as if new":   f.Replace("f", Seen{}, []byte("ours"), 0o600, Self()),
		"replace a gone name": f.Replace("gone", stale, []byte("ours"), 0o600, Self()),
	} {
		if !errors.Is(err, ErrChanged) {
			t.Errorf("%s = %v; want ErrChanged", name, err)
		}
	}
	if data, err := os.ReadFile(filepath.Join(base, "f")); err != nil || string(data) != "theirs" {
		t.Fatalf("f = %q, %v; want theirs", data, err)
	}
	if left, _ := os.ReadDir(base); len(left) != 1 {
		t.Fatalf("left %d names; want f alone, no temporary file", len(left))
	}
}

func TestRemoveTakesAFileAndOnlyAnEmptyFolder(t *testing.T) {
	base := t.TempDir()
	if err := os.MkdirAll(filepath.Join(base, "d"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, "d", "f"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	f := opened(t, base, "d")
	if err := f.RemoveSelf(); !errors.Is(err, ErrNotEmpty) {
		t.Fatalf("a folder that holds a file removed itself: %v", err)
	}
	if err := f.Remove("f", Seen{Data: []byte("x"), There: true}); err != nil {
		t.Fatal(err)
	}
	if err := f.RemoveSelf(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(filepath.Join(base, "d")); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("d is still there: %v", err)
	}
}

// invariant: a held folder keeps every other holder out, even once it is
// renamed, and its writes land in the folder that was held.
func TestAHeldFolderKeepsAnotherHolderOutAcrossARename(t *testing.T) {
	base := t.TempDir()
	if err := os.MkdirAll(filepath.Join(base, "d"), 0o700); err != nil {
		t.Fatal(err)
	}
	f, held, err := Beneath{}.Hold(context.Background(), base, "d", time.Second)
	if err != nil || held {
		t.Fatalf("Hold = %v, %v; want it taken", held, err)
	}
	defer f.Close()
	if err := os.Rename(filepath.Join(base, "d"), filepath.Join(base, "moved")); err != nil {
		t.Fatal(err)
	}
	if _, held, err := (Beneath{}).Hold(context.Background(), base, "moved", 200*time.Millisecond); err != nil || !held {
		t.Fatalf("a second Hold = %v, %v; want it held", held, err)
	}
	if err := os.MkdirAll(filepath.Join(base, "d"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := f.Replace("f", Seen{}, []byte("x"), 0o600, Self()); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(filepath.Join(base, "moved", "f")); err != nil || string(data) != "x" {
		t.Fatalf("the write = %q, %v; want it in the held folder", data, err)
	}
}

func TestOnlyTheFolderJustMadeEmptyAndOursIsFresh(t *testing.T) {
	base := t.TempDir()
	stat := func(name string) (int, *unix.Stat_t) {
		t.Helper()
		if err := os.MkdirAll(filepath.Join(base, name), 0o700); err != nil {
			t.Fatal(err)
		}
		fd, err := unix.Open(filepath.Join(base, name), unix.O_RDONLY|unix.O_DIRECTORY, 0)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = unix.Close(fd) })
		var st unix.Stat_t
		if err := unix.Fstat(fd, &st); err != nil {
			t.Fatal(err)
		}
		return fd, &st
	}
	fd, made := stat("made")
	if err := fresh(fd, made, made); err != nil {
		t.Fatalf("fresh on the empty folder just made = %v; want nil", err)
	}
	other, swapped := stat("swapped")
	if err := fresh(other, made, swapped); !errors.Is(err, errNotMade) {
		t.Fatalf("fresh on another folder = %v; want errNotMade", err)
	}
	if err := os.WriteFile(filepath.Join(base, "made", "planted"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := fresh(fd, made, made); !errors.Is(err, errNotMade) {
		t.Fatalf("fresh on a folder that holds a file = %v; want errNotMade", err)
	}
}

func TestChmodFileChangesOnlyAFileTheOwnerHolds(t *testing.T) {
	base := t.TempDir()
	if err := os.WriteFile(filepath.Join(base, "keys"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := unix.Chmod(filepath.Join(base, "keys"), 0o666); err != nil {
		t.Fatal(err)
	}
	f := opened(t, filepath.Dir(base), filepath.Base(base))
	tighter := func(m fs.FileMode) fs.FileMode { return m &^ 0o022 }
	if err := f.ChmodFile("keys", os.Geteuid()+1, tighter); !errors.Is(err, ErrOwner) {
		t.Fatalf("ChmodFile for another owner = %v; want ErrOwner", err)
	}
	if info, _ := os.Stat(filepath.Join(base, "keys")); info.Mode().Perm() != 0o666 {
		t.Fatalf("mode after a refused chmod = %o; want 666", info.Mode().Perm())
	}
	if err := f.ChmodFile("keys", os.Geteuid(), tighter); err != nil {
		t.Fatal(err)
	}
	if info, _ := os.Stat(filepath.Join(base, "keys")); info.Mode().Perm() != 0o644 {
		t.Fatalf("mode after chmod = %o; want 644", info.Mode().Perm())
	}
}

func TestFoldersAreMadeDownToTheLastAndNeverThroughALink(t *testing.T) {
	base, away := t.TempDir(), t.TempDir()
	if err := (Beneath{}).Folders(base, "a/b/c", 0o755, Self()); err != nil {
		t.Fatal(err)
	}
	if err := (Beneath{}).Folders(base, "a/b/c", 0o755, Self()); err != nil {
		t.Fatalf("Folders over folders there = %v; want them kept", err)
	}
	if info, err := os.Stat(filepath.Join(base, "a", "b", "c")); err != nil || !info.IsDir() || info.Mode().Perm() != 0o755 {
		t.Fatalf("a/b/c = %v, %v; want a folder, 755", info, err)
	}
	if err := os.Symlink(away, filepath.Join(base, "a", "l")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, "a", "f"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{"a/l", "a/l/d", "a/f", "a/f/d"} {
		if err := (Beneath{}).Folders(base, rel, 0o755, Self()); err == nil {
			t.Errorf("Folders(%s) made it; want a refusal", rel)
		}
	}
	if err := (Beneath{}).Folders(base, "a/l/d", 0o755, Self()); !errors.Is(err, ErrLink) {
		t.Errorf("Folders through a link = %v; want ErrLink", err)
	}
	if entries, _ := os.ReadDir(away); len(entries) != 0 {
		t.Errorf("a folder was made through the link: %v", entries)
	}
}

func TestALinkIsMadeReadAndRemovedAsALink(t *testing.T) {
	base, away := t.TempDir(), t.TempDir()
	target := filepath.Join(away, "f")
	if err := os.WriteFile(target, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, "file"), []byte("y"), 0o600); err != nil {
		t.Fatal(err)
	}
	f := opened(t, filepath.Dir(base), filepath.Base(base))
	if err := f.Link("l", target); err != nil {
		t.Fatal(err)
	}
	if err := f.Link("l", "elsewhere"); err != nil {
		t.Fatalf("Link over a link = %v; want it replaced", err)
	}
	if got, err := f.Readlink("l"); err != nil || got != "elsewhere" {
		t.Fatalf("Readlink = %q, %v; want elsewhere", got, err)
	}
	if err := f.Link("file", target); err == nil {
		t.Error("Link over a file replaced it; want a refusal")
	}
	if _, err := f.Readlink("file"); err == nil {
		t.Error("Readlink of a file read it; want a refusal")
	}
	if err := f.RemoveLink("file"); err == nil {
		t.Error("RemoveLink of a file removed it; want a refusal")
	}
	if err := f.RemoveLink("l"); err != nil {
		t.Fatal(err)
	}
	names, err := f.Names()
	if err != nil || len(names) != 1 || names[0] != "file" {
		t.Fatalf("Names = %v, %v; want the file alone", names, err)
	}
	if data, err := os.ReadFile(target); err != nil || string(data) != "x" {
		t.Errorf("the link's target = %q, %v; want it untouched", data, err)
	}
}

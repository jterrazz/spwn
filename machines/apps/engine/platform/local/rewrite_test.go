package local

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"syscall"
	"testing"
)

func replaceWith(data string) func([]byte) ([]byte, bool, error) {
	return func([]byte) ([]byte, bool, error) { return []byte(data), true, nil }
}

func TestARewriteLandsBesideTheFileALinkNamesAndKeepsItsMode(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "dotfiles", "zshrc")
	if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("old\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, ".zshrc")
	if err := os.Symlink("dotfiles/zshrc", link); err != nil {
		t.Fatal(err)
	}
	var seen string
	err := Rewrite(disk{}, link, func(old []byte) ([]byte, bool, error) {
		seen = string(old)
		return []byte("new\n"), true, nil
	})
	if err != nil || seen != "old\n" {
		t.Fatalf("Rewrite = %v, read %q", err, seen)
	}
	to, err := os.Readlink(link)
	if err != nil || to != "dotfiles/zshrc" {
		t.Errorf("the link reads %q, %v; want it kept", to, err)
	}
	info, err := os.Stat(target)
	if data, _ := os.ReadFile(target); err != nil || string(data) != "new\n" || info.Mode().Perm() != 0o600 {
		t.Errorf("the target holds %q with mode %v, %v; want new, 0600", data, info.Mode().Perm(), err)
	}
	if entries, _ := os.ReadDir(filepath.Dir(target)); len(entries) != 1 {
		t.Errorf("left %v beside the target", entries)
	}
}

func TestARewriteOfNoFileMakesItAndOneThatWritesNothingLeavesIt(t *testing.T) {
	at := filepath.Join(t.TempDir(), ".zshrc")
	if err := Rewrite(disk{}, at, func([]byte) ([]byte, bool, error) { return nil, false, nil }); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(at); !os.IsNotExist(err) {
		t.Errorf("a rewrite that wrote nothing made the file: %v", err)
	}
	if err := Rewrite(disk{}, at, replaceWith("x\n")); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(at); err != nil || info.Mode().Perm()&0o600 != 0o600 {
		t.Errorf("made %v, %v", info, err)
	}
}

func oldFile(t *testing.T) (dir, at string) {
	t.Helper()
	dir = t.TempDir()
	at = filepath.Join(dir, ".zshrc")
	if err := os.WriteFile(at, []byte("old\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return dir, at
}

func names(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		out = append(out, e.Name())
	}
	return out
}

type synced struct{ name, holds, target string }

func TestAReplaceSyncsTheNewFileBeforeItsRenameThenItsFolder(t *testing.T) {
	dir, at := oldFile(t)
	var seen []synced
	d := durable{sync: func(f *os.File) error {
		holds, _ := os.ReadFile(f.Name())
		target, _ := os.ReadFile(at)
		seen = append(seen, synced{f.Name(), string(holds), string(target)})
		return f.Sync()
	}}
	if err := d.replace(at, []byte("new\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if len(seen) != 2 {
		t.Fatalf("synced %+v; want the new file, then the folder", seen)
	}
	if file := seen[0]; filepath.Dir(file.name) != dir || file.name == at || file.holds != "new\n" || file.target != "old\n" {
		t.Errorf("first synced %+v; want a file of its own beside the target, whole, while the target is old", file)
	}
	if folder := seen[1]; folder.name != dir || folder.target != "new\n" {
		t.Errorf("then synced %+v; want the folder, once the rename landed", folder)
	}
}

func TestAFailingWriteLeavesTheOldBytesAndNoFileBeside(t *testing.T) {
	dir, at := oldFile(t)
	full := errors.New("no space left on device")
	err := durable{sync: func(*os.File) error { return full }}.replace(at, []byte("new\n"), 0o644)
	if !errors.Is(err, full) {
		t.Fatalf("replace = %v, want the disk's error", err)
	}
	if data, _ := os.ReadFile(at); string(data) != "old\n" {
		t.Errorf("the file holds %q, want its old bytes", data)
	}
	if left := names(t, dir); !slices.Equal(left, []string{".zshrc"}) {
		t.Errorf("left %v beside the file", left)
	}
}

func TestAFileInAFolderStationMayNotWriteInIsRewrittenWhereItStands(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root writes in any folder")
	}
	dir, at := oldFile(t)
	if err := os.Chmod(dir, 0o500); err != nil { //nolint:gosec // value: a folder its owner may read and search, never write in.
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) }) //nolint:gosec // value: the test's folder, emptied by its cleanup.
	if err := Rewrite(disk{}, at, replaceWith("new\n")); err != nil {
		t.Fatalf("Rewrite = %v, want the file overwritten where it stands", err)
	}
	info, err := os.Stat(at)
	if data, _ := os.ReadFile(at); err != nil || string(data) != "new\n" || info.Mode().Perm() != 0o600 {
		t.Errorf("the file holds %q, %v", data, err)
	}
}

func TestARewriteKeepsTheFilesGroup(t *testing.T) {
	dir, at := oldFile(t)
	probe := filepath.Join(dir, "probe")
	if err := os.WriteFile(probe, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	fresh := gidOf(t, probe)
	groups, _ := os.Getgroups()
	other := slices.IndexFunc(groups, func(g int) bool { return g != fresh })
	if other < 0 {
		t.Skip("this user is in one group only")
	}
	if err := os.Chown(at, -1, groups[other]); err != nil {
		t.Fatal(err)
	}
	if err := Rewrite(disk{}, at, replaceWith("new\n")); err != nil {
		t.Fatal(err)
	}
	if got := gidOf(t, at); got != groups[other] {
		t.Errorf("the file's group is %d, want %d kept", got, groups[other])
	}
}

func gidOf(t *testing.T, at string) int {
	t.Helper()
	info, err := os.Stat(at)
	if err != nil {
		t.Fatal(err)
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		t.Fatalf("%s has no owner to read", at)
	}
	return int(stat.Gid)
}

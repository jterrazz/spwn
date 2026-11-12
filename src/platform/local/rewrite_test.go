package local

import (
	"os"
	"path/filepath"
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

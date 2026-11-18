package local

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAFolderMadeAgainUnderItsPathIsAnotherAndARenameKeepsIt(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "vm")
	id := func(p string) FileID {
		t.Helper()
		info, err := os.Stat(p)
		if err != nil {
			t.Fatal(err)
		}
		got, ok := IdentityOf(info)
		if !ok || got == "" {
			t.Fatalf("%s has no identity", p)
		}
		return got
	}
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	first := id(dir)
	if err := os.Mkdir(dir+"-kept", 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(dir); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if again := id(dir); again == first && !strings.HasSuffix(string(first), "@") {
		t.Fatalf("the folder made again reads %s, the identity of the one deleted", again)
	}
	moved := dir + "-moved"
	before := id(dir)
	if err := os.Rename(dir, moved); err != nil {
		t.Fatal(err)
	}
	if after := id(moved); after != before {
		t.Fatalf("the renamed folder reads %s, want %s, its own", after, before)
	}
}

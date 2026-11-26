package platform

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// beside makes <dir>/site and <dir>/outside, the outside holding one file
// with known bytes, and returns both folders.
func beside(t *testing.T) (site, outside string) {
	t.Helper()
	dir := t.TempDir()
	site, outside = filepath.Join(dir, "site"), filepath.Join(dir, "outside")
	for _, d := range []string{site, outside} {
		if err := os.Mkdir(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(outside, "rc"), []byte("untouched\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return site, outside
}

func untouched(t *testing.T, outside string, absent ...string) {
	t.Helper()
	if data, err := os.ReadFile(filepath.Join(outside, "rc")); err != nil || string(data) != "untouched\n" {
		t.Errorf("the outside file was written: %q %v", data, err)
	}
	for _, name := range absent {
		if _, err := os.Lstat(filepath.Join(outside, name)); err == nil {
			t.Errorf("%s was created outside", name)
		}
	}
}

func TestATreeAppendsWritesAndRenamesNothingThroughALinkOut(t *testing.T) {
	site, outside := beside(t)
	for name, target := range map[string]string{
		"runs.jsonl": filepath.Join(outside, "rc"),
		"dangling":   filepath.Join(outside, "not-there"),
		"away":       outside,
	} {
		if err := os.Symlink(target, filepath.Join(site, name)); err != nil {
			t.Fatal(err)
		}
	}
	tree := Within(site)
	defer tree.Close()

	for _, err := range []error{
		tree.AppendFile("runs.jsonl", []byte("$(touch pwned)\n"), 0o644),
		tree.WriteFile("runs.jsonl", []byte("x"), 0o644),
		tree.AppendFile("dangling", []byte("x"), 0o644),
		tree.WriteFile("dangling", []byte("x"), 0o644),
		tree.AppendFile("away/new.jsonl", []byte("x"), 0o644),
		tree.MkdirAll("away/world-states/w1", 0o755),
		tree.WriteFile("mine", []byte("x"), 0o644),
	} {
		if err == nil {
			continue
		}
		if !errors.Is(err, ErrLink) {
			t.Errorf("a refusal through a link does not say it is a link: %v", err)
		}
	}
	if err := tree.Rename("mine", "away/mine"); err == nil {
		t.Error("a rename into a linked folder out was taken")
	}
	untouched(t, outside, "not-there", "new.jsonl", "world-states", "mine")
	if data, err := os.ReadFile(filepath.Join(site, "mine")); err != nil || string(data) != "x" {
		t.Errorf("an honest write inside was refused: %q %v", data, err)
	}
}

func TestATreeLinkRefusalNamesTheLinkAndReadsTheSameWhateverIsOutside(t *testing.T) {
	site, outside := beside(t)
	if err := os.Symlink(filepath.Join(outside, "gone"), filepath.Join(site, "secret.md")); err != nil {
		t.Fatal(err)
	}
	tree := Within(site)
	defer tree.Close()
	_, gone := tree.ReadFile("secret.md")
	if err := os.WriteFile(filepath.Join(outside, "gone"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, there := tree.ReadFile("secret.md")
	if gone == nil || there == nil || gone.Error() != there.Error() {
		t.Fatalf("the refusal tells what is outside:\n gone:  %v\n there: %v", gone, there)
	}
	want := "open secret.md: " + ErrLink.Error()
	if gone.Error() != want {
		t.Fatalf("refusal = %q, want %q", gone, want)
	}
}

func TestATreeAppendNeverWaitsOnAFIFO(t *testing.T) {
	site, _ := beside(t)
	if err := syscall.Mkfifo(filepath.Join(site, "runs.jsonl"), 0o644); err != nil {
		t.Fatal(err)
	}
	tree := Within(site)
	defer tree.Close()
	done := make(chan error, 1)
	go func() { done <- tree.AppendFile("runs.jsonl", []byte("x"), 0o644) }()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("an append to a FIFO was taken")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("an append waited on a FIFO")
	}
}

func TestStateTreeRefusesALinkedStateFolderOut(t *testing.T) {
	site, outside := beside(t)
	if err := os.Symlink(outside, filepath.Join(site, ".spwn")); err != nil {
		t.Fatal(err)
	}
	_, err := WithProjectRoot(site, func() (struct{}, error) {
		tree := StateTree()
		defer tree.Close()
		return struct{}{}, tree.MkdirAll(filepath.Join(LocalStateDir(), "world-states", "w1"), 0o755)
	})
	if !errors.Is(err, ErrLink) {
		t.Fatalf("a world-state folder through a linked .spwn: %v", err)
	}
	untouched(t, outside, "world-states")
}

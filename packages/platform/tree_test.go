package platform

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// treeBeside makes <dir>/project holding a file, a folder and a FIFO, and
// <dir>/outside.md beside it; it returns the Tree of the project and dir.
func treeBeside(t *testing.T) (*Tree, string) {
	t.Helper()
	dir := t.TempDir()
	project := filepath.Join(dir, "project")
	if err := os.MkdirAll(filepath.Join(project, "spwn", "agents", "neo"), 0o755); err != nil {
		t.Fatal(err)
	}
	for at, body := range map[string]string{
		filepath.Join(project, "spwn.yaml"): "version: 1\n",
		filepath.Join(dir, "outside.md"):    "OUTSIDE\n",
	} {
		if err := os.WriteFile(at, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := syscall.Mkfifo(filepath.Join(project, "pipe.md"), 0o644); err != nil {
		t.Fatal(err)
	}
	tree := Within(project)
	t.Cleanup(func() { _ = tree.Close() })
	return tree, dir
}

func TestATreeRefusesEveryPathThatLeavesItByItsNameAlone(t *testing.T) {
	tree, dir := treeBeside(t)
	for _, p := range []string{
		"..",
		"../outside.md",
		"spwn/../../outside.md",
		filepath.Join(dir, "outside.md"),
		"spwn\x00.yaml",
	} {
		if tree.Holds(p) {
			t.Errorf("Holds(%q) = true; it leaves the project", p)
		}
		if _, err := tree.ReadFile(p); !errors.Is(err, ErrOutside) {
			t.Errorf("ReadFile(%q) = %v; want ErrOutside", p, err)
		}
		if _, err := tree.Stat(p); !errors.Is(err, ErrOutside) {
			t.Errorf("Stat(%q) = %v; want ErrOutside", p, err)
		}
		if _, err := tree.ReadDir(p); !errors.Is(err, ErrOutside) {
			t.Errorf("ReadDir(%q) = %v; want ErrOutside", p, err)
		}
	}
	for _, p := range []string{"spwn.yaml", "spwn/agents/../../spwn.yaml", filepath.Join(tree.Dir(), "spwn.yaml")} {
		if data, err := tree.ReadFile(p); err != nil || string(data) != "version: 1\n" {
			t.Errorf("ReadFile(%q) = %q, %v; want the project's file", p, data, err)
		}
	}
}

func TestATreeNeverFollowsALinkOutOfIt(t *testing.T) {
	tree, dir := treeBeside(t)
	at := filepath.Join(tree.Dir(), "spwn", "away.md")
	if err := os.Symlink(filepath.Join(dir, "outside.md"), at); err != nil {
		t.Fatal(err)
	}
	if data, err := tree.ReadFile(at); err == nil {
		t.Errorf("ReadFile through a link out of the tree = %q; want it refused", data)
	}
	if f, err := tree.Open(at); err == nil {
		_ = f.Close()
		t.Errorf("Open through a link out of the tree succeeded; want it refused")
	}
	if tree.IsFile(at) {
		t.Errorf("IsFile through a link out of the tree = true; want it refused")
	}
}

func TestATreeNeverWaitsOnAFIFO(t *testing.T) {
	tree, _ := treeBeside(t)
	at := filepath.Join(tree.Dir(), "pipe.md")
	for name, read := range map[string]func() error{
		"ReadFile": func() error { _, err := tree.ReadFile(at); return err },
		"ReadDir":  func() error { _, err := tree.ReadDir(at); return err },
		"Open":     func() error { _, err := tree.Open(at); return err },
	} {
		got := make(chan error, 1)
		go func() { got <- read() }()
		select {
		case err := <-got:
			if !errors.Is(err, ErrNotAFile) {
				t.Errorf("%s of a FIFO = %v; want ErrNotAFile", name, err)
			}
		case <-time.After(10 * time.Second):
			if w, err := os.OpenFile(at, os.O_WRONLY, 0); err == nil {
				_ = w.Close()
			}
			t.Fatalf("%s waited on a FIFO", name)
		}
	}
}

func TestATreeWalksAsWalkDirDoes(t *testing.T) {
	tree, _ := treeBeside(t)
	var got []string
	err := tree.WalkDir(filepath.Join(tree.Dir(), "spwn"), func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(tree.Dir(), p)
		got = append(got, rel)
		return nil
	})
	want := []string{"spwn", filepath.Join("spwn", "agents"), filepath.Join("spwn", "agents", "neo")}
	if err != nil || len(got) != len(want) || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
		t.Errorf("WalkDir = %v, %v; want %v", got, err, want)
	}
}

func TestATreeOpensAFileSizedByTheFileItReads(t *testing.T) {
	tree, _ := treeBeside(t)
	f, err := tree.Open("spwn.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(f)
	if err != nil || string(body) != "version: 1\n" || info.Size() != int64(len(body)) {
		t.Errorf("Open read %q of %d bytes, %v; want the file and its size", body, info.Size(), err)
	}
}

func TestATreeOfAFolderThatIsNotThereReadsNothing(t *testing.T) {
	tree := Within(filepath.Join(t.TempDir(), "not-there"))
	defer tree.Close()
	if _, err := tree.ReadFile("spwn.yaml"); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("ReadFile = %v; want a missing folder's error", err)
	}
}

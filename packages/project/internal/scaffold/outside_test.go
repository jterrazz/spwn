package scaffold

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

const outsideBody = ".spwn/\n"

func folderBesideOutside(t *testing.T) (project, outside string) {
	t.Helper()
	dir := t.TempDir()
	project = filepath.Join(dir, "project")
	outside = filepath.Join(dir, "outside")
	for _, d := range []string{project, outside} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(outside, "rc"), []byte(outsideBody), 0o644); err != nil {
		t.Fatal(err)
	}
	return project, outside
}

func outsideUntouched(t *testing.T, outside string) {
	t.Helper()
	entries, err := os.ReadDir(outside)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "rc" {
		t.Errorf("the outside folder holds %v; want rc alone", entries)
	}
	if body, err := os.ReadFile(filepath.Join(outside, "rc")); err != nil || string(body) != outsideBody {
		t.Errorf("the outside file = %q, %v; want it untouched", body, err)
	}
}

func TestInitNeverReadsNorAppendsToAGitignoreThatLinksOut(t *testing.T) {
	errs := map[bool]string{}
	for _, there := range []bool{true, false} {
		project, outside := folderBesideOutside(t)
		if err := os.Symlink(filepath.Join(outside, "rc"), filepath.Join(project, ".gitignore")); err != nil {
			t.Fatal(err)
		}
		if !there {
			if err := os.Remove(filepath.Join(outside, "rc")); err != nil {
				t.Fatal(err)
			}
		}
		err := Init(project, Opts{Name: "linked"})
		if err == nil {
			t.Fatalf("Init over a .gitignore linking out (outside there: %v) = nil; want it refused", there)
		}
		errs[there] = err.Error()
		if there {
			outsideUntouched(t, outside)
		} else if _, err := os.Lstat(filepath.Join(outside, "rc")); err == nil {
			t.Errorf("Init created the .gitignore's outside target")
		}
	}
	if errs[true] != errs[false] {
		t.Errorf("Init tells whether the outside file is there:\n there: %s\n gone:  %s", errs[true], errs[false])
	}
}

func TestInitNeverWritesThroughAnAgentsFolderThatLinksOut(t *testing.T) {
	project, outside := folderBesideOutside(t)
	if err := os.Symlink(outside, filepath.Join(project, "agents")); err != nil {
		t.Fatal(err)
	}
	if err := Init(project, Opts{Name: "linked", NoGitignore: true}); err == nil {
		t.Fatal("Init through an agents/ linking out = nil; want it refused")
	}
	outsideUntouched(t, outside)
}

func TestAWorldEditNeverReadsNorWritesASpwnYamlThatLinksOut(t *testing.T) {
	project, outside := folderBesideOutside(t)
	manifest := filepath.Join(project, "spwn.yaml")
	if err := os.Symlink(filepath.Join(outside, "rc"), manifest); err != nil {
		t.Fatal(err)
	}
	if err := AddWorld(manifest, "w", AddWorldOpts{}); err == nil {
		t.Error("AddWorld over a spwn.yaml linking out = nil; want it refused")
	}
	if err := RemoveWorld(manifest, "w"); err == nil {
		t.Error("RemoveWorld over a spwn.yaml linking out = nil; want it refused")
	}
	outsideUntouched(t, outside)
}

func TestAFIFOAtAScaffoldNameNeverHoldsIt(t *testing.T) {
	project, _ := folderBesideOutside(t)
	for _, name := range []string{".gitignore", "spwn.yaml.fifo"} {
		if err := syscall.Mkfifo(filepath.Join(project, name), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got := make(chan [3]error, 1)
	go func() {
		fifo := filepath.Join(project, "spwn.yaml.fifo")
		got <- [3]error{Init(project, Opts{Name: "fifo"}), AddWorld(fifo, "w", AddWorldOpts{}), RemoveWorld(fifo, "w")}
	}()
	select {
	case errs := <-got:
		for i, err := range errs {
			if err == nil {
				t.Errorf("call %d over a FIFO = nil; want it refused", i)
			}
		}
	case <-time.After(10 * time.Second):
		for _, name := range []string{".gitignore", "spwn.yaml.fifo"} {
			if f, err := os.OpenFile(filepath.Join(project, name), os.O_RDWR, 0); err == nil {
				_ = f.Close()
			}
		}
		t.Fatal("a scaffold call waited on a FIFO")
	}
}

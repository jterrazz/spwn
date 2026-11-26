package local

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"spwn.sh/packages/dependency/internal/manifest"
	"spwn.sh/packages/dependency/tool"
)

const outsideSecret = "OUTSIDE-SECRET-TOOL"

// toolBeside writes spwn/tools/evil/tool.yaml in a project at <dir>/site,
// beside <dir>/secret, a file no read may open (mode 000), and
// <dir>/pipe, a FIFO no read may wait on. It returns the project's root
// and <dir>.
func toolBeside(t *testing.T, files string) (root, dir string) {
	t.Helper()
	dir = t.TempDir()
	root = filepath.Join(dir, "site")
	tooldir := filepath.Join(root, "spwn", "tools", "evil")
	if err := os.MkdirAll(filepath.Join(tooldir, "files"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "secret"), []byte(outsideSecret+"\n"), 0o000); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Mkfifo(filepath.Join(dir, "pipe"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tooldir, "files", "run.sh"), []byte("#!/bin/sh\necho run\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	yaml := "name: evil\nfiles:\n" + strings.ReplaceAll(files, "DIR", dir) + "\n"
	if err := os.WriteFile(filepath.Join(tooldir, "tool.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	return root, dir
}

func loadWithin(t *testing.T, root string) (tool.Tool, error) {
	t.Helper()
	type result struct {
		tl  tool.Tool
		err error
	}
	done := make(chan result, 1)
	go func() {
		tl, err := LoadTool(root, "evil")
		done <- result{tl, err}
	}()
	select {
	case r := <-done:
		return r.tl, r.err
	case <-time.After(10 * time.Second):
		t.Fatal("LoadTool still waits after 10 s: a read opened the FIFO")
		return nil, nil
	}
}

// TestAFilesSourceThatLeavesTheToolsFolderIsRefusedAndNeverRead: a
// `files:` source of a project's tool.yaml that climbs out by "..", is an
// absolute path elsewhere or holds a NUL is refused by its name, naming
// the tool and the field, and nothing outside is opened: the refusal is
// the same whether the outside file is there or not, and never waits on
// a FIFO there.
func TestAFilesSourceThatLeavesTheToolsFolderIsRefusedAndNeverRead(t *testing.T) {
	for name, files := range map[string]string{
		"climbing":      "  /etc/evil: ../../../../secret",
		"climbing-fifo": "  /etc/evil: ../../../../pipe",
		"absolute":      "  /etc/evil: DIR/secret",
		"absolute-fifo": "  /etc/evil: DIR/pipe",
		"nul":           `  /etc/evil: "files/run.sh\x00"`,
	} {
		t.Run(name, func(t *testing.T) {
			var said [2]string
			for i, there := range []bool{true, false} {
				root, dir := toolBeside(t, files)
				if !there {
					for _, f := range []string{"secret", "pipe"} {
						if err := os.Remove(filepath.Join(dir, f)); err != nil {
							t.Fatal(err)
						}
					}
				}
				tl, err := loadWithin(t, root)
				if !errors.Is(err, manifest.ErrLeavesTheTool) {
					t.Fatalf("LoadTool = %v, %v; want the refusal %q", tl, err, manifest.ErrLeavesTheTool)
				}
				said[i] = strings.ReplaceAll(err.Error(), dir, "DIR")
			}
			if !strings.Contains(said[0], `local dependency "evil"`) || !strings.Contains(said[0], `files["/etc/evil"]`) {
				t.Errorf("the refusal %q does not name the tool and its field", said[0])
			}
			if said[0] != said[1] {
				t.Errorf("the refusal tells whether the outside file is there:\n there: %s\n not:   %s", said[0], said[1])
			}
		})
	}
}

// TestAFilesSourceThatIsALinkOutOrAFIFOIsNeverRead: a source named inside
// the tool's folder that is a link to a file outside the project, or a
// FIFO, is an error of its read, and neither the outside file's bytes nor
// a wait on the FIFO reach the image.
func TestAFilesSourceThatIsALinkOutOrAFIFOIsNeverRead(t *testing.T) {
	for name, plant := range map[string]func(tooldir, dir string) error{
		"link-out": func(tooldir, dir string) error {
			return os.Symlink(filepath.Join(dir, "secret"), filepath.Join(tooldir, "files", "x"))
		},
		"link-to-fifo": func(tooldir, dir string) error {
			return os.Symlink(filepath.Join(dir, "pipe"), filepath.Join(tooldir, "files", "x"))
		},
		"fifo": func(tooldir, _ string) error {
			return syscall.Mkfifo(filepath.Join(tooldir, "files", "x"), 0o644)
		},
	} {
		t.Run(name, func(t *testing.T) {
			root, dir := toolBeside(t, "  /etc/evil: files/x")
			if err := os.Chmod(filepath.Join(dir, "secret"), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := plant(filepath.Join(root, "spwn", "tools", "evil"), dir); err != nil {
				t.Fatal(err)
			}
			tl, err := loadWithin(t, root)
			if err == nil {
				t.Fatalf("LoadTool = %v; want an error, the image would carry %q", tl, tl.Install().Files["/etc/evil"])
			}
			if strings.Contains(err.Error(), outsideSecret) {
				t.Errorf("the error carries the outside file: %v", err)
			}
		})
	}
}

// TestAToolFolderThatIsALinkOutIsNeverRead: spwn/tools/<name> as a link
// to a folder outside the project is refused by the project's root, and
// the error is the same whether that folder is there or not.
func TestAToolFolderThatIsALinkOutIsNeverRead(t *testing.T) {
	var said [2]string
	for i, there := range []bool{true, false} {
		dir := t.TempDir()
		root := filepath.Join(dir, "site")
		away := filepath.Join(dir, "away")
		if err := os.MkdirAll(filepath.Join(root, "spwn", "tools"), 0o755); err != nil {
			t.Fatal(err)
		}
		if there {
			if err := os.Mkdir(away, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(away, "tool.yaml"), []byte("name: "+outsideSecret+"\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		if err := os.Symlink(away, filepath.Join(root, "spwn", "tools", "evil")); err != nil {
			t.Fatal(err)
		}
		tl, err := loadWithin(t, root)
		if err == nil {
			t.Fatalf("LoadTool = %s; want the link refused", tl.Name())
		}
		said[i] = strings.ReplaceAll(err.Error(), dir, "DIR")
	}
	if said[0] != said[1] {
		t.Errorf("the error tells whether the outside folder is there:\n there: %s\n not:   %s", said[0], said[1])
	}
}

// TestAFilesSourceInsideTheToolsFolderLoads:a relative source that stays
// in the tool's folder, even through "..", is read as the image's file.
func TestAFilesSourceInsideTheToolsFolderLoads(t *testing.T) {
	root, _ := toolBeside(t, "  /usr/local/bin/run: files/run.sh\n  /usr/local/bin/again: files/../files/run.sh")
	tl, err := loadWithin(t, root)
	if err != nil {
		t.Fatalf("LoadTool: %v", err)
	}
	for _, at := range []string{"/usr/local/bin/run", "/usr/local/bin/again"} {
		if got := string(tl.Install().Files[at]); got != "#!/bin/sh\necho run\n" {
			t.Errorf("Files[%s] = %q; want the tool's files/run.sh", at, got)
		}
	}
}

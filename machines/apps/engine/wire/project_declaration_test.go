package wire_test

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/core/confirm"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
	"github.com/jterrazz/studio-station/apps/engine/project"
	"github.com/jterrazz/studio-station/apps/engine/project/calls"
)

const secret = "OUTSIDE-SECRET-0xA2B"

// value: the machine's own disk, runner and clock, as the real adapters
// reach them.
var machine = local.New(nil, nil, nil, nil)

// value: rel of the project at root moved outside it, the secret added to
// each of its files, and a link to it left there instead.
func linkedAway(t *testing.T, root, rel string) {
	t.Helper()
	at := filepath.Join(root, filepath.FromSlash(rel))
	away := filepath.Join(t.TempDir(), "away")
	if err := machine.FS.Rename(at, away); err != nil {
		t.Fatal(err)
	}
	err := machine.FS.WalkDir(away, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(p, append(data, "\n"+secret+"\n"...), 0o600)
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := machine.FS.Symlink(away, at); err != nil {
		t.Fatal(err)
	}
}

func mkfifo(t *testing.T, at string) {
	t.Helper()
	if _, err := machine.Run.Output(t.Context(), local.Cmd{Name: "/usr/bin/mkfifo", Args: []string{at}}); err != nil {
		t.Fatal(err)
	}
}

// value: the error of read, failing the test when it waits past d. A read
// held on the FIFO at fifo holds the process-wide act lock of every
// project, so a writer opens the FIFO until the read lets go, and the tests
// after this one still run.
func within(t *testing.T, name, fifo string, d time.Duration, read func() error) error {
	t.Helper()
	ready, got := make(chan struct{}), make(chan error, 1)
	go func() { got <- read(); close(ready) }()
	_ = machine.Clock.Await(t.Context(), ready, d)
	select {
	case err := <-got:
		return err
	default:
	}
	go func() {
		for {
			select {
			case <-ready:
				return
			default:
				_ = machine.FS.Append(fifo, nil, 0o600)
			}
		}
	}()
	_ = machine.Clock.Await(t.Context(), ready, d)
	t.Fatalf("%s waited %s on the FIFO", name, d)
	return nil
}

// value: each read of the project at root, as the calls that read it.
func reads(t *testing.T, d calls.Deps, root string) map[string]func() error {
	return map[string]func() error{
		"show": func() error {
			_, err := calls.Show.Run(t.Context(), d, calls.ShowIn{Folder: root}, nil)
			return err
		},
		"plan": func() error {
			_, err := calls.Plan.Run(t.Context(), d, calls.PlanIn{Folder: root}, nil)
			return err
		},
		"drift": func() error {
			_, err := calls.Drift.Run(t.Context(), d, calls.DriftIn{Folder: root}, nil)
			return err
		},
		"apply": func() error {
			_, err := calls.Apply.Run(t.Context(), d, calls.ApplyIn{Folder: root, Confirm: confirm.Yes}, nil)
			return err
		},
		"render": func() error {
			_, err := d.Projects.Render(t.Context(), root, project.Starter)
			return err
		},
	}
}

// value: err refuses rel with code, as a call says it, or with why, as a
// method of the contract says it.
func refusedOrBare(err error, code fail.Code, why error, rel string) bool {
	var at *project.Unwritten
	return refusedAbout(err, code, rel) || errors.As(err, &at) && at.Path == rel && errors.Is(at.Err, why)
}

func TestALinkInTheDeclarationIsRefusedAndNothingItPointsToIsRead(t *testing.T) {
	for _, rel := range []string{
		"spwn.yaml", "spwn.lock", "spwn", "commands/refactor.md", "skills", "skills/focus.md",
		"agents/" + project.Starter, "agents/" + project.Starter + "/agent.yaml", "agents/" + project.Starter + "/AGENTS.md",
	} {
		t.Run(strings.ReplaceAll(rel, "/", "_"), func(t *testing.T) {
			d, _, root := applying(t)
			linkedAway(t, root, rel)
			for name, read := range reads(t, d, root) {
				if err := read(); !refusedOrBare(err, calls.LinkInTheWay, project.ErrLink, rel) {
					t.Errorf("%s = %v; want %s refused as a link in the way", name, err, rel)
				}
			}
			if _, err := machine.FS.Stat(filepath.Join(root, project.StateFolder)); !errors.Is(err, fs.ErrNotExist) {
				t.Errorf("%s is there (%v); nothing is written while the declaration is refused", project.StateFolder, err)
			}
		})
	}
}

func TestAFIFOInTheDeclarationIsRefusedAndNoReadWaitsOnIt(t *testing.T) {
	for _, rel := range []string{"commands/wait.md", "spwn.lock"} {
		t.Run(strings.ReplaceAll(rel, "/", "_"), func(t *testing.T) {
			d, _, root := applying(t)
			at := filepath.Join(root, filepath.FromSlash(rel))
			if err := machine.FS.RemoveAll(at); err != nil {
				t.Fatal(err)
			}
			mkfifo(t, at)
			for name, read := range reads(t, d, root) {
				if err := within(t, name, at, 10*time.Second, read); !refusedOrBare(err, calls.NotAFile, project.ErrNotAFile, rel) {
					t.Errorf("%s = %v; want %s refused as not a file", name, err, rel)
				}
			}
		})
	}
}

func TestAProjectWhoseRootIsALinkIsReadAndApplied(t *testing.T) {
	d, _, root := applying(t)
	via := filepath.Join(t.TempDir(), "via")
	if err := machine.FS.Symlink(root, via); err != nil {
		t.Fatal(err)
	}
	applied, err := calls.Apply.Run(t.Context(), d, calls.ApplyIn{Folder: via, Confirm: confirm.Yes}, nil)
	if err != nil || len(applied.Changes) == 0 {
		t.Fatalf("Apply through a linked root = %+v, %v; want the starter's files written", applied.Changes, err)
	}
	if _, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(applied.Changes[0].Path))); err != nil {
		t.Errorf("%s is not in the project: %v", applied.Changes[0].Path, err)
	}
}

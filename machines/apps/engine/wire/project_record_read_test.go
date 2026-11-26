package wire_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/core/confirm"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
	"github.com/jterrazz/studio-station/apps/engine/project"
	"github.com/jterrazz/studio-station/apps/engine/project/calls"
)

// value: the project at root applied once, then a file apply wrote
// changed, so the next apply has work; record is the record's path.
func appliedWithWork(t *testing.T) (d calls.Deps, root, record string) {
	t.Helper()
	d, _, root = applying(t)
	if _, err := calls.Apply.Run(t.Context(), d, calls.ApplyIn{Folder: root}, nil); err != nil {
		t.Fatal(err)
	}
	at := filepath.Join(root, "spwn", "agents", project.Starter, "AGENTS.md")
	data, err := os.ReadFile(at)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(at, append(data, "\nmore\n"...), 0o600); err != nil {
		t.Fatal(err)
	}
	return d, root, filepath.Join(root, project.StateFolder, project.RecordName)
}

func TestARecordThatDoesNotReadIsRefusedAndKeptAsItIs(t *testing.T) {
	for name, tc := range map[string]struct {
		data []byte
		says func(error) bool
	}{
		"truncated": {[]byte(`{".spwn/agents/neo/CLAUDE.md":`), func(err error) bool {
			return err != nil && strings.Contains(err.Error(), recordPath()+" does not read")
		}},
		"past-1-MiB": {append(append([]byte(`{"x":"`), bytes.Repeat([]byte("a"), local.MaxRead)...), `"}`...), func(err error) bool {
			return errors.Is(err, local.ErrTooLarge)
		}},
	} {
		t.Run(name, func(t *testing.T) {
			d, root, record := appliedWithWork(t)
			if err := os.WriteFile(record, tc.data, 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := calls.Plan.Run(t.Context(), d, calls.PlanIn{Folder: root}, nil); !tc.says(err) {
				t.Errorf("Plan = %v; want the record refused", err)
			}
			if _, err := calls.Apply.Run(t.Context(), d, calls.ApplyIn{Folder: root, Confirm: confirm.Yes}, nil); !tc.says(err) {
				t.Errorf("Apply = %v; want the record refused", err)
			}
			if data, err := os.ReadFile(record); err != nil || !bytes.Equal(data, tc.data) {
				t.Errorf("the record changed (%v); a record that does not read is kept as it is", err)
			}
		})
	}
}

func recordPath() string { return project.StateFolder + "/" + project.RecordName }

func TestAFIFOAtTheRecordIsRefusedAndNoReadWaitsOnIt(t *testing.T) {
	d, root, record := appliedWithWork(t)
	if err := machine.FS.Remove(record); err != nil {
		t.Fatal(err)
	}
	mkfifo(t, record)
	for name, read := range map[string]func() error{
		"plan": func() error {
			_, err := calls.Plan.Run(t.Context(), d, calls.PlanIn{Folder: root}, nil)
			return err
		},
		"apply": func() error {
			_, err := calls.Apply.Run(t.Context(), d, calls.ApplyIn{Folder: root, Confirm: confirm.Yes}, nil)
			return err
		},
	} {
		if err := within(t, name, record, 10*time.Second, read); err == nil {
			t.Errorf("%s read a FIFO as the record", name)
		}
	}
}

func TestAHardLinkAtARenderedFileIsRefusedAndWhatItNamesIsKept(t *testing.T) {
	d, folder, root := applying(t)
	if _, err := calls.Apply.Run(t.Context(), d, calls.ApplyIn{Folder: root}, nil); err != nil {
		t.Fatal(err)
	}
	rel := starterFile(t, d, root)
	at := filepath.Join(root, filepath.FromSlash(rel))
	victim := filepath.Join(folder, "victim.txt")
	if err := os.WriteFile(victim, []byte(secret+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := machine.FS.Remove(at); err != nil {
		t.Fatal(err)
	}
	if _, err := machine.Run.Output(t.Context(), local.Cmd{Name: "/bin/ln", Args: []string{victim, at}}); err != nil {
		t.Fatal(err)
	}
	if _, err := calls.Plan.Run(t.Context(), d, calls.PlanIn{Folder: root}, nil); !refusedAbout(err, calls.LinkInTheWay, rel) {
		t.Errorf("Plan = %v; want %s refused as a link in the way", err, rel)
	}
	if _, err := calls.Apply.Run(t.Context(), d, calls.ApplyIn{Folder: root, Confirm: confirm.Yes}, nil); !refusedAbout(err, calls.LinkInTheWay, rel) {
		t.Errorf("Apply = %v; want %s refused as a link in the way", err, rel)
	}
	data, err := os.ReadFile(victim)
	e, lerr := (local.Beneath{}).Lstat(folder, filepath.Base(victim))
	if err != nil || string(data) != secret+"\n" || lerr != nil || e.Nlink != 2 {
		t.Errorf("the file the hard link names changed: %q, %v, %v, %d names", data, err, lerr, e.Nlink)
	}
}

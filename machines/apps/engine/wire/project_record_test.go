package wire_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/core/confirm"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/project"
	"github.com/jterrazz/studio-station/apps/engine/project/calls"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/words"
)

// value: the project at root applied once, then its command and its skill
// changed, so the next apply overwrites what the first wrote; changed is
// the folder of the first file it overwrites.
func appliedThenChanged(t *testing.T) (d calls.Deps, root, changed string) {
	t.Helper()
	if machine.Env.Euid() == 0 {
		t.Skip("root writes in a folder without its write bit")
	}
	d, _, root = applying(t)
	if _, err := calls.Apply.Run(t.Context(), d, calls.ApplyIn{Folder: root}, nil); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{"spwn/commands/refactor.md", "spwn/skills/focus.md"} {
		at := filepath.Join(root, filepath.FromSlash(rel))
		data, err := os.ReadFile(at)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(at, append(data, "\nchanged\n"...), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	plan, err := calls.Plan.Run(t.Context(), d, calls.PlanIn{Folder: root}, nil)
	if err != nil || len(plan.Changes) == 0 {
		t.Fatalf("Plan = %+v, %v; want the changed files to overwrite", plan.Changes, err)
	}
	return d, root, filepath.Join(root, filepath.Dir(filepath.FromSlash(plan.Changes[0].Path)))
}

func readOnly(t *testing.T, dir string) {
	t.Helper()
	if err := machine.FS.Chmod(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = machine.FS.Chmod(dir, 0o755) })
}

// value: err refused as not written, the refusal said with words holding
// says.
func notWritten(t *testing.T, err error, says string) *fail.Error {
	t.Helper()
	var refused *fail.Error
	if !errors.As(err, &refused) || refused.Code != calls.NotWritten {
		t.Fatalf("Apply = %v; want it refused as not written", err)
	}
	if message, _ := words.Codes()[calls.NotWritten](refused); !strings.Contains(message, says) {
		t.Errorf("the refusal reads %q; want it to say %q", message, says)
	}
	return refused
}

func TestARecordThatCannotBeWrittenIsSaidAndTheOneBeforeStaysWhole(t *testing.T) {
	d, root, _ := appliedThenChanged(t)
	record := filepath.Join(root, project.StateFolder, project.RecordName)
	before, err := os.ReadFile(record)
	if err != nil {
		t.Fatal(err)
	}
	readOnly(t, filepath.Join(root, project.StateFolder))
	_, err = calls.Apply.Run(t.Context(), d, calls.ApplyIn{Folder: root, Confirm: confirm.Yes}, nil)
	refused := notWritten(t, err, "the record of what apply wrote, could not be written; every change is made")
	if refused.Subject != project.StateFolder+"/"+project.RecordName || refused.Detail["record"] != "not-written" {
		t.Errorf("refused = %q %v; want the record named, and said not written", refused.Subject, refused.Detail)
	}
	if after, err := os.ReadFile(record); err != nil || !bytes.Equal(after, before) {
		t.Errorf("the record holds %q, %v; want the one before whole: it is replaced at once or not at all", after, err)
	}
}

func TestAChangeAndTheRecordBothRefusedAreBothSaid(t *testing.T) {
	d, root, changed := appliedThenChanged(t)
	readOnly(t, filepath.Join(root, project.StateFolder))
	readOnly(t, changed)
	_, err := calls.Apply.Run(t.Context(), d, calls.ApplyIn{Folder: root, Confirm: confirm.Yes}, nil)
	refused := notWritten(t, err, "could not be written, nor .spwn/applied.json")
	if refused.Subject == project.StateFolder+"/"+project.RecordName || refused.Detail["record"] != "not-written" {
		t.Errorf("refused = %q %v; want the change named, and the record said not written too", refused.Subject, refused.Detail)
	}
}

func TestAChangeRefusedAloneSaysTheRecordKeptWhatWasWritten(t *testing.T) {
	d, root, changed := appliedThenChanged(t)
	readOnly(t, changed)
	_, err := calls.Apply.Run(t.Context(), d, calls.ApplyIn{Folder: root, Confirm: confirm.Yes}, nil)
	refused := notWritten(t, err, "could not be written; what apply wrote before it is recorded")
	if refused.Detail["record"] != "" {
		t.Errorf("refused = %q %v; want the record said written", refused.Subject, refused.Detail)
	}
}

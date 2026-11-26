package wire_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/core/confirm"
	"github.com/jterrazz/studio-station/apps/engine/project"
	"github.com/jterrazz/studio-station/apps/engine/project/calls"
)

// value: the starter's first rendered file, as a path from the root.
func starterFile(t *testing.T, d calls.Deps, root string) string {
	t.Helper()
	files, err := d.Projects.Render(t.Context(), root, project.Starter)
	if err != nil || len(files) == 0 {
		t.Fatalf("Render = %v, %v; want the starter's files", files, err)
	}
	return project.RunFolder(project.Starter) + "/" + files[0].Path
}

func TestAFolderOrAFIFOAtARenderedFileIsAPlanErrorAndApplyWritesNothing(t *testing.T) {
	for name, put := range map[string]func(t *testing.T, at string){
		"folder": func(t *testing.T, at string) {
			if err := machine.FS.MkdirAll(at, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(at, "mine.txt"), []byte("mine\n"), 0o600); err != nil {
				t.Fatal(err)
			}
		},
		"fifo": func(t *testing.T, at string) {
			if err := machine.FS.MkdirAll(filepath.Dir(at), 0o755); err != nil {
				t.Fatal(err)
			}
			mkfifo(t, at)
		},
	} {
		t.Run(name, func(t *testing.T) {
			d, _, root := applying(t)
			rel := starterFile(t, d, root)
			at := filepath.Join(root, filepath.FromSlash(rel))
			put(t, at)
			plan, err := calls.Plan.Run(t.Context(), d, calls.PlanIn{Folder: root}, nil)
			if err != nil || plan.Valid || !slices.ContainsFunc(plan.Issues, func(i project.Issue) bool { return i.Path == rel && i.Level == project.Error }) {
				t.Fatalf("Plan = %+v, %v; want %s an error of the plan", plan.Issues, err, rel)
			}
			if err := within(t, "apply", at, 10*time.Second, func() error {
				_, err := calls.Apply.Run(t.Context(), d, calls.ApplyIn{Folder: root, Confirm: confirm.Yes}, nil)
				return err
			}); !refusedWith(err, calls.Invalid) {
				t.Errorf("Apply = %v; want it refused as the plan's error", err)
			}
			if _, err := machine.FS.Stat(filepath.Join(root, project.StateFolder, project.RecordName)); err == nil {
				t.Errorf("the record is written; apply writes nothing while its plan has an error")
			}
		})
	}
}

func TestAFileWhereARunFolderGoesIsRefusedByPlanAndApply(t *testing.T) {
	d, _, root := applying(t)
	rel := project.RunFolder(project.Starter)
	at := filepath.Join(root, filepath.FromSlash(rel))
	if err := machine.FS.MkdirAll(filepath.Dir(at), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(at, []byte("mine\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := calls.Plan.Run(t.Context(), d, calls.PlanIn{Folder: root}, nil); !refusedAbout(err, calls.NotAFolder, rel) {
		t.Errorf("Plan = %v; want %s refused as a file where a folder goes", err, rel)
	}
	if _, err := calls.Apply.Run(t.Context(), d, calls.ApplyIn{Folder: root, Confirm: confirm.Yes}, nil); !refusedAbout(err, calls.NotAFolder, rel) {
		t.Errorf("Apply = %v; want %s refused as a file where a folder goes", err, rel)
	}
	if data, err := os.ReadFile(at); err != nil || string(data) != "mine\n" {
		t.Errorf("%s holds %q, %v; want the owner's file untouched", rel, data, err)
	}
}

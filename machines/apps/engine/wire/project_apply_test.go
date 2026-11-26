package wire_test

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/confirm"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
	"github.com/jterrazz/studio-station/apps/engine/project"
	"github.com/jterrazz/studio-station/apps/engine/project/adapter/manifest"
	"github.com/jterrazz/studio-station/apps/engine/project/calls"
)

func applying(t *testing.T) (d calls.Deps, folder, root string) {
	t.Helper()
	t.Setenv("SPWN_HOME", t.TempDir())
	d = calls.Deps{Projects: manifest.New(local.New(nil, nil, nil, nil).FS)}
	folder = t.TempDir()
	made, err := d.Projects.Create(t.Context(), folder, "site", catalog.Harnesses()[0])
	if err != nil {
		t.Fatal(err)
	}
	return d, folder, made.Root
}

func refusedWith(err error, code fail.Code) bool {
	var refused *fail.Error
	return errors.As(err, &refused) && refused.Code == code
}

func TestWhatAPlanListsIsWhatAnApplyWrites(t *testing.T) {
	d, folder, root := applying(t)
	plan, err := calls.Plan.Run(t.Context(), d, calls.PlanIn{Folder: root}, nil)
	if err != nil || len(plan.Changes) == 0 {
		t.Fatalf("Plan = %+v, %v; want the starter's files to add", plan.Changes, err)
	}
	applied, err := calls.Apply.Run(t.Context(), d, calls.ApplyIn{Folder: root}, nil)
	if err != nil || len(applied.Changes) != len(plan.Changes) {
		t.Fatalf("Apply = %+v, %v; want what the plan listed", applied.Changes, err)
	}
	for i, c := range plan.Changes {
		if c.Kind != project.Add || applied.Changes[i].Path != c.Path {
			t.Fatalf("change %d: plan %+v, apply %+v", i, c, applied.Changes[i])
		}
		on, read := os.ReadFile(filepath.Join(root, filepath.FromSlash(c.Path)))
		if read != nil || !bytes.Equal(on, c.File.Data) {
			t.Errorf("%s holds %q, %v; the plan said %q", c.Path, on, read, c.File.Data)
		}
	}
	entries, err := os.ReadDir(folder)
	if err != nil || len(entries) != 1 || entries[0].Name() != "site" {
		t.Errorf("the folder around the project holds %v, %v; want the project alone", entries, err)
	}
	again, err := calls.Apply.Run(t.Context(), d, calls.ApplyIn{Folder: root}, nil)
	if err != nil || len(again.Changes) != 0 {
		t.Errorf("a second Apply = %+v, %v; want nothing to apply", again.Changes, err)
	}
}

func TestAFileEditedByHandIsNeverOverwrittenUnasked(t *testing.T) {
	d, _, root := applying(t)
	applied, err := calls.Apply.Run(t.Context(), d, calls.ApplyIn{Folder: root}, nil)
	if err != nil || len(applied.Changes) == 0 {
		t.Fatalf("Apply = %+v, %v", applied, err)
	}
	edited := filepath.Join(root, filepath.FromSlash(applied.Changes[0].Path))
	if err = os.WriteFile(edited, []byte("mine\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	mine := filepath.Join(root, project.StateFolder, "agents", project.Starter, "notes.md")
	if err = os.WriteFile(mine, []byte("kept\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	refused, err := calls.Apply.Run(t.Context(), d, calls.ApplyIn{Folder: root}, nil)
	if !refusedWith(err, confirm.YesNeeded) || len(refused.Changes) != 1 || !refused.Changes[0].ByHand {
		t.Fatalf("Apply over a hand edit = %+v, %v; want it asked first", refused.Changes, err)
	}
	if on, _ := os.ReadFile(edited); string(on) != "mine\n" {
		t.Errorf("the refused apply wrote %q", on)
	}
	if _, err := calls.Apply.Run(t.Context(), d, calls.ApplyIn{Folder: root, Confirm: confirm.Yes}, nil); err != nil {
		t.Fatal(err)
	}
	if on, _ := os.ReadFile(edited); string(on) == "mine\n" {
		t.Error("an apply answered yes kept the hand edit")
	}
	if on, _ := os.ReadFile(mine); string(on) != "kept\n" {
		t.Errorf("a file apply never wrote reads %q; want it untouched", on)
	}
}

func TestALinkIsWrittenAsALinkAndReadAsOne(t *testing.T) {
	d, _, root := applying(t)
	link := project.File{Path: "SOUL.md", Data: []byte("../../../spwn/agents/neo/SOUL.md"), Link: true}
	exec := project.File{Path: "bin/run", Data: []byte("#!/bin/sh\n"), Mode: 0o755}
	rendered := map[string][]project.File{project.Starter: {link, exec}}
	changes, next, err := project.ChangesOf(rendered, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = d.Projects.Write(t.Context(), root, changes, next); err != nil {
		t.Fatal(err)
	}
	_, disk, err := d.Projects.Applied(t.Context(), root, project.Paths(rendered, nil))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range []project.File{link, exec} {
		p := project.RunFolder(project.Starter) + "/" + f.Path
		if disk[p] != project.Digest(f) {
			t.Errorf("%s reads %s; want %s", p, disk[p], project.Digest(f))
		}
	}
}

func TestAPlanWithAnErrorAppliesNothing(t *testing.T) {
	d, _, root := applying(t)
	if err := os.WriteFile(filepath.Join(root, "spwn", "agents", project.Starter, "agent.yaml"), []byte("name: [\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := calls.Apply.Run(t.Context(), d, calls.ApplyIn{Folder: root, Confirm: confirm.Yes}, nil); !refusedWith(err, calls.Invalid) {
		t.Fatalf("Apply = %v; want project.invalid", err)
	}
	if _, err := os.ReadDir(filepath.Join(root, project.StateFolder, "agents")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("an invalid plan wrote its run folders: %v", err)
	}
}

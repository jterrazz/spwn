package wire_test

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path"
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
	made, err := d.Projects.Create(t.Context(), folder, "site", catalog.Harnesses()[0], "")
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

func refusedAbout(err error, code fail.Code, subject string) bool {
	var refused *fail.Error
	return errors.As(err, &refused) && refused.Code == code && refused.Subject == subject
}

func TestALinkWhereApplyReachesIsRefusedAndNothingIsWrittenThroughIt(t *testing.T) {
	for _, c := range []struct {
		name, link string
		// value: whether the link names a folder; else a file, the record.
		folder, applied bool
	}{
		{"the run folder of an agent", ".spwn/agents/" + project.Starter, true, false},
		{"the state folder", project.StateFolder, true, false},
		{"the record", project.StateFolder + "/" + project.RecordName, false, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			d, _, root := applying(t)
			if c.applied {
				if _, err := calls.Apply.Run(t.Context(), d, calls.ApplyIn{Folder: root}, nil); err != nil {
					t.Fatal(err)
				}
			}
			away := t.TempDir()
			target := away
			if !c.folder {
				target = filepath.Join(away, "victim.json")
				if err := os.WriteFile(target, []byte("{}\n"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			at, disk := filepath.Join(root, filepath.FromSlash(c.link)), local.New(nil, nil, nil, nil).FS
			if err := os.MkdirAll(filepath.Dir(at), 0o750); err != nil {
				t.Fatal(err)
			}
			if err := disk.RemoveAll(at); err != nil {
				t.Fatal(err)
			}
			if err := disk.Symlink(target, at); err != nil {
				t.Fatal(err)
			}
			if _, err := calls.Plan.Run(t.Context(), d, calls.PlanIn{Folder: root}, nil); !refusedAbout(err, calls.LinkInTheWay, c.link) {
				t.Errorf("Plan = %v; want project.link-in-the-way about %s", err, c.link)
			}
			if _, err := calls.Apply.Run(t.Context(), d, calls.ApplyIn{Folder: root, Confirm: confirm.Yes}, nil); !refusedAbout(err, calls.LinkInTheWay, c.link) {
				t.Errorf("Apply --yes = %v; want project.link-in-the-way about %s", err, c.link)
			}
			entries, err := os.ReadDir(away)
			if c.folder && (err != nil || len(entries) != 0) {
				t.Errorf("the folder linked to holds %v, %v; want it empty", entries, err)
			}
			if data, _ := os.ReadFile(target); !c.folder && string(data) != "{}\n" {
				t.Errorf("the file linked to holds %q; want it untouched", data)
			}
		})
	}
}

func TestANameThatHoldsADanglingLinkExists(t *testing.T) {
	t.Setenv("SPWN_HOME", t.TempDir())
	d := calls.Deps{Projects: manifest.New(local.New(nil, nil, nil, nil).FS)}
	folder := t.TempDir()
	if err := local.New(nil, nil, nil, nil).FS.Symlink(filepath.Join(folder, "nowhere"), filepath.Join(folder, "site")); err != nil {
		t.Fatal(err)
	}
	for _, from := range []string{"", catalog.Examples()[0]} {
		if _, err := calls.Create.Run(t.Context(), d, calls.CreateIn{Folder: folder, Name: "site", From: from}, nil); !refusedWith(err, calls.Exists) {
			t.Errorf("Create (from %q) over a dangling link = %v; want project.exists", from, err)
		}
	}
	if _, err := (local.Beneath{}).Lstat(folder, "nowhere"); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("the link's target was made: %v", err)
	}
}

func holdsEveryFileOf(t *testing.T, files fs.FS, dir, root string) {
	t.Helper()
	want, err := fs.ReadDir(files, dir)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadDir(filepath.Join(root, filepath.FromSlash(dir)))
	if err != nil {
		t.Fatalf("the project lacks %s of the example: %v", dir, err)
	}
	held := map[string]bool{}
	for _, e := range got {
		held[e.Name()] = true
	}
	for _, e := range want {
		if !held[e.Name()] {
			t.Errorf("the project lacks %s/%s of the example", dir, e.Name())
		} else if e.IsDir() {
			holdsEveryFileOf(t, files, path.Join(dir, e.Name()), root)
		}
	}
}

func TestEachExampleOfTheCatalogPlansValidAndApplies(t *testing.T) {
	for _, example := range catalog.Examples() {
		t.Run(example, func(t *testing.T) {
			t.Setenv("SPWN_HOME", t.TempDir())
			d := calls.Deps{Projects: manifest.New(local.New(nil, nil, nil, nil).FS)}
			made, err := calls.Create.Run(t.Context(), d, calls.CreateIn{Folder: t.TempDir(), From: example}, nil)
			if err != nil || made.Name != example || len(made.Agents) == 0 {
				t.Fatalf("Create = %+v, %v; want the example's project named for it, with its agents", made, err)
			}
			files, _ := catalog.Example(example)
			holdsEveryFileOf(t, files, ".", made.Root)
			plan, err := calls.Plan.Run(t.Context(), d, calls.PlanIn{Folder: made.Root}, nil)
			if err != nil || !plan.Valid || len(plan.Changes) == 0 {
				t.Fatalf("Plan = %+v, %v; want valid, with files to apply", plan, err)
			}
			applied, err := calls.Apply.Run(t.Context(), d, calls.ApplyIn{Folder: made.Root}, nil)
			if err != nil || len(applied.Changes) != len(plan.Changes) {
				t.Fatalf("Apply = %+v, %v; want what the plan listed", applied.Changes, err)
			}
			for _, agent := range made.Agents {
				if entries, err := os.ReadDir(filepath.Join(made.Root, project.StateFolder, "agents", agent)); err != nil || len(entries) == 0 {
					t.Errorf("run folder of %s holds %v, %v; want its rendered files", agent, entries, err)
				}
			}
			if again, err := calls.Apply.Run(t.Context(), d, calls.ApplyIn{Folder: made.Root}, nil); err != nil || len(again.Changes) != 0 {
				t.Errorf("a second Apply = %+v, %v; want nothing to apply", again.Changes, err)
			}
		})
	}
}

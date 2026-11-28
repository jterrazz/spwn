package contracttest

import (
	"context"
	"errors"
	"io/fs"
	"maps"
	"path/filepath"
	"slices"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/project"
	"github.com/jterrazz/studio-station/apps/engine/project/contract"
)

func Projects(t *testing.T, open func(t *testing.T) (contract.Projects, string)) {
	// invariant: open gives Projects over a folder of its own, no project in
	// it nor above it, and that folder.
	ctx := context.Background()
	harnesses := catalog.Harnesses()
	first, last := harnesses[0], harnesses[len(harnesses)-1]

	t.Run("a folder in no project finds none", func(t *testing.T) {
		p, folder := open(t)
		if root, err := p.Find(ctx, folder); !errors.Is(err, project.ErrNotFound) {
			t.Fatalf("Find = %q, %v; want ErrNotFound", root, err)
		}
	})

	t.Run("a project made is a new folder with its starter agent, found from its root", func(t *testing.T) {
		p, folder := open(t)
		made, err := p.Create(ctx, folder, "site", last, "")
		want := project.Project{Name: "site", Root: filepath.Join(folder, "site"), Manifest: filepath.Join(folder, "site", project.ManifestName), Agents: []string{project.Starter}}
		if err != nil || made.Name != want.Name || made.Root != want.Root || made.Manifest != want.Manifest || !slices.Equal(made.Agents, want.Agents) {
			t.Fatalf("Create = %+v, %v; want %+v", made, err, want)
		}
		root, err := p.Find(ctx, made.Root)
		if err != nil || root != made.Root {
			t.Fatalf("Find = %q, %v; want %q", root, err, made.Root)
		}
		shown, err := p.Show(ctx, root)
		if err != nil || shown.Name != want.Name || shown.Root != want.Root || !slices.Equal(shown.Agents, want.Agents) {
			t.Fatalf("Show = %+v, %v; want %+v", shown, err, want)
		}
		if _, err := p.Find(ctx, folder); !errors.Is(err, project.ErrNotFound) {
			t.Errorf("Find of the folder above = %v; want ErrNotFound", err)
		}
	})

	t.Run("a folder that exists is refused", func(t *testing.T) {
		p, folder := open(t)
		if _, err := p.Create(ctx, folder, "site", first, ""); err != nil {
			t.Fatal(err)
		}
		if _, err := p.Create(ctx, folder, "site", first, ""); !errors.Is(err, project.ErrExists) {
			t.Errorf("Create twice = %v; want ErrExists", err)
		}
	})

	t.Run("a new project plans valid and has not drifted", func(t *testing.T) {
		p, folder := open(t)
		made, err := p.Create(ctx, folder, "site", first, "")
		if err != nil {
			t.Fatal(err)
		}
		plan, err := p.Plan(ctx, made.Root)
		if err != nil || !plan.Valid || plan.Summary.Errors != 0 || plan.Root != made.Root || plan.Manifest != made.Manifest {
			t.Fatalf("Plan = %+v, %v; want valid", plan, err)
		}
		drift, err := p.Drift(ctx, made.Root)
		if err != nil || len(drift.Rows) != 0 || drift.Root != made.Root {
			t.Fatalf("Drift = %+v, %v; want no row", drift, err)
		}
	})

	t.Run("a new project's starter renders in memory, and nothing is on disk until written", func(t *testing.T) {
		p, folder := open(t)
		made, err := p.Create(ctx, folder, "site", first, "")
		if err != nil {
			t.Fatal(err)
		}
		files, err := p.Render(ctx, made.Root, project.Starter)
		if err != nil || len(files) == 0 {
			t.Fatalf("Render = %d files, %v; want the starter's", len(files), err)
		}
		rendered := map[string][]project.File{project.Starter: files}
		record, disk, err := p.Applied(ctx, made.Root, project.Paths(rendered, nil))
		if err != nil || len(record) != 0 || len(disk) != 0 {
			t.Fatalf("Applied = %v, %v, %v; want nothing applied", record, disk, err)
		}
		changes, next, err := project.ChangesOf(rendered, record, disk)
		if err != nil || len(changes) != len(files) || len(project.Asked(changes)) != 0 {
			t.Fatalf("ChangesOf = %+v, %v; want one add per file", changes, err)
		}
		if err = p.Write(ctx, made.Root, changes, next); err != nil {
			t.Fatal(err)
		}
		record, disk, err = p.Applied(ctx, made.Root, project.Paths(rendered, nil))
		if err != nil || !maps.Equal(record, next) || !maps.Equal(disk, next) {
			t.Fatalf("Applied after Write = %v, %v, %v; want the record and the disk as rendered", record, disk, err)
		}
		again, _, err := project.ChangesOf(rendered, record, disk)
		if err != nil || len(again) != 0 {
			t.Fatalf("ChangesOf after Write = %+v, %v; want none", again, err)
		}
	})

	t.Run("a project made from each example holds its agents, plans valid and renders each", func(t *testing.T) {
		for _, example := range catalog.Examples() {
			p, folder := open(t)
			files, _ := catalog.Example(example)
			entries, err := fs.ReadDir(files, "agents")
			if err != nil {
				t.Fatal(err)
			}
			agents := make([]string, 0, len(entries))
			for _, e := range entries {
				agents = append(agents, e.Name())
			}
			made, err := p.Create(ctx, folder, "mine", "", example)
			if err != nil || made.Name != "mine" || made.Root != filepath.Join(folder, "mine") || !slices.Equal(made.Agents, agents) {
				t.Fatalf("%s: Create = %+v, %v; want the project mine with agents %v", example, made, err, agents)
			}
			plan, err := p.Plan(ctx, made.Root)
			if err != nil || !plan.Valid {
				t.Fatalf("%s: Plan = %+v, %v; want valid", example, plan, err)
			}
			for _, agent := range agents {
				if rendered, err := p.Render(ctx, made.Root, agent); err != nil || len(rendered) == 0 {
					t.Errorf("%s: Render(%s) = %v, %v; want files", example, agent, rendered, err)
				}
			}
			if _, err := p.Create(ctx, folder, "mine", "", example); !errors.Is(err, project.ErrExists) {
				t.Errorf("%s: Create over its own folder = %v; want ErrExists", example, err)
			}
		}
	})

	t.Run("a file written and then no longer rendered is removed, and the record forgets it", func(t *testing.T) {
		p, folder := open(t)
		made, err := p.Create(ctx, folder, "site", first, "")
		if err != nil {
			t.Fatal(err)
		}
		files, err := p.Render(ctx, made.Root, project.Starter)
		if err != nil {
			t.Fatal(err)
		}
		rendered := map[string][]project.File{project.Starter: files}
		changes, next, _ := project.ChangesOf(rendered, nil, nil)
		if err = p.Write(ctx, made.Root, changes, next); err != nil {
			t.Fatal(err)
		}
		gone, kept, err := project.ChangesOf(map[string][]project.File{}, next, next)
		if err != nil || len(gone) != len(files) || gone[0].Kind != project.Remove || gone[0].ByHand {
			t.Fatalf("ChangesOf = %+v, %v; want each file removed", gone, err)
		}
		if err = p.Write(ctx, made.Root, gone, kept); err != nil {
			t.Fatal(err)
		}
		record, disk, err := p.Applied(ctx, made.Root, project.Paths(rendered, next))
		if err != nil || len(record) != 0 || len(disk) != 0 {
			t.Fatalf("Applied after the removal = %v, %v, %v; want nothing", record, disk, err)
		}
	})
}

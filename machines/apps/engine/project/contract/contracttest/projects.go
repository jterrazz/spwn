package contracttest

import (
	"context"
	"errors"
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
		made, err := p.Create(ctx, folder, "site", last)
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
		if _, err := p.Create(ctx, folder, "site", first); err != nil {
			t.Fatal(err)
		}
		if _, err := p.Create(ctx, folder, "site", first); !errors.Is(err, project.ErrExists) {
			t.Errorf("Create twice = %v; want ErrExists", err)
		}
	})

	t.Run("a new project plans valid and has not drifted", func(t *testing.T) {
		p, folder := open(t)
		made, err := p.Create(ctx, folder, "site", first)
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
}

package contracttest

import (
	"context"
	"errors"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/skills"
	"github.com/jterrazz/studio-station/apps/engine/skills/contract"
)

func Files(t *testing.T, open func(t *testing.T) (contract.Files, string)) {
	// invariant: open gives Files over a project of its own, none of its
	// skills made yet, and that project's root.
	ctx := context.Background()

	t.Run("a project no skill was made in lists none", func(t *testing.T) {
		f, project := open(t)
		got, err := f.List(ctx, project)
		if err != nil || len(got) != 0 {
			t.Fatalf("List = %+v, %v; want none", got, err)
		}
	})

	t.Run("a skill made is its template, listed without its body and read back whole", func(t *testing.T) {
		f, project := open(t)
		made, err := f.Create(ctx, project, "focus")
		if err != nil || made.Name != "focus" || made.Body != skills.Template("focus") || made.Description != skills.Describe(made.Body) || made.Path == "" {
			t.Fatalf("Create = %+v, %v; want focus with its template", made, err)
		}
		list, err := f.List(ctx, project)
		want := made
		want.Body = ""
		if err != nil || len(list) != 1 || list[0] != want {
			t.Fatalf("List = %+v, %v; want %+v alone", list, err, want)
		}
		shown, err := f.Show(ctx, project, "focus")
		if err != nil || shown != made {
			t.Fatalf("Show = %+v, %v; want %+v", shown, err, made)
		}
	})

	t.Run("skills are listed by name", func(t *testing.T) {
		f, project := open(t)
		for _, name := range []string{"refactoring", "focus", "paper-reading"} {
			if _, err := f.Create(ctx, project, name); err != nil {
				t.Fatal(err)
			}
		}
		list, err := f.List(ctx, project)
		if err != nil || len(list) != 3 || list[0].Name != "focus" || list[1].Name != "paper-reading" || list[2].Name != "refactoring" {
			t.Fatalf("List = %+v, %v; want focus, paper-reading, refactoring", list, err)
		}
	})

	t.Run("a name taken is refused, and a name no skill has is not found", func(t *testing.T) {
		f, project := open(t)
		if _, err := f.Create(ctx, project, "focus"); err != nil {
			t.Fatal(err)
		}
		if _, err := f.Create(ctx, project, "focus"); !errors.Is(err, skills.ErrExists) {
			t.Errorf("Create twice = %v; want ErrExists", err)
		}
		for name, act := range map[string]func() error{
			"Show":             func() error { _, err := f.Show(ctx, project, "ghost"); return err },
			"Delete":           func() error { return f.Delete(ctx, project, "ghost") },
			"Show of a path":   func() error { _, err := f.Show(ctx, project, "../focus"); return err },
			"Delete of a path": func() error { return f.Delete(ctx, project, "../focus") },
			"Show of a folder": func() error { _, err := f.Show(ctx, project, "."); return err },
		} {
			if err := act(); !errors.Is(err, skills.ErrNotFound) {
				t.Errorf("%s = %v; want ErrNotFound", name, err)
			}
		}
	})

	t.Run("a deleted skill is gone, the others kept", func(t *testing.T) {
		f, project := open(t)
		for _, name := range []string{"focus", "refactoring"} {
			if _, err := f.Create(ctx, project, name); err != nil {
				t.Fatal(err)
			}
		}
		if err := f.Delete(ctx, project, "focus"); err != nil {
			t.Fatalf("Delete = %v", err)
		}
		if _, err := f.Show(ctx, project, "focus"); !errors.Is(err, skills.ErrNotFound) {
			t.Errorf("Show after the delete = %v; want ErrNotFound", err)
		}
		if list, _ := f.List(ctx, project); len(list) != 1 || list[0].Name != "refactoring" {
			t.Errorf("List after the delete = %+v; want refactoring alone", list)
		}
	})

	t.Run("two projects keep their own skills", func(t *testing.T) {
		f, project := open(t)
		if _, err := f.Create(ctx, project, "focus"); err != nil {
			t.Fatal(err)
		}
		if list, err := f.List(ctx, project+"-other"); err != nil || len(list) != 0 {
			t.Errorf("List of another project = %+v, %v; want none", list, err)
		}
	})
}

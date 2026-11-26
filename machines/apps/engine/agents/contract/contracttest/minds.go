package contracttest

import (
	"context"
	"errors"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/agents"
	"github.com/jterrazz/studio-station/apps/engine/agents/contract"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

func Minds(t *testing.T, open func(t *testing.T) (contract.Minds, string)) {
	// invariant: open gives Minds over a project of its own, none of its
	// agents made yet, and that project's root.
	ctx := context.Background()
	team, harness := "web", catalog.Harnesses()[1]
	asked := agents.Edit{Team: &team, Harness: &harness}
	neo := asked.Apply(agents.Blank("neo"))

	t.Run("a project no agent was made in lists none", func(t *testing.T) {
		m, project := open(t)
		got, err := m.List(ctx, project)
		if err != nil || len(got) != 0 {
			t.Fatalf("List = %+v, %v; want none", got, err)
		}
	})

	t.Run("an agent made is listed and read back, its layers empty", func(t *testing.T) {
		m, project := open(t)
		made, err := m.Create(ctx, project, "neo", asked)
		if err != nil || made.Agent != neo {
			t.Fatalf("Create = %+v, %v; want neo", made, err)
		}
		list, err := m.List(ctx, project)
		if err != nil || len(list) != 1 || list[0] != neo {
			t.Fatalf("List = %+v, %v; want neo alone", list, err)
		}
		got, err := m.Mind(ctx, project, "neo")
		if err != nil || got.Agent != neo || layerNames(got) == nil || !slices.Equal(layerNames(got), []string{"journal", "playbooks"}) {
			t.Fatalf("Mind = %+v, %v; want neo with its two layers", got, err)
		}
		if len(got.Journal) != 0 {
			t.Errorf("Journal = %+v; want none", got.Journal)
		}
	})

	t.Run("a name taken is refused, and a name no agent has is not found", func(t *testing.T) {
		m, project := open(t)
		if _, err := m.Create(ctx, project, "neo", asked); err != nil {
			t.Fatal(err)
		}
		if _, err := m.Create(ctx, project, "neo", agents.Edit{}); !errors.Is(err, agents.ErrExists) {
			t.Errorf("Create twice = %v; want ErrExists", err)
		}
		for name, act := range map[string]func() error{
			"Mind":   func() error { _, err := m.Mind(ctx, project, "trinity"); return err },
			"Edit":   func() error { _, err := m.Edit(ctx, project, "trinity", agents.Edit{}); return err },
			"Delete": func() error { return m.Delete(ctx, project, "trinity") },
			"Fork":   func() error { _, err := m.Fork(ctx, project, "trinity", "smith"); return err },
			"Export": func() error { _, err := m.Export(ctx, project, "trinity", t.TempDir()); return err },
			"Dream":  func() error { _, err := m.Dream(ctx, project, "trinity"); return err },
			"Sleep":  func() error { _, err := m.Sleep(ctx, project, "trinity"); return err },
		} {
			if err := act(); !errors.Is(err, agents.ErrNotFound) {
				t.Errorf("%s of an agent never made = %v; want ErrNotFound", name, err)
			}
		}
	})

	t.Run("an edit changes the fields it names, and the change is kept", func(t *testing.T) {
		m, project := open(t)
		if _, err := m.Create(ctx, project, "neo", asked); err != nil {
			t.Fatal(err)
		}
		team, none := "core", ""
		got, err := m.Edit(ctx, project, "neo", agents.Edit{Team: &team, Harness: &none})
		want := agents.Agent{Name: "neo", Description: neo.Description, Team: "core"}
		if err != nil || got != want {
			t.Fatalf("Edit = %+v, %v; want %+v", got, err, want)
		}
		if read, _ := m.Mind(ctx, project, "neo"); read.Agent != want {
			t.Errorf("Mind after the edit = %+v; want %+v", read.Agent, want)
		}
	})

	t.Run("a deleted agent is gone", func(t *testing.T) {
		m, project := open(t)
		if _, err := m.Create(ctx, project, "neo", asked); err != nil {
			t.Fatal(err)
		}
		if err := m.Delete(ctx, project, "neo"); err != nil {
			t.Fatalf("Delete = %v", err)
		}
		if _, err := m.Mind(ctx, project, "neo"); !errors.Is(err, agents.ErrNotFound) {
			t.Errorf("Mind after Delete = %v; want ErrNotFound", err)
		}
	})

	t.Run("a fork is a new agent with every layer, onto a name not taken", func(t *testing.T) {
		m, project := open(t)
		if _, err := m.Create(ctx, project, "neo", asked); err != nil {
			t.Fatal(err)
		}
		got, err := m.Fork(ctx, project, "neo", "smith")
		if err != nil || got.From != "neo" || got.To != "smith" || !slices.Equal(got.Layers, []string{"playbooks", "journal"}) {
			t.Fatalf("Fork = %+v, %v; want neo to smith, both layers", got, err)
		}
		if _, err := m.Mind(ctx, project, "smith"); err != nil {
			t.Errorf("Mind of the fork = %v", err)
		}
		if _, err := m.Fork(ctx, project, "neo", "smith"); !errors.Is(err, agents.ErrExists) {
			t.Errorf("Fork onto a name taken = %v; want ErrExists", err)
		}
	})

	t.Run("an export imports back as another agent", func(t *testing.T) {
		m, project := open(t)
		if _, err := m.Create(ctx, project, "neo", asked); err != nil {
			t.Fatal(err)
		}
		folder := t.TempDir()
		out, err := m.Export(ctx, project, "neo", folder)
		if err != nil || out.Agent != "neo" || out.Archive != filepath.Join(folder, "neo"+agents.ArchiveSuffix) {
			t.Fatalf("Export = %+v, %v; want neo's archive in the folder", out, err)
		}
		if _, again := m.Import(ctx, project, out.Archive, ""); !errors.Is(again, agents.ErrExists) {
			t.Errorf("Import under the name taken = %v; want ErrExists", again)
		}
		got, err := m.Import(ctx, project, out.Archive, "smith")
		if err != nil || got.Name != "smith" || got.Team != "web" || !slices.Equal(layerNames(got), []string{"journal", "playbooks"}) {
			t.Fatalf("Import = %+v, %v; want smith with neo's Mind", got, err)
		}
	})

	t.Run("an archive that does not open leaves no agent", func(t *testing.T) {
		m, project := open(t)
		bad := filepath.Join(t.TempDir(), "trinity"+agents.ArchiveSuffix)
		if _, err := m.Import(ctx, project, bad, ""); !errors.Is(err, agents.ErrBadArchive) {
			t.Errorf("Import = %v; want ErrBadArchive", err)
		}
		if list, _ := m.List(ctx, project); len(list) != 0 {
			t.Errorf("List = %+v; want none", list)
		}
	})

	t.Run("a Mind with no journal dreams nothing, and sleeps", func(t *testing.T) {
		m, project := open(t)
		if _, err := m.Create(ctx, project, "neo", asked); err != nil {
			t.Fatal(err)
		}
		d, err := m.Dream(ctx, project, "neo")
		if err != nil || d != (agents.Dreamed{Agent: "neo", Skipped: "no journal entries"}) {
			t.Errorf("Dream = %+v, %v; want skipped, no journal entries", d, err)
		}
		s, err := m.Sleep(ctx, project, "neo")
		if err != nil || s != (agents.Slept{Agent: "neo"}) {
			t.Errorf("Sleep = %+v, %v; want nothing archived nor pruned", s, err)
		}
		got, _ := m.Mind(ctx, project, "neo")
		if j := got.Layers[0]; j.Name != "journal" || len(j.Files) != 1 || !strings.HasPrefix(j.Files[0], "sleep-") {
			t.Errorf("journal after Sleep = %+v; want its sleep log alone", j)
		}
	})
}

func layerNames(m agents.Mind) []string {
	out := make([]string, 0, len(m.Layers))
	for _, l := range m.Layers {
		out = append(out, l.Name)
	}
	return out
}

package calls

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/disk"
	"github.com/jterrazz/studio-station/apps/engine/disk/adapter/fake"
	"github.com/jterrazz/studio-station/apps/engine/setup"
)

// value: a machine whose files read as files, as the real disk's do: a
// path the fake holds with nothing under it is not a folder to list.
type withFiles struct{ onMachine }

func (m withFiles) Names(dir string) ([]string, error) {
	if names := m.Children(dir); len(names) > 0 {
		return names, nil
	}
	return nil, errors.New(dir + " is not a folder")
}

type relocations struct{ apps []string }

func (r *relocations) Relocate(_ context.Context, app string) error {
	r.apps = append(r.apps, app)
	return nil
}

var keeper = catalog.App{
	Name:     "keeper",
	Installs: []catalog.LivesOn{{Kinds: []catalog.MachineKind{catalog.Mac}, Install: catalog.CaskInstall{Token: "keeper"}}},
	Disk: catalog.Disk{
		Paths:     []catalog.Stored{{Path: "{cache}/keeper", Level: catalog.LevelCache}},
		Relocate:  []catalog.Relocation{{File: catalog.FileBunfig, Key: "install.cache.dir", Value: "{cache}/keeper"}},
		Misplaced: []catalog.Misplaced{{Path: "~/.keeper", Move: []catalog.Move{{From: "~/.keeper/*", To: "{cache}/keeper"}}, Prune: []string{"~/.keeper"}}},
	},
}

var builder = catalog.App{
	Name:     "builder",
	Installs: []catalog.LivesOn{{Kinds: []catalog.MachineKind{catalog.Mac}, Install: catalog.CaskInstall{Token: "builder"}}},
	Disk: catalog.Disk{
		Paths:     []catalog.Stored{{Path: "{data}/builder/bin/*", Level: catalog.LevelData}},
		Misplaced: []catalog.Misplaced{{Path: "~/builder", Move: []catalog.Move{{From: "~/builder/bin/*", To: "{data}/builder/bin"}}, Prune: []string{"~/builder"}}},
	},
}

func movingOn(files map[string]int64, app catalog.App) (*Surveyor, *fake.Usage, *relocations) {
	usage := fake.NewUsage(files, 1000, 500)
	r := &relocations{}
	return NewSurveyor(withFiles{onMachine{Usage: usage}}, []catalog.App{app}, SurveyorOptions{
		Disk: withSystem{usage}, ThisMachine: catalog.Mac, Relocator: r,
		Expand: disk.Expander{Home: "/h", Roots: setup.Roots{Cache: "/h/.cache", Data: "/h/.local/share"}},
	}), usage, r
}

func TestAMovedFolderTakesItsAppsRelocationSoItIsNotWrittenAgain(t *testing.T) {
	s, _, r := movingOn(map[string]int64{"/h/.keeper/_cacache/x": 30}, keeper)

	if _, err := s.Move(context.Background(), "~/.keeper"); err != nil {
		t.Fatal(err)
	}

	if !slices.Equal(r.apps, []string{"keeper"}) {
		t.Errorf("the move relocated %q; want keeper's settings applied, so it looks at ~/.cache/keeper after it", r.apps)
	}
}

func TestAMoveOntoACacheThatHoldsTheSameEntryKeepsTheCachesOwn(t *testing.T) {
	s, usage, _ := movingOn(map[string]int64{
		"/h/.keeper/_logs/new.log":     8,
		"/h/.keeper/_tmp/a/package":    5,
		"/h/.cache/keeper/_logs/a.log": 8,
	}, keeper)

	got, err := s.Move(context.Background(), "~/.keeper")

	if err != nil || got.Detail != "moved to ~/.cache/keeper" {
		t.Fatalf("move %+v %v; want ~/.keeper moved, its duplicate _logs dropped", got, err)
	}
	if usage.Has("/h/.keeper") || !usage.Has("/h/.cache/keeper/_tmp/a/package") || !usage.Has("/h/.cache/keeper/_logs/a.log") || usage.Has("/h/.cache/keeper/_logs/new.log") {
		t.Errorf("after the move the disk holds %q; want _tmp moved, the cache's _logs kept and ~/.keeper gone", usage.Glob("/h/*/*/*"))
	}
}

func TestAMoveOntoDataThatHoldsTheSameEntryIsRefusedAndMovesNothing(t *testing.T) {
	s, usage, _ := movingOn(map[string]int64{
		"/h/builder/bin/gopls":            8,
		"/h/builder/bin/dlv":              5,
		"/h/.local/share/builder/bin/dlv": 9,
	}, builder)

	_, err := s.Move(context.Background(), "~/builder")

	var held *disk.DestinationHeldError
	if !errors.As(err, &held) || held.Path != "~/.local/share/builder/bin/dlv" || held.Folder != "~/builder" {
		t.Fatalf("move %v; want it refused on ~/.local/share/builder/bin/dlv, naming ~/builder", err)
	}
	if !usage.Has("/h/builder/bin/gopls") || !usage.Has("/h/builder/bin/dlv") {
		t.Errorf("a refused move moved %q; want nothing moved", usage.Glob("/h/builder/bin/*"))
	}
}

type linkedHome struct {
	onMachine
	links  map[string]string
	pruned []string
}

func (m *linkedHome) Link(p string) (string, bool) {
	to, ok := m.links[p]
	return to, ok
}

func (m *linkedHome) Prune(dir string) { m.pruned = append(m.pruned, dir) }

type movedFiles struct {
	*fake.Usage
	moved []string
}

func (f *movedFiles) Move(ctx context.Context, from, to string) error {
	f.moved = append(f.moved, from)
	return f.Usage.Move(ctx, from, to)
}

func TestAMisplacedFolderThatIsALinkIsRefusedAndNothingMovesOrIsPruned(t *testing.T) {
	usage := fake.NewUsage(map[string]int64{"/h/go/pkg/mod/x/f": 2}, 1<<40, 1<<39)
	files := &movedFiles{Usage: usage}
	machine := &linkedHome{onMachine: onMachine{Usage: usage}, links: map[string]string{"/h/go": "/h/elsewhere"}}
	apps := []catalog.App{{
		Name:     "go",
		Installs: []catalog.LivesOn{{Kinds: []catalog.MachineKind{catalog.Mac}, Install: catalog.CaskInstall{Token: "go"}}},
		Disk: catalog.Disk{Misplaced: []catalog.Misplaced{{
			Path:  "~/go",
			Move:  []catalog.Move{{From: "~/go/pkg/mod/*", To: "~/.cache/go/mod"}},
			Prune: []string{"~/go/pkg", "~/go"},
		}}},
	}}
	s := NewSurveyor(machine, apps, SurveyorOptions{Disk: files, ThisMachine: catalog.Mac, Expand: disk.Expander{Home: "/h"}})

	_, err := s.Move(context.Background(), "~/go")

	var linked *disk.LinkedError
	if !errors.As(err, &linked) || linked.Path != "~/go" || linked.Target != "~/elsewhere" {
		t.Fatalf("Move of a linked folder returned %v, want a LinkedError for ~/go to ~/elsewhere", err)
	}
	if len(files.moved) != 0 || len(machine.pruned) != 0 {
		t.Errorf("moved %v and pruned %v, want neither", files.moved, machine.pruned)
	}
	if got := s.misplacedOnes(s.Entries()[0]); len(got) != 0 {
		t.Errorf("the listing shows %+v for a linked folder, want none", got)
	}
}

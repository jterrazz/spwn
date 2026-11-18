package wire

import (
	"context"
	"maps"
	"slices"
	"strconv"
	"testing"
	"testing/fstest"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/call/lanes"
	"github.com/jterrazz/studio-station/apps/engine/disk"
	diskcalls "github.com/jterrazz/studio-station/apps/engine/disk/calls"
	"github.com/jterrazz/studio-station/apps/engine/page"
	pagesdisk "github.com/jterrazz/studio-station/apps/engine/pages/disk"
	"github.com/jterrazz/studio-station/apps/engine/platform/local/localtest"
)

type staggerClock struct{}

func (staggerClock) Now() time.Time { return time.Unix(0, 0) }

func (staggerClock) Sleep(ctx context.Context, d time.Duration) error {
	waited, cancel := context.WithTimeout(ctx, d)
	defer cancel()
	<-waited.Done()
	return ctx.Err()
}

// value: a disk that answers its roots one at a time, 10 ms apart.
type staggered struct{ localtest.Disk }

func (f staggered) Sizes(ctx context.Context, roots []string, each func(int, int64)) {
	for i, root := range roots {
		_ = staggerClock{}.Sleep(ctx, 10*time.Millisecond)
		f.FS.Sizes(ctx, []string{root}, func(_ int, b int64) { each(i, b) })
	}
}

// value: an inventory that names its apps one at a time, the odd ones not
// installed, so some rows (and the sections they alone fill) leave at the end.
type staggeredInventory struct{ names []string }

func (v staggeredInventory) Installed(ctx context.Context, each func(map[string]bool)) {
	got := map[string]bool{}
	for i, n := range v.names {
		_ = staggerClock{}.Sleep(ctx, 10*time.Millisecond)
		got[n] = i%2 == 0
		each(maps.Clone(got))
	}
}

func shape(p page.Page) (ids []string, rows map[string]int) {
	rows = map[string]int{}
	for _, s := range p.Sections {
		ids = append(ids, s.ID)
		rows[s.ID] = len(s.Items)
	}
	return ids, rows
}

func TestNoSectionFlickers(t *testing.T) {
	fs := staggered{Disk: localtest.Disk{FS: localtest.NewFS()}}
	categories := []catalog.Category{catalog.CategoryPackageManagers, catalog.CategoryRuntimes, catalog.CategoryAIApps, catalog.CategoryAgentSkills}
	apps := make([]catalog.App, 0, 40)
	names := make([]string, 0, 40)
	for i := range 40 {
		name := "t" + strconv.Itoa(i)
		fs.Files["h/"+name+"/cache/f"] = &fstest.MapFile{Data: []byte("abc")}
		apps = append(apps, catalog.App{Name: name, Category: categories[i%len(categories)], Installs: []catalog.LivesOn{{Kinds: []catalog.MachineKind{catalog.Mac}, Install: catalog.ShellInstall{Line: name}}},
			Disk: catalog.Disk{Paths: []catalog.Stored{{Path: "~/" + name + "/*", Level: catalog.LevelCache}}}})
		names = append(names, name)
	}
	pool := lanes.NewPool(context.Background(), staggerClock{}, 4, 1)
	defer pool.Close()
	system := localtest.Mac(localtest.NewRunner(t), fs.FS, localtest.Env{}, localtest.NewNet(), &localtest.Clock{})
	system.FS, system.Clock = fs, staggerClock{}
	s := diskcalls.NewSurveyor(diskMachine{system}, apps, diskcalls.SurveyorOptions{Disk: fs, Pool: pool, ThisMachine: catalog.Mac, Expand: disk.Expander{Home: "/h"}, Budget: 5 * time.Second, Inventory: staggeredInventory{names: names}})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var pages []page.Page
	for r := range s.Watch(ctx) {
		pages = append(pages, pagesdisk.Read(disk.Reading{Rows: disk.Classify(r)}, page.Context{Kind: "system"}, false))
	}
	if len(pages) < 5 {
		t.Fatalf("the watch sent %d values, want one per staggered answer", len(pages))
	}
	for i := 1; i < len(pages)-1; i++ {
		before, beforeRows := shape(pages[i-1])
		after, afterRows := shape(pages[i])
		for _, id := range before {
			if !slices.Contains(after, id) {
				t.Fatalf("value %d: section %s left before the final value", i, id)
			}
			if afterRows[id] < beforeRows[id] {
				t.Fatalf("value %d: section %s shrank from %d to %d rows before the final value", i, id, beforeRows[id], afterRows[id])
			}
		}
	}
}

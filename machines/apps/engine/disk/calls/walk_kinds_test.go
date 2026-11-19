package calls

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/disk"
	"github.com/jterrazz/studio-station/apps/engine/disk/adapter/fake"
)

// value: a disk whose held roots answer only once the test releases them.
type heldRoots struct {
	*fake.Usage
	held    []string
	release chan struct{}
}

func (f heldRoots) Sizes(ctx context.Context, roots []string, each func(int, int64)) {
	var later []int
	for i, p := range roots {
		if slices.Contains(f.held, p) {
			later = append(later, i)
			continue
		}
		f.Usage.Sizes(ctx, []string{p}, func(_ int, b int64) { each(i, b) })
	}
	select {
	case <-f.release:
	case <-ctx.Done():
		return
	}
	for _, i := range later {
		f.Usage.Sizes(ctx, roots[i:i+1], func(_ int, b int64) { each(i, b) })
	}
}

func onLinux(d heldRoots) *Surveyor {
	apps := []catalog.App{
		{Name: "agent", Installs: []catalog.LivesOn{{Kinds: []catalog.MachineKind{catalog.Linux}, Install: catalog.ShellInstall{Line: "agent"}}}, Disk: catalog.Disk{Installed: []string{"/usr/local/bin/agent"}}},
		{Name: "slow", Installs: []catalog.LivesOn{{Kinds: []catalog.MachineKind{catalog.Linux}, Install: catalog.ShellInstall{Line: "slow"}}}, Disk: catalog.Disk{Paths: []catalog.Stored{{Path: "~/slow/*", Level: catalog.LevelCache}}}},
	}
	return NewSurveyor(onMachine{Usage: d.Usage}, apps, SurveyorOptions{Disk: d, ThisMachine: catalog.Linux, Expand: disk.Expander{Home: "/h"}})
}

func linuxDisk(held ...string) heldRoots {
	files := map[string]int64{"/usr/lib/libc.so": 100, "/usr/local/bin/agent": 30, "/h/slow/tree/f": 5}
	return heldRoots{Usage: fake.NewUsage(files, 1000, 500), held: held, release: make(chan struct{})}
}

func TestALinuxSystemHoldsNoAppInstalledUnderUsr(t *testing.T) {
	d := linuxDisk()
	close(d.release)
	s, err := onLinux(d).List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if s.System != 100 || s.Apps != 30 {
		t.Fatalf("system %d and apps %d, want /usr's 130 less the 30 an app installed there, and those 30 as apps", s.System, s.Apps)
	}
	got := kindSet(disk.ReadingOf(1000, 500, s, disk.HeldReading{}, "linux-1"))
	if got[disk.KindYourData] != 365 {
		t.Fatalf("kinds %+v, want your data 500 used less 100 system, 30 apps and 5 caches", got)
	}
}

func TestAKindWhoseWalkHasNotAnsweredIsAbsentAndTheReadingPending(t *testing.T) {
	d := linuxDisk("/usr", "/h/slow/tree")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	surveys := onLinux(d).Watch(ctx)
	var first disk.Survey
	select {
	case first = <-surveys:
	case <-ctx.Done():
		t.Fatal("no frame")
	}
	r := disk.ReadingOf(1000, 500, first, disk.HeldReading{}, "linux-1")
	kinds := kindSet(r)
	for _, k := range []disk.Kind{disk.KindSystem, disk.KindCaches, disk.KindYourData} {
		if _, ok := kinds[k]; ok || !r.Pending {
			t.Fatalf("while /usr and a cache are walked the kinds are %+v, pending %v; want no %s and pending: nothing unknown reads 0", r.Kinds, r.Pending, k)
		}
	}
	close(d.release)
	var last disk.Survey
	for s := range surveys {
		last = s
	}
	if r := disk.ReadingOf(1000, 500, last, disk.HeldReading{}, "linux-1"); r.Pending || len(r.Kinds) != len(disk.Kinds())-1 {
		t.Fatalf("once every walk answered the kinds are %+v, pending %v; want all but machines", r.Kinds, r.Pending)
	}
}

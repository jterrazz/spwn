package calls

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/disk"
	"github.com/jterrazz/studio-station/apps/engine/disk/adapter/fake"
	"github.com/jterrazz/studio-station/apps/engine/setup"
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
	if r := disk.ReadingOf(1000, 500, last, disk.HeldReading{}, "linux-1"); r.Pending || len(r.Kinds) != len(disk.Kinds())-2 {
		t.Fatalf("once every walk answered the kinds are %+v, pending %v; want all but machines and the image cache, a Mac's", r.Kinds, r.Pending)
	}
}

func TestEveryAppInAMacsAppFoldersIsAnAppWhoeverInstalledIt(t *testing.T) {
	files := map[string]int64{
		"/Applications/Zed.app/Contents/MacOS/zed":    40,
		"/Applications/Game.app/Contents/data":        390,
		"/h/Applications/Store.app/Contents/MacOS/st": 110,
		"/h/Music/song": 200,
	}
	usage := fake.NewUsage(files, 2000, 1000)
	apps := []catalog.App{{Name: "zed", Installs: []catalog.LivesOn{{Kinds: []catalog.MachineKind{catalog.Mac}, Install: catalog.ShellInstall{Line: "zed"}}}, Disk: catalog.Disk{Installed: []string{"/Applications/Zed.app"}}}}
	s, err := NewSurveyor(onMachine{Usage: usage}, apps, SurveyorOptions{Disk: usage, ThisMachine: catalog.Mac, Expand: disk.Expander{Home: "/h"}}).List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if s.Apps != 540 {
		t.Fatalf("apps %d, want zed's 40 once and the 500 of the apps the catalog does not carry", s.Apps)
	}
}

// value: a machine that keeps the files the survey writes, so its last
// sizes are there for the next walk.
type keptFiles struct {
	onMachine
	files map[string][]byte
}

func (m keptFiles) ReadFile(p string) ([]byte, error) {
	if b, ok := m.files[p]; ok {
		return b, nil
	}
	return m.onMachine.ReadFile(p)
}

func (m keptFiles) Keep(p string, b []byte) error {
	m.files[p] = b
	return nil
}

func TestAKindKeepsItsLastSizeWhileItsWalkAnswersAgain(t *testing.T) {
	kept := map[string][]byte{}
	surveyor := func(d heldRoots) *Surveyor {
		apps := onLinux(d).apps
		return NewSurveyor(keptFiles{onMachine{Usage: d.Usage}, kept}, apps, SurveyorOptions{Disk: d, ThisMachine: catalog.Linux, Expand: disk.Expander{Home: "/h", Roots: setup.Roots{Cache: "/h/.cache"}}})
	}
	first := linuxDisk()
	close(first.release)
	if _, err := surveyor(first).List(context.Background()); err != nil {
		t.Fatal(err)
	}

	again := linuxDisk("/usr", "/usr/local/bin/agent", "/h/slow/tree")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	surveys := surveyor(again).Watch(ctx)
	var frame disk.Survey
	select {
	case frame = <-surveys:
	case <-ctx.Done():
		t.Fatal("no frame")
	}
	close(again.release)
	for range surveys {
	}

	r := disk.ReadingOf(1000, 500, frame, disk.HeldReading{}, "linux-1")
	got := kindSet(r)
	if got[disk.KindSystem] != 100 || got[disk.KindApps] != 30 || got[disk.KindCaches] != 5 || got[disk.KindYourData] != 365 || !r.Pending {
		t.Fatalf("while every root is walked again the kinds are %+v, pending %v; want the last walk's system 100, apps 30, caches 5 and your data 365, pending", r.Kinds, r.Pending)
	}
}

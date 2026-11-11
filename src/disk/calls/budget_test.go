package calls

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/disk"
	"github.com/jterrazz/studio-station/src/disk/adapter/fake"
)

type realClock struct{}

func (realClock) Now() time.Time { return time.Unix(0, 0) }

func (realClock) Sleep(ctx context.Context, d time.Duration) error {
	waited, cancel := context.WithTimeout(ctx, d)
	defer cancel()
	<-waited.Done()
	return ctx.Err()
}

type onMachine struct {
	realClock
	*fake.Usage
}

func (m onMachine) Exists(p string) bool               { return m.Usage.Has(p) }
func (m onMachine) Names(dir string) ([]string, error) { return m.Children(dir), nil }
func (onMachine) ReadFile(string) ([]byte, error)      { return nil, errors.ErrUnsupported }
func (onMachine) Prune(string)                         {}
func (onMachine) Has(string) bool                      { return false }
func (onMachine) Keep(string, []byte) error            { return nil }
func (onMachine) Run(context.Context, string, []string, []string) (Ran, error) {
	return Ran{}, nil
}

type slowUsage struct {
	*fake.Usage
	release chan struct{}
}

func (f slowUsage) Sizes(ctx context.Context, roots []string, each func(int, int64)) {
	slow := -1
	for i, p := range roots {
		if p == "/h/slow/tree" {
			slow = i
			continue
		}
		f.Usage.Sizes(ctx, []string{p}, func(_ int, b int64) { each(i, b) })
	}
	if slow < 0 {
		return
	}
	select {
	case <-f.release:
		f.Usage.Sizes(ctx, roots[slow:slow+1], func(_ int, b int64) { each(slow, b) })
	case <-ctx.Done():
		each(slow, -1)
	}
}

func TestARowPastTheBudgetIsUnknownAndDelaysNoOtherRow(t *testing.T) {
	usage := fake.NewUsage(map[string]int64{"/h/slow/tree/f": 1, "/h/fast/tree/f": 3}, 1<<40, 1<<39)
	fs := slowUsage{Usage: usage, release: make(chan struct{})}
	defer close(fs.release)
	apps := []catalog.App{
		{Name: "slow", Installs: []catalog.LivesOn{{Kinds: []catalog.MachineKind{catalog.Mac}, Install: catalog.CaskInstall{Token: "slow"}}}, Disk: catalog.Disk{Paths: []catalog.Stored{{Path: "~/slow/*", Level: catalog.LevelCache}}}},
		{Name: "fast", Installs: []catalog.LivesOn{{Kinds: []catalog.MachineKind{catalog.Mac}, Install: catalog.CaskInstall{Token: "fast"}}}, Disk: catalog.Disk{Paths: []catalog.Stored{{Path: "~/fast/*", Level: catalog.LevelCache}}}},
	}
	s := NewSurveyor(onMachine{Usage: usage}, apps, SurveyorOptions{Disk: fs, ThisMachine: catalog.Mac, Expand: disk.Expander{Home: "/h"}, Budget: 30 * time.Millisecond})

	waited, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	listed := make(chan disk.Survey, 1)
	go func() {
		r, _ := s.List(context.Background())
		listed <- r
	}()
	var report disk.Survey
	select {
	case report = <-listed:
	case <-waited.Done():
		t.Fatal("List waited on a stuck tree past its budget")
	}
	got := map[string]disk.Tally{}
	for _, r := range report.Rows {
		got[r.Entry.Name] = r
	}
	if slow := got["slow"]; !slow.Unknown || slow.Bytes[catalog.LevelCache] != unmeasured || slow.Within != 30*time.Millisecond {
		t.Errorf("the stuck row answered %+v, want unknown sizes, not sized within the walk's budget", slow)
	}
	if fast := got["fast"]; fast.Unknown || fast.Bytes[catalog.LevelCache] != 3 {
		t.Errorf("the fast row answered %+v, want its 3 bytes", fast)
	}
}

package calls

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/disk"
	"github.com/jterrazz/studio-station/apps/engine/disk/adapter/fake"
)

type touches struct {
	mu    sync.Mutex
	paths []string
}

func (t *touches) add(p ...string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.paths = append(t.paths, p...)
}

func (t *touches) inside() []string {
	t.mu.Lock()
	defer t.mu.Unlock()
	var out []string
	for _, p := range t.paths {
		if lower := strings.ToLower(p); strings.HasPrefix(lower, "/h/library/containers") || strings.HasPrefix(lower, "/h/library/group containers") {
			out = append(out, p)
		}
	}
	return out
}

type touchingMachine struct {
	onMachine
	seen *touches
}

func (m touchingMachine) Glob(pattern string) []string {
	m.seen.add(pattern)
	return m.onMachine.Glob(pattern)
}

func (m touchingMachine) Exists(p string) bool {
	m.seen.add(p)
	return m.onMachine.Exists(p)
}

func (m touchingMachine) Names(dir string) ([]string, error) {
	m.seen.add(dir)
	return m.onMachine.Names(dir)
}

type touchingUsage struct {
	*fake.Usage
	seen *touches
}

func (u touchingUsage) Sizes(ctx context.Context, roots []string, each func(int, int64)) {
	u.seen.add(roots...)
	u.Usage.Sizes(ctx, roots, each)
}

func (u touchingUsage) Erase(ctx context.Context, paths []string) error {
	u.seen.add(paths...)
	return u.Usage.Erase(ctx, paths)
}

func TestTheSurveyNeverTouchesAnotherAppsContainer(t *testing.T) {
	usage := fake.NewUsage(map[string]int64{
		"/h/Library/Group Containers/TEAM.example.app/data.img": 15,
		"/h/Library/Containers/com.example.app/Data/x":          7,
		"/h/.example/f": 3,
	}, 1<<40, 1<<39)
	seen := &touches{}
	apps := []catalog.App{{
		Name:     "example",
		Installs: []catalog.LivesOn{{Kinds: []catalog.MachineKind{catalog.Mac}, Install: catalog.CaskInstall{Token: "example"}}},
		Disk: catalog.Disk{
			Paths: []catalog.Stored{
				{Path: "~/Library/Group Containers/TEAM.example.app", Level: catalog.LevelData},
				{Path: "~/Library/Containers/com.example.app*", Level: catalog.LevelData},
				{Path: "~/.example", Level: catalog.LevelData},
			},
			Erase: []catalog.Action{{Delete: []string{"~/Library/Group Containers/TEAM.example.app", "~/.example"}}},
		},
	}}
	s := NewSurveyor(touchingMachine{onMachine{Usage: usage}, seen}, apps, SurveyorOptions{Disk: touchingUsage{usage, seen}, ThisMachine: catalog.Mac, Expand: disk.Expander{Home: "/h"}})

	survey, err := s.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Erase(context.Background(), "example"); err != nil {
		t.Fatal(err)
	}
	if inside := seen.inside(); len(inside) > 0 {
		t.Errorf("the survey and the erase touched %q, inside another app's container", inside)
	}
	for _, row := range survey.Rows {
		if row.Entry.Name == "example" && row.Bytes[catalog.LevelData] != 3 {
			t.Errorf("the example row holds %d bytes of data, want the 3 of ~/.example alone", row.Bytes[catalog.LevelData])
		}
	}
}

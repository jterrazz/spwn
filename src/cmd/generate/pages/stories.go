package pages

import (
	"slices"
	"time"
)

var storyNow = time.Date(2026, 11, 7, 10, 30, 0, 0, time.UTC)

type World struct {
	Name  string
	Edit  func(*Scene)
	Pages []string
}

func Worlds() []World {
	return append([]World{
		{Name: "machines"},
		{Name: "machines.down", Edit: infraDown, Pages: []string{"menubar", "machine.infra", "needs", "find"}},
		{Name: "machines.fine", Edit: nothingNeedsYou, Pages: []string{"menubar", "needs", "find"}},
		{Name: "machines.empty", Edit: thisMacAlone, Pages: []string{"menubar", "find"}},
		{Name: "machines.loading", Edit: miniStillAnswering, Pages: []string{"menubar", "find"}},
	}, append(append(append(shellWorlds(), homeWorlds()...), macWorlds()...), workspaceWorlds()...)...)
}

func StoryPages() []Fixture {
	var out []Fixture
	for _, w := range Worlds() {
		f := everyMachine()
		if w.Edit != nil {
			w.Edit(&f)
		}
		for _, p := range f.draw() {
			if len(w.Pages) == 0 || slices.Contains(w.Pages, p.Name) {
				out = append(out, Fixture{Name: w.Name + "." + p.Name, Page: p.Page})
			}
		}
	}
	return out
}

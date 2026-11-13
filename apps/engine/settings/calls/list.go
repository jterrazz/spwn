package calls

import (
	"context"
	"slices"
	"sync"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/section"
	"github.com/jterrazz/studio-station/apps/engine/settings"
)

type ListIn struct{}

var List = call.Watch[Deps, ListIn, []settings.Row]{
	Name: "settings.list", Section: section.Settings, Where: call.AtTarget, Applies: settings.Applies,
	Run: func(ctx context.Context, d Deps, _ ListIn, _ call.Progress) (<-chan call.Update[[]settings.Row], error) {
		// invariant: the first value is every setting, those that live here
		// being read; each fills as its Applier answers, so the list costs its
		// slowest read.
		items := d.Settings.Items()
		off := offThisMac(d)
		type answer struct {
			i   int
			row settings.Row
		}
		rows := make([]settings.Row, len(items), len(items)+len(catalog.CheckedSettings))
		answers := make(chan answer, len(items)+len(catalog.CheckedSettings))
		left := 0
		for i, it := range items {
			if !d.Settings.Lives(it) || off[it.Name] {
				rows[i], _ = judge(ctx, d, it, off)
				continue
			}
			rows[i] = settings.Row{Name: it.Name, Item: it, Group: settings.GroupOf(it), State: settings.StateReading}
			left++
			go func() {
				row, _ := judge(ctx, d, it, off)
				answers <- answer{i, row}
			}()
		}
		expect := sync.OnceValue(func() settings.Expectation { return expectation(ctx, d, off) })
		for _, it := range catalog.CheckedSettings {
			i := len(rows)
			rows = append(rows, settings.Row{Name: it.Name, Group: settings.Checked, State: settings.StateReading})
			left++
			go func() { answers <- answer{i, judgeChecked(ctx, d, it, off, expect())} }()
		}
		out := make(chan []settings.Row)
		go func() {
			defer close(out)
			send := func() bool {
				select {
				case out <- slices.Clone(rows):
					return true
				case <-ctx.Done():
					return false
				}
			}
			if !send() {
				return
			}
			for ; left > 0; left-- {
				a := <-answers
				rows[a.i] = a.row
				if !send() {
					return
				}
			}
		}()
		return call.Updates(ctx, out), nil
	},
}

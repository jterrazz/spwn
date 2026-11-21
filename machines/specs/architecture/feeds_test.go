package architecture_test

import (
	"slices"
	"testing"
)

// invariant: rule feeds-read-no-call (ADR-070): the menu bar's pages are
// projections of the engine's store, so the package that draws them reads
// nothing: it imports no way to make a call, and a page opens on what the
// engine already holds.
func ruleFeedsReadNoCall(w *world) []finding {
	calling := []string{module + "/core/call", module + "/core/call/lanes"}
	var out []finding
	for _, f := range w.files {
		if f.test || f.pkg != "engine/feeds" {
			continue
		}
		for i, ip := range f.imports {
			if slices.Contains(calling, ip) {
				out = append(out, finding{"feeds-read-no-call", f.at(f.syntax.Imports[i].Pos()), "engine/feeds projects pages from the store: it may not import " + ip})
			}
		}
	}
	return out
}

func TestFeedsReadNoCall(t *testing.T) { judge(t, "FeedsReadNoCall") }

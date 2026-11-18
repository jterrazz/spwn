package menubar

import (
	"time"

	model "github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/overview"
	"github.com/jterrazz/studio-station/apps/engine/page"
)

func (r Reading) needsReading() NeedsReading {
	overviews := make([]overview.Summary, 0, len(r.Away)+1)
	if r.Home != nil {
		overviews = append(overviews, *r.Home)
	}
	overviews = append(overviews, r.Away...)
	return NeedsReading{Now: r.now(), Zone: r.Zone, All: r.All, Ports: r.Ports, Overviews: overviews, Asks: r.Asks}
}

// value: SPEC B5.15: the first Mac not heard from for 30 s that is not
// asleep, as the interface names it; empty when every Mac answers.
func staleMac(r Reading, self string) string {
	if r.All == nil {
		return ""
	}
	for _, h := range r.All.Macs {
		if !h.Asleep && model.Unheard(h, r.now()) {
			return page.MacName(h.Name, self)
		}
	}
	return ""
}

func (r Reading) now() time.Time {
	if !r.Now.IsZero() {
		return r.Now
	}
	if r.All != nil {
		return r.All.Read
	}
	return time.Time{}
}

func empty(all model.AllMachines) bool {
	for _, h := range all.Macs {
		if len(h.VMs) > 0 {
			return false
		}
	}
	return true
}

// value: a part the line drops by its priority, the highest first.
func dropFirst(v page.Value, pri int) page.Value {
	v.Pri = pri
	return v
}

// value: the verbs of what needs you, one per need, for Needs you's rows.
func verbs(ns []Need) []page.Action {
	out := make([]page.Action, 0, len(ns))
	for _, n := range ns {
		out = append(out, n.Verb)
	}
	return out
}

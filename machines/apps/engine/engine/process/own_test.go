package process

import (
	"maps"
	"slices"
	"testing"

	enginecalls "github.com/jterrazz/studio-station/apps/engine/engine/calls"
)

// invariant: the engine's own calls are what enginecalls.Owns says, name for
// name and type for type, so calls.json and packages/api describe the calls
// this server binds; its watches are the Streams it serves.
func TestTheServedCallsAreTheOnesOwnsDescribes(t *testing.T) {
	described := map[string]enginecalls.Own{}
	var watches []string
	for _, o := range enginecalls.Owns() {
		if o.Watch {
			watches = append(watches, o.Name)
			continue
		}
		described[o.Name] = o
	}
	bound := calls(nil, nil)
	if len(bound) != len(described) {
		t.Errorf("the engine binds %d calls of its own, Owns describes %d", len(bound), len(described))
	}
	for _, h := range bound {
		m := h.Meta()
		o, ok := described[m.Name]
		switch {
		case !ok:
			t.Errorf("%s: bound by the engine, missing from enginecalls.Owns", m.Name)
		case o.InType != m.InType || o.OutType != m.OutType:
			t.Errorf("%s: bound as %v -> %v, described as %v -> %v", m.Name, m.InType, m.OutType, o.InType, o.OutType)
		}
	}
	served := slices.Sorted(maps.Keys(Server(nil, Serving{}).Streams))
	slices.Sort(watches)
	if !slices.Equal(served, watches) {
		t.Errorf("the engine serves the watches %v, Owns describes %v", served, watches)
	}
}

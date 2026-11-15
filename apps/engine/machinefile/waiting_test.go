package machinefile_test

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/machinefile"
)

func TestALineCarriesTheVersionThatWaitsFromItsOwnFilesNight(t *testing.T) {
	// invariant: a night speaks of its own Machinefile: another file's
	// night, or none that ran, leaves every line without a waiting version.
	r, err := catalog.ParseRange("24")
	if err != nil {
		t.Fatal(err)
	}
	var node, plain machinefile.Line
	node.Step.Package, node.Step.Version = "node", r
	plain.Step.Package = "go"
	p := machinefile.Plan{Name: "m", Lines: []machinefile.Line{node, plain}}
	night := machinefile.Night{Machinefile: "m", Waiting: []machinefile.Waiting{{App: "node", Newest: "25.1.0", Range: "24", Moved: "25"}}}
	got := p.WithWaiting(machinefile.Nights{Bound: "m", Night: night, Ran: true})
	if got.Lines[0].Waiting != "25.1.0 is out" || got.Lines[1].Waiting != "" || p.Lines[0].Waiting != "" {
		t.Errorf("lines %+v, plan kept %+v", got.Lines, p.Lines)
	}
	for _, other := range []machinefile.Nights{{Bound: "m", Night: machinefile.Night{Machinefile: "other", Waiting: night.Waiting}, Ran: true}, {Bound: "m", Night: night}} {
		if w := p.WithWaiting(other).Lines[0].Waiting; w != "" {
			t.Errorf("%+v: waiting %q, want none", other, w)
		}
	}
}

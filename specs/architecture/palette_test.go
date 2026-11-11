package architecture_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/core/section"
	"github.com/jterrazz/studio-station/src/surface/tui/screens/palette"
)

// value: the targets the palette is judged on, one per machine of the
// references: This Mac with a workspace and a server, another Mac, a
// workspace, a server.
func paletteTargets() map[string][]palette.Target {
	ws := palette.Target{Name: "linux-1", Label: "linux-1", Kind: machine.Workspace, Mac: "this-mac", MacHere: true, Sections: section.For(machine.Workspace)}
	sv := palette.Target{Name: "infra", Label: "infra", Kind: machine.Server, Mac: "this-mac", MacHere: true, Sections: section.For(machine.Server)}
	here := palette.Target{Name: "this-mac", Label: "This Mac", Kind: machine.Computer, ThisMachine: true, Sections: section.For(machine.Computer), VMs: []palette.Target{ws, sv}}
	other := palette.Target{Name: "mac-mini", Label: "mac-mini", Kind: machine.Computer, Sections: section.For(machine.Computer)}
	return map[string][]palette.Target{"laptop": {here, other}, "desktop": {here, other}, "workspace": {ws}, "server": {sv}}
}

// invariant: rule palette-covers-calls, strict (TU-11): every call of
// calls.json is offered by the palette on some machine its Applies takes,
// and never on one it refuses.
func TestPaletteReachesEveryCall(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(repoRoot(t), "src", "remote", "api", "calls.json"))
	if err != nil {
		t.Fatal(err)
	}
	var c call.Catalogue
	if err = json.Unmarshal(data, &c); err != nil {
		t.Fatal(err)
	}
	metas := allCalls(t).Metas()
	for _, e := range c.Calls {
		i := slices.IndexFunc(metas, func(m call.Meta) bool { return m.Name == e.Name })
		if i < 0 {
			t.Errorf("%s: in calls.json, not in the registry", e.Name)
			continue
		}
		m := metas[i]
		reached := false
		for _, ref := range references() {
			targets := paletteTargets()[ref.name]
			offered := false
			for _, tg := range targets {
				offered = offered || slices.ContainsFunc(palette.Offer([]call.Meta{m}, tg, targets), func(it palette.Item) bool { return it.Call == m.Name })
			}
			applies := m.Applies(ref.profile).OK
			switch {
			case offered && !applies:
				t.Errorf("%s: offered on the %s, which its Applies refuses", m.Name, ref.name)
			case offered:
				reached = true
			}
		}
		if !reached {
			t.Errorf("%s: offered on no machine its Applies takes", m.Name)
		}
	}
}

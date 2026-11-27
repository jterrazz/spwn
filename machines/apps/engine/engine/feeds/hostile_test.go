package feeds

import (
	"encoding/json"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/engine/store"
	"github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/page"
)

// invariant: a far Mac whose answer holds a VM in an empty or garbled
// state, and a VM with no name, leaves the menu bar drawn: the home page and
// Find read that VM's state unknown and leave the unnamed one out.
func TestAFarMacsGarbledAnswerLeavesTheMenuBarDrawn(t *testing.T) {
	for _, answered := range []string{
		`{"Name":"mac-mini","Reachable":true,"VMs":[{"Name":"linux-3","Kind":"linux","Lifetime":"ephemeral","State":""},{"Name":"","Kind":"linux","Lifetime":"ephemeral","State":"running"}]}`,
		`{"Name":"mac-mini","Reachable":true,"VMs":[{"Name":"linux-3","Kind":"linux","Lifetime":"ephemeral","State":" \t "},{"Name":"","State":""}]}`,
		`{"Name":"mac-mini","Reachable":true,"VMs":[{"Name":"linux-3","Kind":"linux","Lifetime":"ephemeral","State":"exploded"}]}`,
	} {
		var far machines.MacView
		if err := json.Unmarshal([]byte(answered), &far); err != nil {
			t.Fatalf("the answer %s is refused: %v", answered, err)
		}
		own, minis := store.Of(store.FamilyMac, thisMac), store.Of(store.FamilyMac, mini)
		s := &store.Snapshot{At: at, Slices: map[store.Key]store.Slice{
			own:   slice(own, machines.MacView{Name: thisMac, ThisMac: true, Reachable: true}, store.Fresh, at),
			minis: slice(minis, far, store.Fresh, at),
		}}
		o := Pages{ThisMachine: page.Context{Machine: thisMac, Kind: "computer"}}
		for name, p := range map[string]page.Page{"home": o.Home(s, ""), "find": o.Find(s, "")} {
			if err := page.Check(p); err != nil {
				t.Errorf("after %s the %s page fails its checks: %v", answered, name, err)
			}
		}
		all := view{s: s, this: thisMac, now: at}.machines()
		vms := all.Macs[1].VMs
		if len(vms) != 1 || vms[0].Name != "linux-3" || vms[0].State != machines.UnknownState {
			t.Errorf("after %s the mini holds %+v, want linux-3 alone, its state unknown", answered, vms)
		}
	}
}

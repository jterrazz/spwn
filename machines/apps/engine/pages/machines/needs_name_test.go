package machines

import (
	"testing"

	model "github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/page"
)

func TestANeedsYouRowNamesItsMacAsTheMacsOwnRowDoes(t *testing.T) {
	f := model.AllMachines{Macs: []model.MacView{
		{Name: "sam-macbook-pro", User: "sam", StationVersion: "v0.2.0", ThisMac: true, Reachable: true},
		{Name: "sam-mac-mini", User: "sam", StationVersion: "v0.1.0", Reachable: true},
	}}
	var head, need *page.Row
	for _, s := range List(f, nil, page.Context{}).Sections {
		for _, it := range s.Items {
			switch {
			case s.ID == "machines" && it.Group != nil && it.Group.Head.ID == "sam-mac-mini":
				head = &it.Group.Head
			case s.ID == "needs-you" && it.Row != nil:
				need = it.Row
			}
		}
	}
	if head == nil || need == nil {
		t.Fatalf("mac row %v, needs-you row %v", head, need)
	}
	if need.Label != head.Label || need.Short != head.Short {
		t.Fatalf("needs-you row %q (%q), the Mac's row %q (%q)", need.Label, need.Short, head.Label, head.Short)
	}
}

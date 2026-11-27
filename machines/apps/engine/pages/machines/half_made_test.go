package machines

import (
	"testing"

	model "github.com/jterrazz/studio-station/apps/engine/machines"
)

func TestAHalfMadeVMSaysSoWhileItRuns(t *testing.T) {
	for _, state := range []model.VMState{model.Running, model.Stopped} {
		vm := model.VM{Name: "web", Mac: "macbook-pro", State: state, Lifetime: model.Ephemeral, Detail: model.HalfMade}
		list := listVMRow("macbook-pro", vm).Cells
		if got := list[len(list)-1].Text; got != model.HalfMade {
			t.Errorf("%s: the list's row ends %q; want %q", state, got, model.HalfMade)
		}
		row := vmRow(vm).Cells
		if got := row[len(row)-1].Text; got != model.HalfMade {
			t.Errorf("%s: the machines page's row ends %q; want %q", state, got, model.HalfMade)
		}
	}
}

package machines

import (
	"encoding/json"
	"testing"
)

func TestAStateOffTheEnumDecodesAsUnknown(t *testing.T) {
	for answered, want := range map[string]VMState{
		`""`:          UnknownState,
		`"   "`:       UnknownState,
		`"exploded"`:  UnknownState,
		`"Running"`:   UnknownState,
		`"running"`:   Running,
		`"creating"`:  Creating,
		`"unknown"`:   UnknownState,
		`"suspended"`: Suspended,
	} {
		var vm VM
		if err := json.Unmarshal([]byte(`{"Name":"linux-1","State":`+answered+`}`), &vm); err != nil {
			t.Fatalf("a state %s is refused: %v", answered, err)
		}
		if vm.State != want {
			t.Errorf("a state %s decodes as %q, want %q", answered, vm.State, want)
		}
	}
}

func TestAVMAnsweredWithNoStateDecodesAsUnknown(t *testing.T) {
	for _, answered := range []string{`{"Name":"linux-1"}`, `{"Name":"linux-1","State":null}`} {
		var vm VM
		if err := json.Unmarshal([]byte(answered), &vm); err != nil {
			t.Fatalf("%s is refused: %v", answered, err)
		}
		if vm.Name != "linux-1" || vm.State != UnknownState {
			t.Errorf("%s decodes as %+v, want linux-1 in the state unknown", answered, vm)
		}
	}
}

func TestAStateThatIsNoStringIsRefused(t *testing.T) {
	var vm VM
	if err := json.Unmarshal([]byte(`{"State":7}`), &vm); err == nil {
		t.Errorf("a state 7 decodes as %q, want it refused", vm.State)
	}
}

func TestAFarMacsVMWithNoNameIsLeftOut(t *testing.T) {
	far := MacView{VMs: []VM{{Name: "linux-3", State: Running}, {State: Running}, {Name: "api", State: Stopped}}}
	view := ViewOf("mac-mini", MacRecord{}, Node{}, far, true)
	if len(view.VMs) != 2 || view.VMs[0].Name != "linux-3" || view.VMs[1].Name != "api" {
		t.Fatalf("the mini's VMs are %+v, want linux-3 and api", view.VMs)
	}
	for _, vm := range view.VMs {
		if vm.Mac != "mac-mini" {
			t.Errorf("%s is on %q, want mac-mini", vm.Name, vm.Mac)
		}
	}
}

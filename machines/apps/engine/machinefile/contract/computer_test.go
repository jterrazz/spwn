package contract_test

import (
	"errors"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/machinefile"
)

func TestAComputerResolvesAComputersMachinefileAloneWhateverItsFilesSay(t *testing.T) {
	lib, err := library(t, map[string]string{
		"machinefiles/laptop.machine.yaml": header + "name: laptop\nmachine: { kind: computer, system: { os: macos } }\n",
	}, map[string]string{
		"machines/mini/host.machine.yaml":              computer("apps: [fd]\n"),
		"machines/mini/development-linux.machine.yaml": header + "name: development-linux\nextends: [nightly]\nmachine: { kind: workspace, system: { os: ubuntu } }\n",
		"kits/nightly.kit.yaml":                        header + "name: nightly\nrun: { upgrade: echo up }\n",
	})
	if err != nil {
		t.Fatal(err)
	}
	mini := machinefile.Machine{Name: "mini", Kind: catalog.RoleComputer}
	_, err = machinefile.Resolve(lib, "development-linux", mini, machinefile.Lock{})
	var other *machinefile.OtherKindError
	if !errors.As(err, &other) || other.Name != "development-linux" || other.This != "mini" || other.Kind != "workspace" {
		t.Errorf("mini resolved the shared workspace file with %v; want it refused as another kind's", err)
	}
	for _, name := range []string{"mini", "laptop"} {
		if _, err := machinefile.Resolve(lib, name, mini, machinefile.Lock{}); err != nil {
			t.Errorf("mini resolved %s with %v", name, err)
		}
	}
	for _, read := range []machinefile.Machine{{Name: "development-linux", Kind: catalog.RoleWorkspace}, {Name: "mini"}} {
		if _, err := machinefile.Resolve(lib, "development-linux", read, machinefile.Lock{}); err != nil {
			t.Errorf("%+v resolved the VM's file with %v; want it read", read, err)
		}
	}
}

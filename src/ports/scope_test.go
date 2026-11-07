package ports_test

import (
	"testing"

	"github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/ports"
)

// invariant: a VM's Ports page lists its shares and its listeners alone,
// under the Mac it is on, even while it is down; a Mac's lists it whole.
func TestTheVMScopeKeepsItsPortsUnderItsMac(t *testing.T) {
	infra := machine.Ref{Name: "infra", On: "jterrazz-mac-mini"}
	all := ports.AllMachines{Macs: []ports.MacPorts{
		{Name: "jterrazz-macbook-pro", ThisMac: true, Shared: []ports.SharedPort{{VM: "linux-1", GuestPort: 5173}}},
		{Name: "jterrazz-mac-mini", Range: &ports.Range{Low: 40000, High: 40999},
			Shared: []ports.SharedPort{{VM: "linux-3", GuestPort: 8080}},
			Ports:  []ports.Port{{Machine: infra, Port: 443, Answer: ports.Unreachable}, {Machine: machine.Ref{Name: "jterrazz-mac-mini"}, Port: 5000}}},
	}}
	got := ports.Of(all, "infra")
	if len(got.Macs) != 1 || got.Macs[0].Name != "jterrazz-mac-mini" || len(got.Macs[0].Shared) != 0 || got.Macs[0].Range != nil {
		t.Fatalf("infra's scope is %+v", got)
	}
	if ps := got.Macs[0].Ports; len(ps) != 1 || ps[0].Port != 443 || ps[0].Answer != ports.Unreachable {
		t.Fatalf("infra's ports are %+v", ps)
	}
	if mini := ports.Of(all, "mac-mini"); len(mini.Macs) != 1 || len(mini.Macs[0].Ports) != 2 || len(mini.Macs[0].Shared) != 1 {
		t.Fatalf("mac-mini's scope is %+v", mini)
	}
	if none := ports.Of(all, "agent-7f"); len(none.Macs) != 0 {
		t.Fatalf("a VM with no port lists %+v", none)
	}
}

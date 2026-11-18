package ports_test

import (
	"slices"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/ports"
)

// invariant: a Mac's listener on its ephemeral range is counted, never
// listed, unless it reaches outside the tailnet, is declared or is known;
// the system's own are left out uncounted.
func TestAMacsShortLivedListenersAreCountedButNeverAnExposure(t *testing.T) {
	m := ports.MacOf("macbook-pro", []ports.Listener{
		{Port: 3000, Bind: "127.0.0.1", Process: "node"},
		{Port: 50001, Bind: "127.0.0.1", Process: "node"},
		{Port: 50001, Bind: "::1", Process: "node"},
		{Port: 50002, Bind: "100.101.102.103", Process: "Code Helper"},
		{Port: 50003, Bind: "*", Process: "node"},
		{Port: 50004, Bind: "127.0.0.1", Process: catalog.SystemListeners[0]},
		{Port: 50005, Bind: "127.0.0.1", Process: "api"},
		{Port: 50005, Bind: "0.0.0.0", Process: "api"},
		{Port: 50006, Bind: "127.0.0.1", Process: "node"},
	})
	decl := []ports.Declaration{{Machine: "macbook-pro", Port: 50006, Via: "lan"}}
	kept, quiet := ports.QuietOnMac(m, 49152, 65535, decl)
	if quiet != 2 {
		t.Errorf("quiet = %d, want 2 (50001 and 50002)", quiet)
	}
	var listed []int
	for _, l := range kept.Listening {
		if !slices.Contains(listed, l.Port) {
			listed = append(listed, l.Port)
		}
	}
	if want := []int{3000, 50003, 50005, 50006}; !slices.Equal(listed, want) {
		t.Errorf("listed = %v, want %v: the ports below the range, the exposed and the declared", listed, want)
	}
}

// invariant: a VM's port its Mac remembers stopped is marked, a Mac's own
// port of the same number never is.
func TestAPortTheOwnerStoppedIsMarkedOnItsVMOnly(t *testing.T) {
	ps := []ports.Port{
		{Machine: machine.Ref{Name: "linux-1", On: "macbook-pro"}, Port: 7000},
		{Machine: machine.Ref{Name: "linux-1", On: "macbook-pro"}, Port: 5173},
		{Machine: machine.Ref{Name: "macbook-pro"}, Port: 7000},
	}
	got := ports.MarkStopped(ps, ports.Policy{NotShared: []string{"linux-1:7000", "macbook-pro:7000"}})
	if !got[0].Stopped || got[1].Stopped || got[2].Stopped {
		t.Errorf("stopped = %v %v %v, want linux-1:7000 alone", got[0].Stopped, got[1].Stopped, got[2].Stopped)
	}
}

package ports_test

import (
	"testing"

	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/ports"
)

// invariant: a right allows a port inside open and outside except; a
// workspace with no Machinefile ports opens what min_port 1024 and
// ignored_ports [22] shared, a server nothing.
func TestARightAllowsWhatItOpensAndNothingItExcepts(t *testing.T) {
	r := ports.RightsOf(&catalog.PortRights{Open: catalog.PortSet{{From: 3000, To: 3999}}, To: catalog.AudiencePrivate, Except: []int{3001}}, false)
	for port, want := range map[int]bool{2999: false, 3000: true, 3001: false, 3999: true, 8443: false} {
		if got := r.Allows(port); got != want {
			t.Errorf("Allows(%d) = %v, want %v", port, got, want)
		}
	}
	if got := ports.NotAllowed(r); got != "listens, not allowed (open: 3000-3999; except: 3001)" {
		t.Errorf("NotAllowed = %q", got)
	}
	workspace := ports.RightsOf(nil, false)
	if !workspace.Allows(1024) || !workspace.Allows(65535) || workspace.Allows(1023) || workspace.Audience() != catalog.AudiencePrivate {
		t.Errorf("a workspace's default rights = %+v, want 1024-65535 to private", workspace)
	}
	if server := ports.RightsOf(nil, true); server.Opens() || server.Allows(3000) {
		t.Errorf("a server's default rights = %+v, want none", server)
	}
	if refused := (ports.Rights{Refused: "linux-1.yaml:4: no"}); !refused.Opens() || refused.Allows(3000) {
		t.Errorf("a refused Machinefile's rights = %+v, want every port flagged", refused)
	}
}

// invariant: the guest reports from 1024 up, or from the lowest port the
// rights open below it, and never the VM's own ssh port.
func TestTheGuestReportsFromTheRightsFloorButNeverItsSSH(t *testing.T) {
	low := ports.RightsOf(&catalog.PortRights{Open: catalog.PortSet{{From: 80, To: 80}, {From: 3000, To: 3999}}, To: catalog.AudiencePrivate}, false)
	if low.Floor() != 80 || !low.Reported(80, 22) || low.Reported(53, 22) || low.Reported(22, 22) {
		t.Errorf("rights opening 80: floor %d, want 80, 53 and 22 unreported", low.Floor())
	}
	if f := ports.RightsOf(nil, false).Floor(); f != catalog.SystemPorts {
		t.Errorf("default floor = %d, want %d", f, catalog.SystemPorts)
	}
}

// invariant: a public right's ports map in order onto Funnel's.
func TestAPublicPortTakesTheFunnelPortOfItsPlace(t *testing.T) {
	r := ports.RightsOf(&catalog.PortRights{Open: catalog.PortSet{{From: 3000, To: 3000}, {From: 5000, To: 5000}, {From: 8080, To: 8080}}, To: catalog.AudiencePublic}, false)
	for port, want := range map[int]int{3000: 443, 5000: 8443, 8080: 10000, 4000: 0} {
		if got := r.PublicPort(port); got != want {
			t.Errorf("PublicPort(%d) = %d, want %d", port, got, want)
		}
	}
	if got := ports.RightsOf(nil, false).PublicPort(3000); got != 0 {
		t.Errorf("a private port's Funnel port = %d, want none", got)
	}
}

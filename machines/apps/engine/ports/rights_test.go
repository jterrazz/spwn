package ports_test

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/ports"
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
	if !workspace.Allows(1024) || !workspace.Allows(65535) || workspace.Allows(1023) || workspace.AudienceOf(1024) != catalog.AudiencePrivate {
		t.Errorf("a workspace's default rights = %+v, want 1024-65535 to private", workspace)
	}
	if server := ports.RightsOf(nil, true); server.Opens() || server.Allows(3000) {
		t.Errorf("a server's default rights = %+v, want none", server)
	}
	if refused := (ports.Rights{Refused: "linux-1.yaml:4: no"}); !refused.Opens() || refused.Allows(3000) {
		t.Errorf("a refused Machinefile's rights = %+v, want every port flagged", refused)
	}
}

// invariant: a port audiences names opens to its own audience whatever open
// and except say, counts among the public ports after open's, and is said
// by the rights' line.
func TestAPortOfAudiencesOpensToItsOwnAudience(t *testing.T) {
	r := ports.RightsOf(&catalog.PortRights{
		Open: catalog.PortSet{{From: 3000, To: 3999}}, To: catalog.AudiencePrivate, Except: []int{3306, 5432},
		Audiences: catalog.PortAudiences{5432: catalog.AudienceLocal, 3001: catalog.AudiencePublic, 80: catalog.AudiencePrivate},
	}, false)
	for port, want := range map[int]catalog.Audience{3000: catalog.AudiencePrivate, 3001: catalog.AudiencePublic, 5432: catalog.AudienceLocal, 80: catalog.AudiencePrivate} {
		if !r.Allows(port) || r.AudienceOf(port) != want {
			t.Errorf("port %d: allowed %v, audience %q; want allowed, %q", port, r.Allows(port), r.AudienceOf(port), want)
		}
	}
	if r.Allows(3306) || r.Allows(5433) {
		t.Errorf("3306 (excepted) or 5433 (outside) allowed: %+v", r)
	}
	if r.Floor() != 80 {
		t.Errorf("floor %d, want 80, the lowest port audiences names", r.Floor())
	}
	if got := r.PublicPort(3001, ports.HTTPSPort); got != 443 {
		t.Errorf("PublicPort(3001) = %d, want 443", got)
	}
	want := "It may share ports 3000 to 3999 except 3306 and 5432 with your devices, port 5432 with this Mac only, port 80 with your devices and port 3001 with the internet"
	if got := r.Line(); got != want {
		t.Errorf("Line = %q\nwant   %q", got, want)
	}
	if got := ports.NotAllowed(r); got != "listens, not allowed (open: 3000-3999; except: 3306, 5432; audiences: 80: private, 3001: public, 5432: local)" {
		t.Errorf("NotAllowed = %q", got)
	}
	public := ports.RightsOf(&catalog.PortRights{
		Open: catalog.PortSet{{From: 3000, To: 3000}, {From: 5000, To: 5000}}, To: catalog.AudiencePublic,
		Audiences: catalog.PortAudiences{3000: catalog.AudienceLocal, 9000: catalog.AudiencePublic},
	}, false)
	for port, want := range map[int]int{5000: 443, 9000: 8443, 3000: 0} {
		if got := public.PublicPort(port, ports.HTTPSPort); got != want {
			t.Errorf("PublicPort(%d) = %d, want %d: open's public ports first, then audiences', a port given local none", port, got, want)
		}
	}
	if only := ports.RightsOf(&catalog.PortRights{Audiences: catalog.PortAudiences{5432: catalog.AudienceLocal}}, false); !only.Opens() || !only.Allows(5432) || only.Allows(5433) {
		t.Errorf("rights of audiences alone = %+v, want 5432 open alone", only)
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
		if got := r.PublicPort(port, ports.HTTPSPort); got != want {
			t.Errorf("PublicPort(%d) = %d, want %d", port, got, want)
		}
	}
	if got := ports.RightsOf(nil, false).PublicPort(3000, ports.HTTPSPort); got != 0 {
		t.Errorf("a private port's Funnel port = %d, want none", got)
	}
}

// invariant: in path mode 443 carries the private paths, so a public
// right's ports map onto 8443 and 10000 only, and a third gets none.
func TestPathModeNeverFunnels443(t *testing.T) {
	r := ports.RightsOf(&catalog.PortRights{Open: catalog.PortSet{{From: 3000, To: 3000}, {From: 5000, To: 5000}, {From: 8080, To: 8080}}, To: catalog.AudiencePublic}, false)
	for port, want := range map[int]int{3000: 8443, 5000: 10000, 8080: 0} {
		if got := r.PublicPort(port, ports.HTTPSPath); got != want {
			t.Errorf("PublicPort(%d, path) = %d, want %d", port, got, want)
		}
	}
	if _, keep := r.Judge(ports.Reservation{VM: "linux-1", GuestPort: 8080}, ports.HTTPSPath); keep {
		t.Error("a public port no Funnel port is left for is kept, want released")
	}
}

// invariant: the rights judge a reservation again as they are now: the
// one they made and no longer allow goes, the owner's stays private, and
// a kept one takes today's audience.
func TestARightsMadeReservationIsReleasedWhenTheRightsNarrow(t *testing.T) {
	private := ports.RightsOf(&catalog.PortRights{Open: catalog.PortSet{{From: 3000, To: 3999}}, To: catalog.AudiencePrivate, Except: []int{3001}}, false)
	public := ports.Reservation{VM: "linux-1", GuestPort: 3000, MacPort: 40000, To: catalog.AudiencePublic, PublicPort: 443}
	if got, keep := private.Judge(public, ports.HTTPSPort); !keep || got.Audience() != catalog.AudiencePrivate || got.PublicPort != 0 {
		t.Errorf("a public reservation under private rights = %+v kept %v, want private", got, keep)
	}
	for _, port := range []int{3001, 5173} {
		if _, keep := private.Judge(ports.Reservation{VM: "linux-1", GuestPort: port}, ports.HTTPSPort); keep {
			t.Errorf("rights-made %d outside the rights is kept, want released", port)
		}
		if got, keep := private.Judge(ports.Reservation{VM: "linux-1", GuestPort: port, By: ports.ByOwner}, ports.HTTPSPort); !keep || got.Audience() != catalog.AudiencePrivate {
			t.Errorf("owner-made %d = %+v kept %v, want kept private", port, got, keep)
		}
	}
	if _, keep := (ports.Rights{Refused: "linux-1.yaml:4: no"}).Judge(ports.Reservation{VM: "linux-1", GuestPort: 3000}, ports.HTTPSPort); keep {
		t.Error("a reservation under a refused Machinefile is kept, want released")
	}
}

// invariant: an audience the owner chose wins over the rights, inside them
// or outside, and a rights-made reservation never keeps one.
func TestAnAudienceTheOwnerChoseWinsOverTheRights(t *testing.T) {
	public := ports.RightsOf(&catalog.PortRights{Open: catalog.PortSet{{From: 3000, To: 3000}}, To: catalog.AudiencePublic}, false)
	local := ports.Reservation{VM: "linux-1", GuestPort: 3000, MacPort: 40000, By: ports.ByOwner, Chosen: catalog.AudienceLocal}
	if got, keep := public.Judge(local, ports.HTTPSPort); !keep || got.Audience() != catalog.AudienceLocal || got.PublicPort != 0 {
		t.Errorf("an owner's local share under public rights = %+v, %v, want local, kept", got, keep)
	}
	outside := ports.Reservation{VM: "linux-1", GuestPort: 5000, MacPort: 40001, By: ports.ByOwner, Chosen: catalog.AudiencePublic, PublicPort: 8443}
	if got, keep := public.Judge(outside, ports.HTTPSPort); !keep || got.Audience() != catalog.AudiencePublic || got.PublicPort != 8443 {
		t.Errorf("an owner's public share outside the rights = %+v, %v, want public on 8443, kept", got, keep)
	}
	rights := ports.Reservation{VM: "linux-1", GuestPort: 3000, MacPort: 40000, By: ports.ByRights, Chosen: catalog.AudienceLocal}
	if got, _ := public.Judge(rights, ports.HTTPSPort); got.Audience() != catalog.AudiencePublic || got.PublicPort != 443 {
		t.Errorf("a rights-made reservation = %+v, want the rights' public on 443", got)
	}
}

// invariant: the rights line names each span, what it excepts and its
// audience in words, and a machine that opens nothing shares only on demand.
func TestTheRightsLineSaysWhatAVMMayShareAndWithWhom(t *testing.T) {
	for _, c := range []struct {
		r    ports.Rights
		want string
	}{
		{ports.RightsOf(nil, false), "It may share ports 1024 to 65535 with your devices"},
		{ports.RightsOf(&catalog.PortRights{Open: catalog.PortSet{{From: 3000, To: 3000}}, To: catalog.AudiencePublic}, false), "It may share port 3000 with the internet"},
		{ports.RightsOf(&catalog.PortRights{Open: catalog.PortSet{{From: 3000, To: 3999}, {From: 5173, To: 5173}, {From: 8080, To: 8080}}, To: catalog.AudienceLocal, Except: []int{3306, 3307}}, false), "It may share ports 3000 to 3999, 5173 and 8080 except 3306 and 3307 with this Mac only"},
		{ports.RightsOf(nil, true), "It shares nothing by itself; you share a port when you need it"},
		{ports.Rights{Refused: "linux-1.machine.yaml:4: no"}, "It shares nothing: linux-1.machine.yaml:4: no"},
	} {
		if got := c.r.Line(); got != c.want {
			t.Errorf("Line() = %q, want %q", got, c.want)
		}
	}
}

package ports_test

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/core/finding"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/ports"
)

func listening(binds ...string) []ports.Listener {
	out := make([]ports.Listener, 0, len(binds))
	for _, b := range binds {
		out = append(out, ports.Listener{Port: 9000, Bind: b, Process: "node"})
	}
	return out
}

func TestWhoReachesAPortDependsOnItsAddressAndWhereItsMachineSits(t *testing.T) {
	cases := []struct {
		at    ports.Placement
		binds []string
		red   bool
		class ports.Class
		reach ports.Reach
	}{
		{ports.OnMac, []string{"127.0.0.1", "::1"}, false, ports.Private, ports.ThisMachine},
		{ports.OnMac, []string{"100.64.0.2"}, false, ports.Private, ports.Tailnet},
		{ports.OnMac, []string{"fd7a:115c:a1e0::2"}, false, ports.Private, ports.Tailnet},
		{ports.OnMac, []string{"*"}, true, ports.Public, ports.Outside},
		{ports.OnMac, []string{"127.0.0.1", "192.168.1.20"}, true, ports.Public, ports.Outside},
		{ports.OnMac, []string{"::"}, true, ports.Public, ports.Outside},
		{ports.OnMac, []string{"*", "127.0.0.1"}, true, ports.Public, ports.Outside},
		{ports.OnMac, []string{"100.64.0.2", "::1"}, false, ports.Private, ports.Tailnet},
		{ports.BehindMac, []string{"0.0.0.0"}, false, ports.Private, ports.Tailnet},
		{ports.BehindMac, []string{"127.0.0.1"}, false, ports.Private, ports.ThisMachine},
		{ports.Bridged, []string{"0.0.0.0"}, true, ports.Public, ports.Outside},
		{ports.Bridged, []string{"[fe80::1%en0]"}, true, ports.Public, ports.Outside},
	}
	for _, c := range cases {
		m := ports.Machine{Ref: machine.Ref{Name: "m"}, At: c.at, Listening: listening(c.binds...)}
		got := ports.Classify(m, nil)
		if len(got) != 1 {
			t.Fatalf("%s %v: %d ports, want one per number", c.at, c.binds, len(got))
		}
		if p := got[0]; p.Red() != c.red || p.Class() != c.class || p.Reach != c.reach {
			t.Errorf("%s %v: red %v class %s reach %d, want %v %s %d", c.at, c.binds, p.Red(), p.Class(), p.Reach, c.red, c.class, c.reach)
		}
	}
}

func TestADeclarationTakesARedPortOutOfNeedsYouButNotOutOfPublic(t *testing.T) {
	mini := machine.Ref{Name: "jterrazz-mac-mini"}
	m := ports.Machine{Ref: mini, At: ports.OnMac, Listening: []ports.Listener{
		{Port: 5000, Bind: "*", Process: "ControlCenter"},
		{Port: 9000, Bind: "*", Process: "node"},
		{Port: 49152, Bind: "*", Process: "rapportd"},
	}}
	declared := []ports.Declaration{{Machine: mini.Name, Port: 9000, Via: "lan"}, {Machine: "infra", Port: 5000, Via: "lan"}}
	ps := ports.Classify(m, declared)
	if len(ps) != 2 {
		t.Fatalf("%d ports, want the system's own unknown listener left out", len(ps))
	}
	found := ports.Exposed(ps)
	if len(found) != 1 || found[0].Subject != "jterrazz-mac-mini:5000" || found[0].Severity != finding.Broken || found[0].Fix.Path == "" {
		t.Fatalf("findings %+v, want 5000 alone, broken, with where to turn it off", found)
	}
	if ps[1].Class() != ports.Public || ps[1].Red() {
		t.Errorf("a declared port reachable outside is public and not red: %+v", ps[1])
	}
}

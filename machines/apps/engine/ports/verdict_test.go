package ports_test

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/core/verdict"
	"github.com/jterrazz/studio-station/apps/engine/ports"
)

func TestThePortsHeadSaysWhatIsExposedElseWhatListens(t *testing.T) {
	quiet := ports.Port{Port: 3000, Reach: ports.ThisMachine}
	red := ports.Port{Port: 5432, Reach: ports.Outside, Answer: ports.Answers}
	of := func(ps []ports.Port, shared int) ports.AllMachines {
		return ports.AllMachines{Macs: []ports.MacPorts{{Ports: ps, Shared: make([]ports.SharedPort, shared)}}}
	}
	for name, c := range map[string]struct {
		f    ports.AllMachines
		want verdict.Verdict
	}{
		"nothing":   {of(nil, 0), verdict.Verdict{Tone: verdict.Plain, Sentence: ports.SentenceNothingListens, Line: ports.LineNothingListens}},
		"one":       {of([]ports.Port{quiet}, 0), verdict.Verdict{Tone: verdict.OK, Sentence: "One port listens.", Line: "None is shared."}},
		"shared":    {of([]ports.Port{quiet, quiet}, 2), verdict.Verdict{Tone: verdict.OK, Sentence: "2 ports listen.", Line: "2 are shared."}},
		"exposed":   {of([]ports.Port{quiet, red}, 1), verdict.Verdict{Tone: verdict.Bad, Sentence: "Port 5432 is exposed.", Line: ports.LineExposed}},
		"two of it": {of([]ports.Port{red, red}, 0), verdict.Verdict{Tone: verdict.Bad, Sentence: "2 ports are exposed.", Line: ports.LineExposed}},
	} {
		if got := ports.VerdictOf(c.f); got != c.want {
			t.Errorf("%s: %+v; want %+v", name, got, c.want)
		}
	}
}

func TestOneMachinesPortsHeadCountsItsOwnAndTheShortLivedLeftOut(t *testing.T) {
	on := func(vm string, port int) ports.Port {
		return ports.Port{Machine: machine.Ref{Name: vm, On: "macbook"}, Port: port, Reach: ports.ThisMachine}
	}
	all := ports.AllMachines{Macs: []ports.MacPorts{{Name: "macbook", ThisMac: true,
		Ports: []ports.Port{on("macbook", 22), on("linux-1", 3000), on("linux-1", 8080), on("linux-2", 5432), on("linux-2", 6379)},
		Quiet: []ports.Quiet{{VM: "linux-1", Count: 3}, {VM: "linux-2", Count: 1}}, MacQuiet: 2}}}
	if got := ports.VerdictOf(ports.Of(all, "linux-1")); got.Sentence != "2 ports listen." || got.Line != "None is shared. 3 short-lived listeners are left out." {
		t.Errorf("linux-1's head: %+v", got)
	}
	if got := ports.VerdictOf(ports.Of(all, "linux-2")); got.Line != "None is shared. One short-lived listener is left out." {
		t.Errorf("linux-2's head: %+v", got)
	}
	if got := ports.VerdictOf(all); got.Sentence != "5 ports listen." || got.Line != "None is shared. 6 short-lived listeners are left out." {
		t.Errorf("the whole listing's head: %+v", got)
	}
}

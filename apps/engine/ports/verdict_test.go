package ports_test

import (
	"testing"

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

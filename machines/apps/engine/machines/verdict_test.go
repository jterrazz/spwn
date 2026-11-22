package machines_test

import (
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/core/verdict"
	"github.com/jterrazz/studio-station/apps/engine/machines"
)

func TestAServersHealthIsItsFirstStepThatDoesNotPass(t *testing.T) {
	up := machines.VM{Name: "infra", Lifetime: machines.AlwaysOn, State: machines.Running, Want: "running", IP: "192.168.64.5", SSH: true,
		Check: "https://infra/health", CheckStatus: machines.CheckStatus{State: machines.CheckPassing}}
	for name, c := range map[string]struct {
		change func(*machines.VM)
		want   verdict.Verdict
	}{
		"all four pass":   {func(*machines.VM) {}, verdict.Verdict{Tone: verdict.OK, Sentence: machines.SentenceHealthy}},
		"its check fails": {func(v *machines.VM) { v.CheckStatus.State = machines.CheckFailing }, verdict.Verdict{Tone: verdict.Bad, Sentence: machines.SentenceDown}},
		"its check fails since 10:18": {func(v *machines.VM) {
			v.CheckStatus = machines.CheckStatus{State: machines.CheckFailing, Since: time.Date(2026, 11, 18, 10, 18, 0, 0, time.UTC)}
		}, verdict.Verdict{Tone: verdict.Bad, Sentence: machines.SentenceDown, Line: "Its check fails since 10:18."}},
		"no check set":   {func(v *machines.VM) { v.Check = "" }, verdict.Verdict{Tone: verdict.OK, Sentence: machines.SentenceHealthy, Line: machines.LineNoCheck}},
		"SSH silent":     {func(v *machines.VM) { v.SSH = false }, verdict.Verdict{Tone: verdict.Bad, Sentence: machines.SentenceNoSSH}},
		"no address":     {func(v *machines.VM) { v.IP = "" }, verdict.Verdict{Tone: verdict.Warn, Sentence: machines.SentenceNoAddress}},
		"in a loop":      {func(v *machines.VM) { v.Restarts.Looping, v.Cause = true, "out of memory" }, verdict.Verdict{Tone: verdict.Bad, Sentence: machines.SentenceDown, Line: "out of memory"}},
		"restarting":     {func(v *machines.VM) { v.Restart = &machines.RestartWalk{} }, verdict.Verdict{Tone: verdict.New, Sentence: machines.SentenceRestarting}},
		"stopped by you": {func(v *machines.VM) { v.State, v.Want = machines.Stopped, "stopped" }, verdict.Verdict{Tone: verdict.Plain, Sentence: "Stopped."}},
	} {
		vm := up
		c.change(&vm)
		if got := machines.HealthVerdict(vm, false); got == nil || *got != c.want {
			t.Errorf("%s: %+v; want %+v", name, got, c.want)
		}
	}
	if machines.HealthVerdict(machines.VM{Lifetime: machines.Persistent, State: machines.Running}, false) != nil {
		t.Errorf("a workspace has a server's health verdict")
	}
}

func TestTheEngineSaysHowLittleOfAChecksLatencyItKept(t *testing.T) {
	vm := machines.VM{Lifetime: machines.AlwaysOn, Check: "infra:443"}
	if got := machines.LatencyNote(vm); got != "The engine kept no check so far." {
		t.Errorf("no point: %q", got)
	}
	vm.Latency = []machines.LatencyPoint{{At: time.Unix(0, 0)}}
	if got := machines.LatencyNote(vm); got != "The engine kept one check so far." {
		t.Errorf("one point: %q", got)
	}
	vm.Latency = append(vm.Latency, machines.LatencyPoint{At: time.Unix(60, 0)})
	if got := machines.LatencyNote(vm); got != "" {
		t.Errorf("two points draw a chart, yet: %q", got)
	}
	f := machines.Judged(machines.AllMachines{Macs: []machines.MacView{{VMs: []machines.VM{vm}}}})
	if f.Macs[0].VMs[0].Health == nil {
		t.Errorf("machines.list leaves a server unjudged")
	}
}

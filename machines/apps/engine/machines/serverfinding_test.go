package machines

import (
	"slices"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/core/finding"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
)

func TestAServerWhoseCheckFailsNeedsYouAndOneNotAnsweringYetDoesNot(t *testing.T) {
	ref := machine.Ref{Name: "api", On: "mini"}
	server := VM{Name: "api", Lifetime: AlwaysOn, State: Running, CheckStatus: CheckStatus{State: CheckFailing, Since: time.Date(2026, 11, 14, 10, 1, 0, 0, time.Local)}}
	f, ok := vmFinding(ref, server, "", false)
	if !ok || f.Severity != finding.Broken || f.Fix.Call != "machines.restart" || f.Fix.Args["Name"] != "api" || f.Fix.On != "mini" || f.Fix.Confirm != "" || f.Text != "api check failing since 10:01" || f.Detail != "mini. Since 10:01." {
		t.Fatalf("a server whose check fails: %+v, %v", f, ok)
	}
	server.CheckStatus = CheckStatus{State: CheckWaiting}
	if f, ok := vmFinding(ref, server, "", false); ok {
		t.Errorf("a check that has not answered yet is a finding: %+v", f)
	}
}

func TestADownServerSaysSSHAnswersOnlyWhenItsConnectionOpened(t *testing.T) {
	ref := machine.Ref{Name: "api", On: "mini"}
	server := VM{Name: "api", Lifetime: AlwaysOn, State: Running, SSH: true, CheckStatus: CheckStatus{State: CheckFailing, Since: time.Date(2026, 11, 14, 14, 2, 0, 0, time.Local)}}
	if f, _ := vmFinding(ref, server, "", false); f.Detail != "mini. Since 14:02. SSH still answers." {
		t.Errorf("detail %q", f.Detail)
	}
}

func TestAnotherMacWhoseEngineIsNotInstalledIsToldSoWithItsOn(t *testing.T) {
	f := AllMachines{Macs: []MacView{
		{Name: "macbook-pro", ThisMac: true, Engine: EngineView{State: EngineRunning}},
		{Name: "mac-mini", Reachable: true, Engine: EngineView{State: EngineNotInstalled}},
	}}
	fs := Findings(f)
	if len(fs) != 1 || fs[0].Text != "mac-mini has no engine installed" || fs[0].Fix.Call+fs[0].Fix.Command+fs[0].Fix.Path != "" || fs[0].Severity != finding.NeedsALook {
		t.Errorf("findings %+v", fs)
	}
}

func TestThisMacsStoppedEngineIsStartedFromATerminal(t *testing.T) {
	f := AllMachines{Macs: []MacView{{Name: "macbook-pro", ThisMac: true, Reachable: true, Engine: EngineView{State: EngineNotInstalled}}}}
	fs := Findings(f)
	if len(fs) != 1 || fs[0].Fix.Command != "station engine start" || !fs[0].Fix.InTerminal {
		t.Errorf("findings %+v", fs)
	}
}

func TestAServerInALoopIsOfferedItsRestartAndABrokenOneStopsOrRestarts(t *testing.T) {
	ref := machine.Ref{Name: "api", On: "mini"}
	now := time.Date(2026, 11, 14, 10, 0, 0, 0, time.UTC)
	recent := []time.Time{now, now, now, now, now}
	server := VM{Name: "api", Lifetime: AlwaysOn, State: Broken, Restarts: Restarts{Looping: true, Recent: recent}}
	f, ok := vmFinding(ref, server, "", false)
	if !ok || f.Severity != finding.Broken || f.Fix.Call != "machines.restart" || f.Fix.Args["Name"] != "api" || f.Fix.On != "mini" || f.Fix.Confirm != "" || f.Text != "api is in a loop: ↺ 5 in 10 min" {
		t.Fatalf("a server in a loop: %+v, %v", f, ok)
	}
	if got := GesturesOf(server); !slices.Equal(got, []Gesture{GestureStop, GestureRestart}) {
		t.Errorf("a broken server offers %v; want stop and restart", got)
	}
	broken := VM{Name: "api", Lifetime: AlwaysOn, State: Broken, Detail: BrokenDetail()}
	f, ok = vmFinding(ref, broken, "", false)
	if !ok || f.Severity != finding.Broken || f.Fix.Call != "machines.restart" || f.Fix.Command != "" || f.Fix.Args["Name"] != "api" || f.Fix.On != "mini" || f.Fix.Confirm != "" || f.Text != "api is broken: rebooting in a loop" {
		t.Errorf("a broken server: %+v, %v", f, ok)
	}
	broken.Lifetime = Persistent
	if f, _ = vmFinding(ref, broken, "", false); f.Fix.Command != "station machines delete api" || f.Text != "api rebooting in a loop, stopped" {
		t.Errorf("a broken workspace: %+v", f)
	}
	server.Lifetime = Persistent
	if got := GesturesOf(server); got != nil {
		t.Errorf("a broken workspace offers %v; want none", got)
	}
}

func TestAVMWhoseCreateNoStartCanFinishIsOfferedItsDeleteAlone(t *testing.T) {
	ref := machine.Ref{Name: "mac-1", On: "mini"}
	vm := VM{Name: "mac-1", Lifetime: Ephemeral, State: Stopped, Birth: &Birth{Failed: true, Gone: true, Step: BornSSH}}
	f, ok := vmFinding(ref, vm, " --on mini", false)
	if !ok || f.Severity != finding.Broken || !f.Again || f.Fix.Call != "machines.delete" || f.Fix.Args["VM"] != "mac-1" || f.Fix.On != "mini" || f.Fix.Confirm != "mac-1" || f.Fix.Command != "station machines delete mac-1 --on mini" || f.Text != "mac-1 was not created: it failed at SSH" {
		t.Errorf("a create no start can finish: %+v, %v", f, ok)
	}
	if got := GesturesOf(vm); got != nil {
		t.Errorf("a create no start can finish offers %v; want none, its delete on its page", got)
	}
}

func TestEveryVMFindingButTheTwoPathOnesNamesTheCallThatFixesIt(t *testing.T) {
	ref := machine.Ref{Name: "vm-1", On: "mini"}
	for name, c := range map[string]struct {
		vm      VM
		call    string
		arg     string
		confirm string
	}{
		"powered off inside": {VM{Name: "vm-1", Lifetime: AlwaysOn, State: Stopped, Detail: PoweredOffInside}, "machines.start", "VM", ""},
		"ended unexpectedly": {VM{Name: "vm-1", Lifetime: Persistent, State: Stale}, "machines.start", "VM", ""},
		"broken workspace":   {VM{Name: "vm-1", Lifetime: Persistent, State: Broken, Detail: BrokenDetail()}, "machines.delete", "VM", "vm-1"},
		"guest absent":       {VM{Name: "vm-1", Lifetime: Persistent, State: Running, Guest: "absent"}, "machines.restart", "Name", ""},
	} {
		f, ok := vmFinding(ref, c.vm, " --on mini", false)
		if !ok || f.Fix.Call != c.call || f.Fix.On != "mini" || f.Fix.Confirm != c.confirm || f.Fix.Command == "" {
			t.Errorf("%s: %+v, %v", name, f, ok)
			continue
		}
		if f.Fix.Args[c.arg] != "vm-1" {
			t.Errorf("%s: args %v", name, f.Fix.Args)
		}
	}
}

func TestASleepingMacThatSaidItsHardwareAddressIsOfferedItsWake(t *testing.T) {
	sleeper := MacView{Name: "mac-mini", Asleep: true, Hardware: machine.Hardware{Ether: "aa:bb:cc:dd:ee:ff"}, VMs: []VM{{Name: "infra", Lifetime: AlwaysOn, State: Running}}}
	f := AllMachines{Macs: []MacView{{Name: "macbook-pro", ThisMac: true, Engine: EngineView{State: EngineRunning}}, sleeper}}
	fs := Findings(f)
	if len(fs) != 1 || fs[0].Fix.Call != "machines.wake" || fs[0].Fix.Args["Name"] != "mac-mini" || fs[0].Fix.On != "macbook-pro" {
		t.Fatalf("a sleeping Mac with an address: %+v", fs)
	}
	sleeper.Hardware = machine.Hardware{}
	f.Macs[1] = sleeper
	if fs := Findings(f); len(fs) != 1 || fs[0].Fix.Call != "" {
		t.Errorf("a sleeping Mac with no address is offered no wake: %+v", fs)
	}
}

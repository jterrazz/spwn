package machines

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/core/finding"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
)

const fixOnMini = "approve station on mac-mini's screen, or in System Settings > Privacy & Security > Local Network"

func TestARunningVMItsMacBlocksIsUnreachableNamingTheMacAndItsFix(t *testing.T) {
	ref := machine.Ref{Name: "linux-2", On: "mac-mini"}
	for _, vm := range []VM{
		{Name: "linux-2", Lifetime: Ephemeral, State: Running, Guest: "absent", Blocked: LocalNetworkBlock},
		{Name: "linux-2", Lifetime: AlwaysOn, State: Running, Guest: "absent", Blocked: LocalNetworkBlock, Check: "https://x", CheckStatus: CheckStatus{State: CheckFailing}},
	} {
		f, ok := vmFinding(ref, vm, " --on mini", false)
		if !ok || f.Text != "linux-2 is unreachable: mac-mini has not approved station for Local Network" || f.Fix.Path != fixOnMini || f.Fix.Command != "" || f.Fix.Call != "" {
			t.Errorf("%s blocked: %+v, %v", vm.Lifetime, f, ok)
		}
	}
}

func TestAVMNotRunningIsNeverToldBlocked(t *testing.T) {
	vm := VM{Name: "linux-2", Lifetime: Persistent, State: Stopped, Blocked: LocalNetworkBlock}
	if f, ok := vmFinding(machine.Ref{Name: "linux-2", On: "mac-mini"}, vm, "", false); ok {
		t.Errorf("a stopped VM is told: %+v", f)
	}
}

func TestABlockedServerFailsSSHWithItsMacNamed(t *testing.T) {
	server := VM{Name: "infra", Mac: "mac-mini", Lifetime: AlwaysOn, State: Running, IP: "192.168.64.3", Guest: "absent", Blocked: LocalNetworkBlock}
	v := HealthVerdict(server, false)
	if v.Sentence != SentenceNoSSH || v.Line != "mac-mini has not approved station for Local Network." {
		t.Errorf("a blocked server: %+v", v)
	}
	server.Blocked = ""
	if v := HealthVerdict(server, false); v.Line != "" {
		t.Errorf("a server whose SSH fails otherwise says %q", v.Line)
	}
}

func TestAMacWhoseEngineReadsNoApprovalNeedsALookNamingIt(t *testing.T) {
	for _, tc := range []struct {
		read LocalNetwork
		want bool
	}{{LocalNetworkNotApproved, true}, {LocalNetworkApproved, false}, {"", false}} {
		mac := MacView{Name: "mac-mini", ThisMac: true, Reachable: true, Engine: EngineView{State: EngineRunning, LocalNetwork: tc.read}}
		var got []finding.Finding
		for _, f := range Findings(AllMachines{Macs: []MacView{mac}}) {
			if f.Fix.Path == fixOnMini {
				got = append(got, f)
			}
		}
		switch {
		case tc.want && (len(got) != 1 || got[0].Text != "mac-mini has not approved station for Local Network" || got[0].Severity != finding.NeedsALook):
			t.Errorf("read %q: %+v, want one look naming mac-mini", tc.read, got)
		case !tc.want && len(got) != 0:
			t.Errorf("read %q: %+v, want none", tc.read, got)
		}
	}
}

func TestTheFixNamesTheAppTheBlockingMacsEngineRunsAs(t *testing.T) {
	ref := machine.Ref{Name: "linux-2", On: "mac-mini"}
	vm := VM{Name: "linux-2", Lifetime: Ephemeral, State: Running, Guest: "absent", Blocked: LocalNetworkBlock}
	if f, _ := vmFinding(ref, vm, "", true); f.Text != "linux-2 is unreachable: mac-mini has not approved Spwn for Local Network" {
		t.Errorf("a VM blocked on a Mac whose engine Spwn.app carries says %q", f.Text)
	}
	if f, _ := vmFinding(ref, vm, "", true); f.Fix.Path != "approve Spwn on mac-mini's screen, or in System Settings > Privacy & Security > Local Network" {
		t.Errorf("a VM blocked on a Mac whose engine Spwn.app carries: %q", f.Fix.Path)
	}
	mac := MacView{Name: "mac-mini", ThisMac: true, Reachable: true, Engine: EngineView{State: EngineRunning, LocalNetwork: LocalNetworkNotApproved, Bundled: true}}
	for _, f := range Findings(AllMachines{Macs: []MacView{mac}}) {
		if f.Fix.Path == fixOnMini {
			t.Errorf("a Mac whose engine Spwn.app carries is told the bare binary's row: %+v", f)
		}
	}
}

func TestAnEngineIsInTheBundleOnlyAsTheFileSpwnAppCarries(t *testing.T) {
	for executable, want := range map[string]bool{
		"/Users/me/Applications/Spwn.app/Contents/MacOS/Spwn Engine": true,
		"/Users/me/.local/bin/station":                               false,
		"/Users/me/Applications/Spwn.app/Contents/MacOS/spwn-window": false,
		"": false,
	} {
		if got := InBundle(executable); got != want {
			t.Errorf("InBundle(%q) = %v, want %v", executable, got, want)
		}
	}
}

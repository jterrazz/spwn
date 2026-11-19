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
		f, ok := vmFinding(ref, vm, " --on mini")
		if !ok || f.Text != "linux-2 is unreachable: mac-mini has not approved station for Local Network" || f.Fix.Path != fixOnMini || f.Fix.Command != "" || f.Fix.Call != "" {
			t.Errorf("%s blocked: %+v, %v", vm.Lifetime, f, ok)
		}
	}
}

func TestAVMNotRunningIsNeverToldBlocked(t *testing.T) {
	vm := VM{Name: "linux-2", Lifetime: Persistent, State: Stopped, Blocked: LocalNetworkBlock}
	if f, ok := vmFinding(machine.Ref{Name: "linux-2", On: "mac-mini"}, vm, ""); ok {
		t.Errorf("a stopped VM is told: %+v", f)
	}
}

func TestABlockedServerFailsSSHWithItsMacNamed(t *testing.T) {
	server := VM{Name: "infra", Mac: "mac-mini", Lifetime: AlwaysOn, State: Running, IP: "192.168.64.3", Guest: "absent", Blocked: LocalNetworkBlock}
	v := HealthVerdict(server)
	if v.Sentence != SentenceNoSSH || v.Line != "mac-mini has not approved station for Local Network." {
		t.Errorf("a blocked server: %+v", v)
	}
	server.Blocked = ""
	if v := HealthVerdict(server); v.Line != "" {
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

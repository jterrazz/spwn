package machines

import (
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/core/finding"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
)

func TestAServerWhoseCheckFailsNeedsYouAndOneNotAnsweringYetDoesNot(t *testing.T) {
	ref := machine.Ref{Name: "api", On: "mini"}
	server := VM{Name: "api", Lifetime: AlwaysOn, State: Running, CheckStatus: CheckStatus{State: CheckFailing, Since: time.Date(2026, 11, 14, 10, 1, 0, 0, time.Local)}}
	f, ok := vmFinding(ref, server, "")
	if !ok || f.Severity != finding.Broken || f.Fix.Call != "machines.restart" || f.Fix.Args["Name"] != "api" || f.Fix.On != "mini" || f.Fix.Confirm != "api" || f.Text != "api check failing since 10:01" || f.Detail != "mini. Since 10:01." {
		t.Fatalf("a server whose check fails: %+v, %v", f, ok)
	}
	server.CheckStatus = CheckStatus{State: CheckWaiting}
	if f, ok := vmFinding(ref, server, ""); ok {
		t.Errorf("a check that has not answered yet is a finding: %+v", f)
	}
}

func TestADownServerSaysSSHAnswersOnlyWhenItsConnectionOpened(t *testing.T) {
	ref := machine.Ref{Name: "api", On: "mini"}
	server := VM{Name: "api", Lifetime: AlwaysOn, State: Running, SSH: true, CheckStatus: CheckStatus{State: CheckFailing, Since: time.Date(2026, 11, 14, 14, 2, 0, 0, time.Local)}}
	if f, _ := vmFinding(ref, server, ""); f.Detail != "mini. Since 14:02. SSH still answers." {
		t.Errorf("detail %q", f.Detail)
	}
}

func TestAnotherMacWhoseEngineIsNotInstalledIsToldSoWithItsOn(t *testing.T) {
	f := AllMachines{Macs: []MacView{
		{Name: "macbook-pro", ThisMac: true, Engine: EngineView{State: EngineRunning}},
		{Name: "mac-mini", Reachable: true, Engine: EngineView{State: EngineNotInstalled}},
	}}
	fs := Findings(f)
	if len(fs) != 1 || fs[0].Text != "mac-mini has no engine installed" || fs[0].Fix.Command != "station engine start --on mac-mini" || fs[0].Severity != finding.NeedsALook {
		t.Errorf("findings %+v", fs)
	}
}

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
	f, ok := serverFinding(ref, server, "")
	if !ok || f.Severity != finding.Broken || f.Fix.Call != "machines.restart" || f.Fix.Args["Name"] != "api" || f.Fix.On != "mini" || f.Fix.Confirm != "api" || f.Text != "api check failing since 10:01" || f.Detail != "mini. Since 10:01." {
		t.Fatalf("a server whose check fails: %+v, %v", f, ok)
	}
	server.CheckStatus = CheckStatus{State: CheckWaiting}
	if f, ok := serverFinding(ref, server, ""); ok {
		t.Errorf("a check that has not answered yet is a finding: %+v", f)
	}
}

func TestADownServerSaysSSHAnswersOnlyWhenItsConnectionOpened(t *testing.T) {
	ref := machine.Ref{Name: "api", On: "mini"}
	server := VM{Name: "api", Lifetime: AlwaysOn, State: Running, SSH: true, CheckStatus: CheckStatus{State: CheckFailing, Since: time.Date(2026, 11, 14, 14, 2, 0, 0, time.Local)}}
	if f, _ := serverFinding(ref, server, ""); f.Detail != "mini. Since 14:02. SSH still answers." {
		t.Errorf("detail %q", f.Detail)
	}
}

package machinefile_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/machinefile"
)

// value: a token an issuer names, built by join so no file holds one.
var planted = "sk-" + strings.Repeat("station-", 4) + "key"

func carried() (machinefile.Line, machinefile.Script) {
	line := machinefile.Line{
		Step: machinefile.Step{Package: "served", Values: map[string]catalog.OptionValue{"endpoint": catalog.OptionValue(planted), "port": "8080"}, From: "https://bot:" + planted + "@example.com/x"},
		Seen: machinefile.Seen{Detail: "read refused: " + planted},
	}
	script := machinefile.Script{Phase: machinefile.RunBoot, Lines: "export KEY=" + planted + "\nmake install"}
	return line, script
}

func leaked(t *testing.T, what string, answer any) {
	t.Helper()
	data, err := json.Marshal(answer)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), planted) {
		t.Errorf("%s carries the token: %s", what, data)
	}
}

func TestAPlanLeavesItsCallScreenedAndCarriesItsOwnValues(t *testing.T) {
	line, script := carried()
	plan := machinefile.Plan{
		Lines:   []machinefile.Line{line},
		Extra:   []machinefile.ExtraLine{{Line: line}},
		Blocks:  []machinefile.Block{{ID: "served", Zsh: "export KEY=" + planted, Diff: "+export KEY=" + planted}},
		Scripts: []machinefile.Script{script},
	}
	answer := plan.Answered()
	leaked(t, "the plan's answer", answer)
	if got := answer.Lines[0].Step.Values; got["endpoint"] != "withheld" || got["port"] != "8080" {
		t.Errorf("the answer's values are %v, want the token withheld whole and the rest kept", got)
	}
	if got := answer.Scripts[0].Lines; got != "export KEY=withheld\nmake install" {
		t.Errorf("the answer's run lines are %q, want the token withheld word by word", got)
	}
	if got := answer.Lines[0].Seen.Detail; got != "read refused: withheld" {
		t.Errorf("the answer's detail is %q", got)
	}
	if plan.Lines[0].Step.Values["endpoint"] != catalog.OptionValue(planted) || plan.Scripts[0].Lines != script.Lines {
		t.Error("answering the plan changed the plan its call carries out")
	}
	if (machinefile.Plan{}).Answered().Lines != nil {
		t.Error("a plan with no line answers an empty list, where it said none")
	}
}

func TestARunAndAnUpgradeLeaveTheirCallScreened(t *testing.T) {
	line, script := carried()
	plan := machinefile.Plan{Lines: []machinefile.Line{line}, Scripts: []machinefile.Script{script}}
	done := machinefile.Done{Line: line, State: catalog.StateBroken, Detail: "exited 1: " + planted}
	ran := &machinefile.Ran{Script: script, State: catalog.StateBroken, Detail: "exited 1: station: exit 1 at: export KEY=" + planted}
	applied := machinefile.Applied{Plan: plan, Done: []machinefile.Done{done}, Removed: []machinefile.Done{done}, Ran: ran,
		Blocks: []machinefile.BlockDone{{Block: machinefile.Block{ID: "served", Zsh: planted}, Detail: "write refused: " + planted}}}
	leaked(t, "the run's answer", applied.Answered())
	if !strings.Contains(ran.Detail, planted) {
		t.Error("answering the run changed the run it reports")
	}
	upgraded := machinefile.Upgraded{Plan: plan, Done: []machinefile.Done{done}, Ran: ran, Unpulled: "fetch refused for https://bot:" + planted + "@example.com/x"}
	answer := upgraded.Answered()
	leaked(t, "the upgrade's answer", answer)
	if answer.Unpulled != "fetch refused for https://withheld@example.com/x" {
		t.Errorf("the upgrade says %q of its source, want the credential withheld", answer.Unpulled)
	}
}

func TestANightKeepsNoTokenInItsLedger(t *testing.T) {
	u := machinefile.Upgraded{Plan: machinefile.Plan{Name: "top"}}.Answered()
	n := machinefile.NightOf(u, errors.New("lock not written: "+planted), "top", time.Unix(0, 0))
	leaked(t, "the night", n)
	if n.Error != "lock not written: withheld" {
		t.Errorf("the night keeps %q, want the token withheld", n.Error)
	}
}

func TestADriftLeavesItsCallScreenedThroughItsPlan(t *testing.T) {
	line, script := carried()
	drift := machinefile.Drift{Plan: machinefile.Plan{Lines: []machinefile.Line{line}, Scripts: []machinefile.Script{script}}}
	leaked(t, "the drift's answer", drift.Answered())
	if drift.Plan.Lines[0].Step.Values["endpoint"] != catalog.OptionValue(planted) {
		t.Error("answering the drift changed the plan its call carries out")
	}
}

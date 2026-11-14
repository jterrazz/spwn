package overview_test

import (
	"strings"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/awake"
	"github.com/jterrazz/studio-station/apps/engine/core/finding"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/core/section"
	"github.com/jterrazz/studio-station/apps/engine/core/verdict"
	"github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/overview"
)

var mac = overview.Head{Name: "macbook", Kind: machine.Computer, ThisMac: true}

func TestTheVerdictSaysTheEngineFirstThenWhatNeedsYou(t *testing.T) {
	down := finding.Finding{Machine: machine.Ref{Name: "infra", On: "macbook"}, Section: section.Overview, Severity: finding.Broken, Subject: "infra", Text: "infra is in a loop"}
	update := finding.Finding{Machine: machine.Ref{Name: "macbook"}, Section: section.Apps, Severity: finding.NeedsALook, Subject: "lazygit", Text: "3 outdated"}
	exposed := finding.Finding{Machine: machine.Ref{Name: "macbook"}, Section: section.Ports, Severity: finding.Broken, Subject: "5432", Text: "port 5432 is reachable"}
	vms := []overview.VM{{Name: "infra", Running: true}}
	for name, c := range map[string]struct {
		engine overview.Engine
		own    []finding.Finding
		vms    []overview.VM
		want   verdict.Verdict
	}{
		"all is well":        {overview.EngineRunning, nil, vms, verdict.Verdict{Tone: verdict.OK, Sentence: overview.SentenceFine}},
		"a Mac with no VM":   {overview.EngineRunning, nil, []overview.VM{}, verdict.Verdict{Tone: verdict.OK, Sentence: overview.SentenceReady}},
		"upkeep alone":       {overview.EngineRunning, []finding.Finding{update}, vms, verdict.Verdict{Tone: verdict.OK, Sentence: overview.SentenceFine, Line: "One thing can wait."}},
		"a server down":      {overview.EngineRunning, []finding.Finding{update, down}, vms, verdict.Verdict{Tone: verdict.Bad, Sentence: "One thing needs you.", Line: "infra is in a loop"}},
		"two needs":          {overview.EngineRunning, []finding.Finding{down, exposed}, vms, verdict.Verdict{Tone: verdict.Bad, Sentence: "2 things need you.", Line: "infra is in a loop"}},
		"the engine stopped": {overview.EngineStopped, []finding.Finding{down}, vms, verdict.Verdict{Tone: verdict.Bad, Sentence: overview.SentenceEngineStopped, Line: overview.LineEngineStopped}},
		"VMs not read yet":   {overview.EngineRunning, nil, nil, verdict.Verdict{Tone: verdict.OK, Sentence: overview.SentenceFine}},
	} {
		if got := overview.Summarize(mac, c.engine, c.own, nil, c.vms).Verdict; got != c.want {
			t.Errorf("%s: %+v; want %+v", name, got, c.want)
		}
	}
}

func TestOnlyWhatIsRedNeedsYouAndEachNeedHasItsVerb(t *testing.T) {
	for name, c := range map[string]struct {
		f      finding.Finding
		upkeep overview.Upkeep
		verb   overview.Verb
	}{
		"a port exposed":     {finding.Finding{Section: section.Ports, Severity: finding.Broken}, overview.NeedsYou, overview.VerbReview},
		"a server down":      {finding.Finding{Section: section.Overview, Severity: finding.Broken, Fix: finding.Fix{Call: "machines.restart"}}, overview.NeedsYou, overview.VerbRestart},
		"the engine stopped": {finding.Finding{Section: section.Overview, Severity: finding.NeedsALook, Urgency: finding.Red, Fix: finding.Fix{Call: "engine.start"}}, overview.NeedsYou, overview.VerbStartEngine},
		"an app that failed": {finding.Finding{Section: section.Apps, Severity: finding.Broken}, overview.CanWait, overview.VerbFix},
		"updates":            {finding.Finding{Section: section.Apps, Severity: finding.NeedsALook, Fix: finding.Fix{Call: "apps.update"}}, overview.CanWait, overview.VerbUpdate},
		"an idle workspace":  {finding.Finding{Section: section.Overview, Severity: finding.NeedsALook, Fix: finding.Fix{Call: "machines.delete"}}, overview.CanWait, overview.VerbDelete},
	} {
		if got := overview.UpkeepOf(c.f); got != c.upkeep {
			t.Errorf("%s: %s; want %s", name, got, c.upkeep)
		}
		if got := overview.VerbOf(c.f); got != c.verb {
			t.Errorf("%s: verb %s; want %s", name, got, c.verb)
		}
	}
	s := overview.Summarize(mac, overview.EngineRunning, []finding.Finding{
		{Section: section.Apps, Severity: finding.Broken, Subject: "jq"},
		{Section: section.Ports, Severity: finding.Broken, Subject: "5432"},
	}, nil, nil)
	if s.Needs[0].Subject != "5432" || s.Needs[0].Upkeep != overview.NeedsYou {
		t.Errorf("an exposed port ranks after a failed app: %+v", s.Needs)
	}
}

func TestAWorkspaceIsIdleAnHourAfterItsAgentFinished(t *testing.T) {
	now := time.Date(2026, 11, 14, 18, 0, 0, 0, time.UTC)
	vm := machines.VM{Name: "linux-1", Lifetime: machines.Persistent, State: machines.Running, Resources: machines.Resources{MemoryBytes: 10 << 30},
		Agent: &machines.AgentView{State: machines.AgentFinished, Ended: now.Add(-5 * time.Hour)}}
	f, ok := overview.Idle("macbook", vm, now)
	if !ok || f.Text != "linux-1 has been idle for 5 h and holds 10 GB" || f.Fix.Call != "machines.delete" || f.Fix.Args["VM"] != "linux-1" {
		t.Fatalf("a workspace finished 5 h ago: %+v, %v", f, ok)
	}
	if overview.UpkeepOf(f) != overview.CanWait {
		t.Errorf("an idle workspace needs you; it can wait")
	}
	vm.Agent.Ended = now.Add(-overview.IdleAfter)
	if _, ok := overview.Idle("macbook", vm, now); !ok {
		t.Errorf("a workspace finished %s ago is not idle", overview.IdleAfter)
	}
	vm.Agent.Ended = now.Add(-overview.IdleAfter + time.Minute)
	if _, ok := overview.Idle("macbook", vm, now); ok {
		t.Errorf("a workspace finished under %s ago is idle", overview.IdleAfter)
	}
	if overview.IdleAfter != time.Hour {
		t.Errorf("a workspace is idle after %s; the rule says an hour", overview.IdleAfter)
	}
	vm.Agent.Ended = now.Add(-5 * time.Hour)
	for name, other := range map[string]machines.VM{
		"a server":          {Name: "infra", Lifetime: machines.AlwaysOn, State: machines.Running, Agent: vm.Agent},
		"a stopped one":     {Name: "linux-1", State: machines.Stopped, Agent: vm.Agent},
		"one still at work": {Name: "linux-1", State: machines.Running, Agent: &machines.AgentView{State: "working"}},
	} {
		if _, ok := overview.Idle("macbook", other, now); ok {
			t.Errorf("%s is idle", name)
		}
	}
}

func TestAnAwakeSessionThatFollowsARunEndsWithIt(t *testing.T) {
	now := time.Date(2026, 11, 14, 18, 0, 0, 0, time.UTC)
	r := awake.Report{Now: now, Engine: awake.EngineRunning, Session: &awake.Session{Until: now.Add(24 * time.Hour), Run: "linux-1"}}
	if got := overview.AwakeRead(r, machine.Ref{Name: "macbook"}).Line.Parts[0].Word; got != "until linux-1 is done" {
		t.Errorf("a session following linux-1's run says %q", got)
	}
	r.Session.Run = ""
	if got := overview.AwakeRead(r, machine.Ref{Name: "macbook"}).Line.Parts[0].Word; !strings.HasPrefix(got, "until 18:00") {
		t.Errorf("a session for a time says %q", got)
	}
}

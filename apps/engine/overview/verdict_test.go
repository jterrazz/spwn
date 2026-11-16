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
	expired := finding.Finding{Machine: machine.Ref{Name: "macbook"}, Section: section.Accounts, Severity: finding.Broken, Subject: "github", Text: "github expired"}
	vms := []overview.VM{{Name: "infra", Running: true}}
	const rest = "1 machine on macbook"
	for name, c := range map[string]struct {
		engine overview.Engine
		own    []finding.Finding
		vms    []overview.VM
		want   verdict.Verdict
	}{
		"all is well":        {overview.EngineRunning, nil, vms, verdict.Verdict{Tone: verdict.OK, Sentence: overview.SentenceFine, Line: rest}},
		"a Mac with no VM":   {overview.EngineRunning, nil, []overview.VM{}, verdict.Verdict{Tone: verdict.OK, Sentence: overview.SentenceReady}},
		"upkeep alone":       {overview.EngineRunning, []finding.Finding{update}, vms, verdict.Verdict{Tone: verdict.OK, Sentence: overview.SentenceFine, Line: rest}},
		"a server down":      {overview.EngineRunning, []finding.Finding{update, down}, vms, verdict.Verdict{Tone: verdict.Bad, Sentence: "infra is down.", Line: "The rest is fine. " + rest}},
		"one need alone":     {overview.EngineRunning, []finding.Finding{exposed}, vms, verdict.Verdict{Tone: verdict.Bad, Sentence: "1 thing needs you.", Line: "The rest is fine. " + rest}},
		"two needs":          {overview.EngineRunning, []finding.Finding{expired, exposed}, vms, verdict.Verdict{Tone: verdict.Bad, Sentence: "2 things need you.", Line: "The rest is fine. " + rest}},
		"a server down more": {overview.EngineRunning, []finding.Finding{down, exposed}, vms, verdict.Verdict{Tone: verdict.Bad, Sentence: "infra is down.", Line: "1 more thing needs you. The rest is fine. " + rest}},
		"the engine stopped": {overview.EngineStopped, []finding.Finding{down}, vms, verdict.Verdict{Tone: verdict.Bad, Sentence: overview.SentenceEngineStopped, Line: overview.LineEngineStopped}},
		"VMs not read yet":   {overview.EngineRunning, nil, nil, verdict.Verdict{Tone: verdict.OK, Sentence: overview.SentenceFine}},
	} {
		if got := overview.Summarize(mac, c.engine, c.own, nil, c.vms).Verdict; got != c.want {
			t.Errorf("%s: %+v; want %+v", name, got, c.want)
		}
	}
}

func TestTheHeadOfAMacSaysItOnlineWithTheMachinesItRuns(t *testing.T) {
	two := []overview.VM{{Name: "a", Running: true}, {Name: "b", Running: true}, {Name: "c"}}
	for name, c := range map[string]struct {
		engine overview.Engine
		vms    []overview.VM
		want   verdict.Verdict
	}{
		"two running":        {overview.EngineRunning, two, verdict.Verdict{Tone: verdict.OK, Sentence: "Online, with 2 machines running."}},
		"one running":        {overview.EngineRunning, two[:1], verdict.Verdict{Tone: verdict.OK, Sentence: "Online, with 1 machine running."}},
		"none running":       {overview.EngineRunning, two[2:], verdict.Verdict{Tone: verdict.OK, Sentence: "Online, with no machine running."}},
		"not read yet":       {overview.EngineRunning, nil, verdict.Verdict{Tone: verdict.OK, Sentence: "Online."}},
		"the engine stopped": {overview.EngineStopped, two, verdict.Verdict{Tone: verdict.Bad, Sentence: overview.SentenceEngineStopped, Line: overview.LineEngineStopped}},
	} {
		got := overview.Summarize(mac, c.engine, nil, nil, c.vms)
		if got.Head.Verdict == nil || *got.Head.Verdict != c.want {
			t.Errorf("%s: %+v; want %+v", name, got.Head.Verdict, c.want)
		}
	}
	up := mac
	up.Up = 3*24*time.Hour + 2*time.Hour
	if got := overview.Summarize(up, overview.EngineRunning, nil, nil, []overview.VM{{Name: "a", Running: true}}).Head.Verdict; got.Line != "Up 3 days." || got.Sentence != "Online, with 1 machine running." {
		t.Errorf("a Mac's head line is how long it is up: %+v", got)
	}
	if stopped := overview.Summarize(up, overview.EngineStopped, nil, nil, nil).Head.Verdict; stopped.Line != overview.LineEngineStopped {
		t.Errorf("a stopped engine keeps its own line: %+v", stopped)
	}
	inside := overview.Summarize(overview.Head{Name: "linux-1", Kind: machine.Workspace}, overview.EngineNone, nil, nil, nil)
	if inside.Head.Verdict != nil {
		t.Errorf("a VM's head is its Mac's to say: %+v", inside.Head.Verdict)
	}
	if stopped := overview.Stopped("api", machine.Workspace); stopped.Head.Verdict != nil {
		t.Errorf("a stopped VM's head is its Mac's to say: %+v", stopped.Head.Verdict)
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
		"a check failing":    {finding.Finding{Section: section.Overview, Severity: finding.Broken, Fix: finding.Fix{Call: "machines.restart", Args: map[string]string{"Name": "api"}}}, overview.NeedsYou, overview.VerbRestart},
		"the engine stopped": {finding.Finding{Section: section.Overview, Severity: finding.NeedsALook, Urgency: finding.Red, Fix: finding.Fix{Call: "engine.start"}}, overview.NeedsYou, overview.VerbStartEngine},
		"an app that failed": {finding.Finding{Section: section.Apps, Severity: finding.Broken}, overview.NeedsYou, overview.VerbFix},
		"a sign-in expired":  {finding.Finding{Section: section.Accounts, Severity: finding.Broken}, overview.NeedsYou, overview.VerbFix},
		"a task failing":     {finding.Finding{Section: section.Background, Severity: finding.Broken, Fix: finding.Fix{Call: "background.remove"}}, overview.NeedsYou, overview.VerbRemove},
		"a setting off":      {finding.Finding{Section: section.Settings, Severity: finding.NeedsALook}, overview.CanWait, overview.VerbFix},
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
		{Section: section.Apps, Severity: finding.NeedsALook, Subject: "jq"},
		{Section: section.Ports, Severity: finding.Broken, Subject: "5432"},
	}, nil, nil)
	if s.Needs[0].Subject != "5432" || s.Needs[0].Upkeep != overview.NeedsYou || s.Needs[1].Upkeep != overview.CanWait {
		t.Errorf("an exposed port ranks after an update: %+v", s.Needs)
	}
}

func TestAWorkspaceIsIdleAnHourAfterItsAgentFinished(t *testing.T) {
	now := time.Date(2026, 11, 14, 18, 0, 0, 0, time.UTC)
	vm := machines.VM{Name: "linux-1", Lifetime: machines.Persistent, State: machines.Running, Resources: machines.Resources{MemoryBytes: 10 << 30},
		Agent: &machines.AgentView{State: machines.AgentFinished, Ended: now.Add(-5 * time.Hour)}}
	f, ok := overview.Idle("macbook", vm, now)
	if !ok || f.Text != "linux-1 has been idle for 5 h and holds 10 GB" || f.Fix.Call != "machines.delete" || f.Fix.Args["VM"] != "linux-1" || f.Fix.On != "macbook" || f.Fix.Confirm != "linux-1" {
		t.Fatalf("a workspace finished 5 h ago: %+v, %v", f, ok)
	}
	if overview.UpkeepOf(f) != overview.CanWait {
		t.Errorf("an idle workspace needs you; it can wait")
	}
	vm.Agent.Ended = now.Add(-machines.IdleAfter)
	if _, ok := overview.Idle("macbook", vm, now); !ok {
		t.Errorf("a workspace finished %s ago is not idle", machines.IdleAfter)
	}
	vm.Agent.Ended = now.Add(-machines.IdleAfter + time.Minute)
	if _, ok := overview.Idle("macbook", vm, now); ok {
		t.Errorf("a workspace finished under %s ago is idle", machines.IdleAfter)
	}
	if machines.IdleAfter != time.Hour {
		t.Errorf("a workspace is idle after %s; the rule says an hour", machines.IdleAfter)
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

func TestASignInNeedNamesTheSignInsTabItOpens(t *testing.T) {
	s := overview.Summarize(mac, overview.EngineRunning, []finding.Finding{
		{Section: section.Accounts, Severity: finding.Broken, Subject: "docs-bot", Text: "docs-bot sign-in expired"},
		{Section: section.Apps, Severity: finding.NeedsALook, Subject: "jq"},
	}, nil, nil)
	nouns := map[string]string{}
	for _, n := range s.Needs {
		nouns[n.Subject] = n.Noun
	}
	if nouns["docs-bot"] != "Sign-ins" || nouns["jq"] != "Apps" {
		t.Errorf("the nouns of a sign-in and an update: %v", nouns)
	}
}

func TestAMacsMachinesAreCountedRunningThenStopped(t *testing.T) {
	for want, vms := range map[string][]overview.VM{
		"3 running, 1 stopped": {{Name: "api", Running: true}, {Name: "linux-1", Running: true}, {Name: "linux-2", Running: true}, {Name: "linux-3"}},
		"1 running":            {{Name: "api", Running: true}},
		"2 stopped":            {{Name: "api"}, {Name: "linux-1"}},
		"":                     {},
	} {
		if got := overview.Summarize(mac, overview.EngineRunning, nil, nil, vms).Machines; got != want {
			t.Errorf("%v: %q; want %q", vms, got, want)
		}
	}
}

func TestANeedWearsExposedForAPortDownForAServerElseItsUpkeepWord(t *testing.T) {
	ref := machine.Ref{Name: "macbook"}
	exposed := finding.Finding{Machine: ref, Section: section.Ports, Severity: finding.Broken, Subject: "macbook:5000", Fix: finding.Fix{Path: "System Settings > General"}, Detail: "macbook. ControlCenter. Not declared public."}
	down := finding.Finding{Machine: machine.Ref{Name: "infra", On: "macbook"}, Section: section.Overview, Severity: finding.Broken, Subject: "infra"}
	macDown := finding.Finding{Machine: ref, Section: section.Overview, Severity: finding.Broken, Subject: "mini"}
	expiring := finding.Finding{Machine: ref, Section: section.Accounts, Urgency: finding.Expiring, Subject: "token"}
	update := finding.Finding{Machine: ref, Section: section.Apps, Severity: finding.NeedsALook, Subject: "lazygit"}
	got := map[string]overview.Need{}
	for _, n := range overview.Summarize(mac, overview.EngineRunning, []finding.Finding{exposed, down, macDown, expiring, update}, nil, []overview.VM{{Name: "infra", Running: true}}).Needs {
		got[n.Subject] = n
	}
	for subject, want := range map[string]overview.Mark{
		"macbook:5000": overview.MarkExposed,
		"infra":        overview.MarkDown,
		"mini":         overview.Mark(overview.NeedsYou),
		"token":        overview.Mark(overview.NeedsYou),
		"lazygit":      overview.Mark(overview.CanWait),
	} {
		if got[subject].Mark != want {
			t.Errorf("%s wears %q; want %q", subject, got[subject].Mark, want)
		}
	}
	if n := got["macbook:5000"]; n.Detail != exposed.Detail || n.Verb != overview.VerbReview || n.Noun != "Ports" || n.Command != "" || n.Path != "" {
		t.Errorf("an exposed port is reviewed on Ports with its detail and no command: %+v", n)
	}
}

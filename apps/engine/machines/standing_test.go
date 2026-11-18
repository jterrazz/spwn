package machines

import (
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/core/finding"
	"github.com/jterrazz/studio-station/apps/engine/core/verdict"
)

func healthyServer() VM {
	return VM{Name: "infra", Lifetime: AlwaysOn, State: Running, Want: "running", IP: "192.168.64.5", SSH: true,
		Check: "infra:443", CheckStatus: CheckStatus{State: CheckPassing}}
}

func TestEachRuleOfAVMsStandingNamesItsValue(t *testing.T) {
	looping := Restarts{Looping: true}
	with := func(change func(*VM)) VM {
		vm := healthyServer()
		change(&vm)
		return vm
	}
	workspace := VM{Name: "linux-1", Lifetime: Persistent, State: Running}
	for name, c := range map[string]struct {
		vm   VM
		want VMStanding
	}{
		"its Mac silent":                     {with(func(v *VM) { v.State = UnknownState }), StandingNotSeen},
		"a restart walking":                  {with(func(v *VM) { v.Restart = &RestartWalk{Step: 2} }), StandingRestarting},
		"a create running":                   {VM{Name: "linux-2", Lifetime: Persistent, State: Creating}, StandingCreating},
		"a workspace in a loop":              {VM{Name: "linux-1", Lifetime: Persistent, State: Broken}, StandingDown},
		"a workspace ended unexpectedly":     {VM{Name: "linux-1", Lifetime: Persistent, State: Stale}, StandingDown},
		"a server stopped in a loop":         {with(func(v *VM) { v.State, v.Want, v.Restarts = Stopped, "stopped", looping }), StandingDown},
		"a server powered off inside":        {with(func(v *VM) { v.State, v.Want, v.Detail = Stopped, "stopped", PoweredOffInside }), StandingDown},
		"a server stopped, wanted running":   {with(func(v *VM) { v.State = Stopped }), StandingDown},
		"a server running in a loop":         {with(func(v *VM) { v.Restarts = looping }), StandingDown},
		"a server running, its check failed": {with(func(v *VM) { v.CheckStatus.State = CheckFailing }), StandingDown},
		"a server stopped on purpose":        {with(func(v *VM) { v.State, v.Want = Stopped, "stopped" }), StandingStopped},
		"a workspace stopped, wanted running": {VM{Name: "linux-1", Lifetime: Persistent, State: Stopped, Want: "running", Restarts: looping},
			StandingStopped},
		"a workspace suspended":              {VM{Name: "linux-1", Lifetime: Persistent, State: Suspended}, StandingSuspended},
		"a server passing every step":        {healthyServer(), StandingHealthy},
		"a server with no check set":         {with(func(v *VM) { v.Check, v.CheckStatus = "", CheckStatus{} }), StandingHealthy},
		"a server whose check waits":         {with(func(v *VM) { v.CheckStatus.State = CheckWaiting }), StandingRunning},
		"a server with no address":           {with(func(v *VM) { v.IP = "" }), StandingRunning},
		"a server whose SSH does not answer": {with(func(v *VM) { v.SSH = false }), StandingRunning},
		"a workspace running":                {workspace, StandingRunning},
		"a workspace whose check fails":      {VM{Name: "linux-1", Lifetime: Persistent, State: Running, CheckStatus: CheckStatus{State: CheckFailing}}, StandingRunning},
	} {
		if got := StandingOf(c.vm); got != c.want {
			t.Errorf("%s: %q, want %q", name, got, c.want)
		}
	}
}

func TestAVMsStandingIsItsFirstRuleThatMatches(t *testing.T) {
	for name, c := range map[string]struct {
		vm   VM
		want VMStanding
	}{
		"a silent Mac's VM mid-create": {VM{Name: "linux-2", State: UnknownState, Birth: &Birth{}}, StandingNotSeen},
		"a silent Mac's VM restarting": {VM{Name: "infra", Lifetime: AlwaysOn, State: UnknownState, Restart: &RestartWalk{Step: 1}}, StandingNotSeen},
		"a looping server that restarts": {VM{Name: "infra", Lifetime: AlwaysOn, State: Running, Restarts: Restarts{Looping: true}, Restart: &RestartWalk{Step: 3}},
			StandingRestarting},
		"a create walking a restart":    {VM{Name: "linux-2", Lifetime: Persistent, State: Creating, Restart: &RestartWalk{Step: 1}}, StandingRestarting},
		"a stale workspace":             {VM{Name: "linux-1", Lifetime: Ephemeral, State: Stale}, StandingDown},
		"a server whose check fails":    {VM{Name: "infra", Lifetime: AlwaysOn, State: Running, IP: "192.168.64.5", SSH: true, Check: "infra:443", CheckStatus: CheckStatus{State: CheckFailing}}, StandingDown},
		"a server stopped while wanted": {VM{Name: "infra", Lifetime: AlwaysOn, State: Stopped, Want: "running"}, StandingDown},
		"a healthy server":              {healthyServer(), StandingHealthy},
		"a server whose check waits":    {VM{Name: "infra", Lifetime: AlwaysOn, State: Running, IP: "192.168.64.5", SSH: true, Check: "infra:443", CheckStatus: CheckStatus{State: CheckWaiting}}, StandingRunning},
	} {
		if got := StandingOf(c.vm); got != c.want {
			t.Errorf("%s: %q, want %q", name, got, c.want)
		}
	}
}

func TestEveryStandingHasItsLowercaseWordAndTone(t *testing.T) {
	want := map[VMStanding]struct {
		word string
		tone verdict.Tone
	}{
		StandingNotSeen: {"not seen", verdict.Plain}, StandingRestarting: {"restarting", verdict.New}, StandingCreating: {"creating", verdict.New},
		StandingDown: {"down", verdict.Bad}, StandingStopped: {"stopped", verdict.Plain}, StandingSuspended: {"suspended", verdict.Plain},
		StandingHealthy: {"healthy", verdict.OK}, StandingRunning: {"running", verdict.OK},
	}
	values := VMStanding("").Enum()
	if len(values) != len(want) {
		t.Fatalf("the standings are %v", values)
	}
	for _, v := range values {
		s := VMStanding(v)
		if s.Word() != want[s].word || s.Tone() != want[s].tone {
			t.Errorf("%s: %q %q, want %q %q", v, s.Word(), s.Tone(), want[s].word, want[s].tone)
		}
	}
}

// invariant: Head, Health's down and Down are functions of the standing
// over every combination of the facts the rule reads.
func TestHeadAndDownAgreeWithTheStandingEverywhere(t *testing.T) {
	now := time.Date(2026, 11, 18, 10, 0, 0, 0, time.UTC)
	states := []VMState{Running, Stopped, Suspended, Stale, Broken, Creating, UnknownState}
	lifetimes := []Lifetime{AlwaysOn, Persistent, Ephemeral}
	checks := []CheckState{"", CheckWaiting, CheckPassing, CheckFailing}
	n := 0
	for _, state := range states {
		for _, lifetime := range lifetimes {
			for _, check := range checks {
				for facts := range 32 {
					vm := VM{Name: "vm", Lifetime: lifetime, State: state, CheckStatus: CheckStatus{State: check}, Check: "vm:443",
						Restarts: Restarts{Looping: facts&1 != 0}, IP: "192.168.64.9", SSH: facts&8 != 0, Want: "stopped"}
					if facts&2 != 0 {
						vm.Restart = &RestartWalk{Step: 1}
					}
					if facts&4 != 0 {
						vm.Want = "running"
					}
					if facts&16 != 0 {
						vm.Detail = PoweredOffInside
					}
					n++
					agree(t, vm, now)
				}
			}
		}
	}
	if n != len(states)*len(lifetimes)*len(checks)*32 {
		t.Fatalf("%d combinations judged", n)
	}
}

func agree(t *testing.T, vm VM, now time.Time) {
	t.Helper()
	s := StandingOf(vm)
	if !slices.Contains(VMStanding("").Enum(), string(s)) {
		t.Fatalf("%+v: no standing %q", vm, s)
	}
	head, _ := HeadOf(vm, now)
	if Down(vm) != (s == StandingDown) {
		t.Errorf("%+v: Down %v, standing %q", vm, Down(vm), s)
	}
	if (head.Sentence == vm.Name+" is down.") != (s == StandingDown) {
		t.Errorf("%+v: head %q, standing %q", vm, head.Sentence, s)
	}
	switch s {
	case StandingHealthy, StandingRunning:
		word := map[VMStanding]string{StandingHealthy: "Healthy", StandingRunning: "Running"}[s]
		if !strings.HasPrefix(head.Sentence, word) || (head.Tone != verdict.OK && head.Tone != verdict.Warn) {
			t.Errorf("%+v: head %+v, standing %q", vm, head, s)
		}
	case StandingNotSeen, StandingRestarting, StandingCreating, StandingDown, StandingStopped, StandingSuspended:
		if head.Tone != s.Tone() {
			t.Errorf("%+v: head tone %q, standing %q's %q", vm, head.Tone, s, s.Tone())
		}
	}
	if health := HealthVerdict(vm); health != nil && vm.State != UnknownState {
		if (health.Sentence == SentenceDown) != (s == StandingDown) {
			t.Errorf("%+v: health %q, standing %q", vm, health.Sentence, s)
		}
		if (health.Sentence == SentenceHealthy) != (s == StandingHealthy) {
			t.Errorf("%+v: health %q, standing %q", vm, health.Sentence, s)
		}
	}
}

func TestMachinesListJudgesEachVMsStanding(t *testing.T) {
	f := Judged(AllMachines{Macs: []MacView{{VMs: []VM{healthyServer(), {Name: "api", Lifetime: Persistent, State: Stopped}}}}})
	if got := []VMStanding{f.Macs[0].VMs[0].Standing, f.Macs[0].VMs[1].Standing}; !slices.Equal(got, []VMStanding{StandingHealthy, StandingStopped}) {
		t.Errorf("standings %v", got)
	}
}

func TestEachRuleOfAMacsStandingNamesItsValue(t *testing.T) {
	seen := time.Date(2026, 11, 18, 9, 0, 0, 0, time.UTC)
	for name, c := range map[string]struct {
		mac  MacView
		want MacStanding
	}{
		"silent after it answered on battery, lid closed": {MacView{Answered: seen, Asleep: true}, MacAsleep},
		"silent otherwise":                          {MacView{Answered: seen, Laptop: true}, MacNotSeen},
		"never answered":                            {MacView{}, MacNotSeen},
		"silent with its engine last stopped":       {MacView{Answered: seen, Engine: EngineView{State: EngineStopped}}, MacNotSeen},
		"asleep with its engine last stopped":       {MacView{Answered: seen, Asleep: true, Engine: EngineView{State: EngineStopped}}, MacAsleep},
		"its engine stopped":                        {MacView{ThisMac: true, Engine: EngineView{State: EngineStopped}}, MacEngineStopped},
		"its engine not installed":                  {MacView{ThisMac: true, Engine: EngineView{State: EngineNotInstalled}}, MacEngineNotInstalled},
		"another Mac whose engine is not installed": {MacView{Reachable: true, Engine: EngineView{State: EngineNotInstalled}}, MacEngineNotInstalled},
		"its engine running":                        {MacView{ThisMac: true, Engine: EngineView{State: EngineRunning}}, MacOnline},
		"answering, its engine not known":           {MacView{Reachable: true}, MacOnline},
		"This Mac that last slept on battery":       {MacView{ThisMac: true, Asleep: true, Engine: EngineView{State: EngineRunning}}, MacOnline},
	} {
		if got := MacStandingOf(c.mac); got != c.want {
			t.Errorf("%s: %q, want %q", name, got, c.want)
		}
	}
}

func TestEveryMacStandingHasItsLowercaseWordAndTone(t *testing.T) {
	want := map[MacStanding]struct {
		word string
		tone verdict.Tone
	}{
		MacAsleep: {"asleep", verdict.Plain}, MacNotSeen: {"not seen", verdict.Warn}, MacEngineStopped: {"engine stopped", verdict.Bad},
		MacEngineNotInstalled: {"engine not installed", verdict.Warn}, MacOnline: {"online", verdict.OK},
	}
	values := MacStanding("").Enum()
	if len(values) != len(want) {
		t.Fatalf("the Mac standings are %v", values)
	}
	for _, v := range values {
		s := MacStanding(v)
		if s.Word() != want[s].word || s.Tone() != want[s].tone {
			t.Errorf("%s: %q %q, want %q %q", v, s.Word(), s.Tone(), want[s].word, want[s].tone)
		}
	}
}

func TestARollUpIsItsFirstRuleThatMatchesAndCountsWhatItNames(t *testing.T) {
	seen := time.Date(2026, 11, 18, 9, 0, 0, 0, time.UTC)
	online := func(vms ...VM) MacView {
		return MacView{Reachable: true, Engine: EngineView{State: EngineRunning}, VMs: vms}
	}
	down := VM{Name: "linux-1", Lifetime: Persistent, State: Stale}
	stopped := VM{Name: "api", Lifetime: Persistent, State: Stopped}
	silent := MacView{Answered: seen, VMs: []VM{{Name: "infra", Lifetime: AlwaysOn, State: UnknownState, Restarts: Restarts{Looping: true}}}}
	asleep := MacView{Answered: seen, Asleep: true, VMs: []VM{{Name: "linux-3", State: UnknownState}}}
	for name, c := range map[string]struct {
		macs []MacView
		want RollUp
	}{
		"no Mac":                                   {nil, RollUp{State: RollUpEmpty}},
		"a Mac with no VM":                         {[]MacView{online()}, RollUp{State: RollUpEmpty}},
		"every VM up or stopped":                   {[]MacView{online(healthyServer(), stopped)}, RollUp{State: RollUpFine}},
		"a VM down":                                {[]MacView{online(down, stopped)}, RollUp{State: RollUpDown, N: 1}},
		"an engine stopped, no VM":                 {[]MacView{{ThisMac: true, Engine: EngineView{State: EngineStopped}}}, RollUp{State: RollUpDown, N: 1}},
		"an engine not installed and a VM down":    {[]MacView{{ThisMac: true, Engine: EngineView{State: EngineNotInstalled}}, online(down)}, RollUp{State: RollUpDown, N: 2}},
		"down before not seen":                     {[]MacView{online(down), silent, {}}, RollUp{State: RollUpDown, N: 1}},
		"two Macs not seen":                        {[]MacView{online(), silent, {}}, RollUp{State: RollUpNotSeen, N: 2}},
		"a silent Mac's VM counts through its Mac": {[]MacView{silent}, RollUp{State: RollUpNotSeen, N: 1}},
		"not seen before empty":                    {[]MacView{online(), {}}, RollUp{State: RollUpNotSeen, N: 1}},
		"an asleep Mac is no fault":                {[]MacView{online(stopped), asleep}, RollUp{State: RollUpFine}},
		"an asleep Mac's VMs are VMs":              {[]MacView{online(), asleep}, RollUp{State: RollUpFine}},
	} {
		if got := RollUpOf(c.macs); got != c.want {
			t.Errorf("%s: %+v, want %+v", name, got, c.want)
		}
	}
}

func TestEveryRollUpStateHasItsTone(t *testing.T) {
	want := map[RollUpState]verdict.Tone{RollUpDown: verdict.Bad, RollUpNotSeen: verdict.Warn, RollUpEmpty: verdict.Plain, RollUpFine: verdict.OK}
	values := RollUpState("").Enum()
	if len(values) != len(want) {
		t.Fatalf("the roll-up states are %v", values)
	}
	for _, v := range values {
		if got := RollUpState(v).Tone(); got != want[RollUpState(v)] {
			t.Errorf("%s: %q, want %q", v, got, want[RollUpState(v)])
		}
	}
}

func TestMachinesListJudgesEachMacsStandingItsRollUpAndAllOfThem(t *testing.T) {
	seen := time.Date(2026, 11, 18, 9, 0, 0, 0, time.UTC)
	f := Judged(AllMachines{Read: seen.Add(time.Hour), Macs: []MacView{
		{Name: "macbook-pro", ThisMac: true, Engine: EngineView{State: EngineRunning}, VMs: []VM{{Name: "linux-1", Lifetime: Persistent, State: Stale}}},
		{Name: "mac-mini", Reachable: true},
		{Name: "macbook-air", Answered: seen},
	}})
	got := make([][]any, 0, len(f.Macs))
	for _, h := range f.Macs {
		got = append(got, []any{h.Name, h.Standing, *h.Machines})
	}
	want := [][]any{
		{"macbook-pro", MacOnline, RollUp{State: RollUpDown, N: 1}},
		{"mac-mini", MacOnline, RollUp{State: RollUpEmpty}},
		{"macbook-air", MacNotSeen, RollUp{State: RollUpNotSeen, N: 1}},
	}
	if !reflect.DeepEqual(got, want) || f.Standing == nil || *f.Standing != (RollUp{State: RollUpDown, N: 1}) {
		t.Errorf("macs %v, all %+v", got, f.Standing)
	}
}

// invariant: a Mac's Unseen verdict, and every finding's severity, Mac or
// VM, are functions of the standing over every combination of the facts the
// rules read: Unseen only for a Mac that does not answer, in its standing's
// tone and never bad; broken only where the standing's tone is bad.
func TestUnseenAndFindingsAgreeWithTheStandingEverywhere(t *testing.T) {
	now := time.Date(2026, 11, 18, 10, 0, 0, 0, time.UTC)
	engines := []EngineState{"", EngineRunning, EngineStopped, EngineNotInstalled}
	n := 0
	for _, engine := range engines {
		for facts := range 64 {
			h := MacView{Name: "mac", ThisMac: facts&1 != 0, Reachable: facts&2 != 0, Asleep: facts&4 != 0, LAN: "192.168.1.9",
				Engine: EngineView{State: engine}}
			if facts&8 != 0 {
				h.Answered = now.Add(-time.Hour)
			}
			if facts&16 != 0 {
				h.VMs = []VM{healthyServer()}
			}
			if facts&32 != 0 {
				h.Laptop = true
			}
			n++
			s := MacStandingOf(h)
			unseen := UnseenVerdict(h, now)
			silent := s == MacAsleep || s == MacNotSeen
			if (unseen != nil) != silent || unseen != nil && (unseen.Tone != s.Tone() || unseen.Tone == verdict.Bad) {
				t.Errorf("%+v: unseen %+v, standing %q", h, unseen, s)
			}
			for _, f := range Findings(AllMachines{Macs: []MacView{h}}) {
				if f.Machine.On == "" && (f.Severity == finding.Broken) != (s.Tone() == verdict.Bad) {
					t.Errorf("%+v: %q is %q, standing %q", h, f.Text, f.Severity, s)
				}
			}
		}
	}
	if n != len(engines)*64 {
		t.Fatalf("%d Macs judged", n)
	}
	states := []VMState{Running, Stopped, Suspended, Stale, Broken, Creating, UnknownState}
	for _, state := range states {
		for _, lifetime := range []Lifetime{AlwaysOn, Persistent, Ephemeral} {
			for _, check := range []CheckState{"", CheckWaiting, CheckPassing, CheckFailing} {
				for facts := range 64 {
					vm := VM{Name: "vm", Lifetime: lifetime, State: state, CheckStatus: CheckStatus{State: check}, Check: "vm:443",
						Restarts: Restarts{Looping: facts&1 != 0}, IP: "192.168.64.9", SSH: facts&8 != 0, Want: "stopped"}
					if facts&2 != 0 {
						vm.Restart = &RestartWalk{Step: 1}
					}
					if facts&4 != 0 {
						vm.Want = "running"
					}
					if facts&16 != 0 {
						vm.Detail = PoweredOffInside
					}
					if facts&32 != 0 {
						vm.Guest = "absent"
					}
					s := StandingOf(vm)
					fs := Findings(AllMachines{Macs: []MacView{{Name: "mac", ThisMac: true, Engine: EngineView{State: EngineRunning}, VMs: []VM{vm}}}})
					broken := slices.ContainsFunc(fs, func(f finding.Finding) bool { return f.Machine.On != "" && f.Severity == finding.Broken })
					if broken != (s == StandingDown) {
						t.Errorf("%+v: findings %+v, standing %q", vm, fs, s)
					}
				}
			}
		}
	}
}

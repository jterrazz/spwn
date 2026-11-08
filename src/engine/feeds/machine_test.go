package feeds_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/src/accounts"
	accountscalls "github.com/jterrazz/studio-station/src/accounts/calls"
	"github.com/jterrazz/studio-station/src/activity"
	activitycalls "github.com/jterrazz/studio-station/src/activity/calls"
	"github.com/jterrazz/studio-station/src/awake"
	awakecalls "github.com/jterrazz/studio-station/src/awake/calls"
	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/core/finding"
	"github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/engine"
	"github.com/jterrazz/studio-station/src/engine/feeds"
	"github.com/jterrazz/studio-station/src/machines"
	machinescalls "github.com/jterrazz/studio-station/src/machines/calls"
	"github.com/jterrazz/studio-station/src/overview"
	overviewcalls "github.com/jterrazz/studio-station/src/overview/calls"
	"github.com/jterrazz/studio-station/src/page"
	"github.com/jterrazz/studio-station/src/ports"
	portscalls "github.com/jterrazz/studio-station/src/ports/calls"
)

const (
	thisMac = "jterrazz-macbook-pro"
	mini    = "jterrazz-mac-mini"
	gb      = int64(1) << 30
)

type steps struct {
	mu    sync.Mutex
	wakes map[time.Duration]chan struct{}
}

func newSteps() *steps {
	return &steps{wakes: map[time.Duration]chan struct{}{2 * time.Second: make(chan struct{}), 30 * time.Second: make(chan struct{})}}
}

func (s *steps) Sleep(ctx context.Context, d time.Duration) error {
	s.mu.Lock()
	wake := s.wakes[d]
	s.mu.Unlock()
	select {
	case <-wake:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// value: something of the page waits d, as the clock sees it within 200 ms.
func (s *steps) asleep(d time.Duration) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	select {
	case s.wakes[d] <- struct{}{}:
		return true
	case <-ctx.Done():
		return false
	}
}

func (s *steps) Now() time.Time { return time.Date(2026, 11, 3, 10, 30, 0, 0, time.UTC) }

func (s *steps) pass(t *testing.T, d time.Duration) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	select {
	case s.wakes[d] <- struct{}{}:
	case <-ctx.Done():
		t.Fatalf("nothing of the page waits %s", d)
	}
}

type world struct {
	mu     sync.Mutex
	infra  []machines.VM
	beside []machines.VM
	here   []machines.VM
	added  []machines.MacView
	lent   accounts.Report
	// value: each Mac's own accounts reading, by name; nil answers lent.
	signedIn map[string]accounts.Report
	updates  bool
	asked    map[string][]string
	activity map[string]uint64
	// value: the activity reads asked, by the machine they ran on.
	reads map[string][]activitycalls.ReadIn
	// value: each Mac's Awake, by name; a Mac absent does not answer.
	awake map[string]awake.Report
}

func (w *world) machines(here bool) machines.AllMachines {
	w.mu.Lock()
	defer w.mu.Unlock()
	infra := w.infra[0]
	if len(w.infra) > 1 {
		w.infra = w.infra[1:]
	}
	if here {
		return machines.AllMachines{Macs: []machines.MacView{{Name: thisMac, ThisMac: true, Reachable: true, Engine: machines.EngineView{Known: true, Running: true}, VMs: w.here}}}
	}
	return machines.AllMachines{Macs: append([]machines.MacView{
		{Name: thisMac, ThisMac: true, Reachable: true, Engine: machines.EngineView{Known: true, Running: true}, VMs: w.here},
		{Name: mini, Reachable: true, Engine: machines.EngineView{Known: true, Running: true}, VMs: append([]machines.VM{infra}, w.beside...)},
	}, w.added...)}
}

func (w *world) answer(name string, target machine.Ref, in any) any {
	w.mu.Lock()
	at := target.Name
	if at == "" {
		at = thisMac
	}
	w.asked[name] = append(w.asked[name], at)
	w.mu.Unlock()
	switch name {
	case machinescalls.List.Name:
		listed, _ := in.(machinescalls.ListIn)
		return machinescalls.ListOut{All: w.machines(listed.ThisMac)}
	case portscalls.List.Name:
		return ports.Listing{}
	case accountscalls.List.Name:
		w.mu.Lock()
		defer w.mu.Unlock()
		if r, ok := w.signedIn[at]; ok {
			return r
		}
		return w.lent
	case overviewcalls.Read.Name:
		need := map[string]string{thisMac: "a sign-in expired", mini: "port 5000 is reachable from outside", "infra": "a container exited"}[at]
		needs := []overview.Need{{Machine: at, Section: "ports", Broken: true, Text: need}}
		w.mu.Lock()
		if at == thisMac && w.updates {
			needs = append(needs, overview.Need{Machine: at, Section: "apps", Subject: "updates", Text: "2 app updates", Call: "apps.update"})
		}
		w.mu.Unlock()
		head := overview.Head{Name: at}
		if at == mini {
			head.Up = 21 * 24 * time.Hour
		}
		return overviewcalls.ReadOut{Summary: overview.Summary{Head: head, Needs: needs}}
	case awakecalls.Read.Name:
		w.mu.Lock()
		defer w.mu.Unlock()
		if r, ok := w.awake[at]; ok {
			return r
		}
		return nil
	case activitycalls.Read.Name:
		w.mu.Lock()
		defer w.mu.Unlock()
		if w.reads == nil {
			w.reads = map[string][]activitycalls.ReadIn{}
		}
		read, _ := in.(activitycalls.ReadIn)
		w.reads[at] = append(w.reads[at], read)
		return activity.Report{Source: activity.FromEngine, Findings: []finding.Finding{}}
	case activitycalls.Sample.Name:
		memory := map[string]int64{thisMac: 16 * gb, mini: 64 * gb, "infra": 24 * gb}[at]
		w.mu.Lock()
		w.activity[at]++
		n := w.activity[at]
		w.mu.Unlock()
		return activity.Reading{Sample: activity.Sample{CPUBusy: n * 300, CPUTotal: n * 1000, MemUsed: memory / 2, MemTotal: memory}}
	}
	return nil
}

func (w *world) Invoke(_ context.Context, target machine.Ref, name string, in, out any, _ call.Progress) error {
	v := w.answer(name, target, in)
	if v == nil {
		return errors.New(name + " is not answered in this world")
	}
	reflect.ValueOf(out).Elem().Set(reflect.ValueOf(v))
	return nil
}

func (w *world) Watch(_ context.Context, target machine.Ref, name string, in any, _ call.Progress) (<-chan call.Update[any], error) {
	once := make(chan call.Update[any], 1)
	if v := w.answer(name, target, in); v != nil {
		once <- call.Update[any]{Value: v}
	}
	close(once)
	return once, nil
}

func (w *world) askedOf(name string) []string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return slices.Clone(w.asked[name])
}

func infraAt(state machines.VMState, restarts int) machines.VM {
	return machines.VM{Name: "infra", Mac: mini, Lifetime: machines.AlwaysOn, State: state, Want: "running", Guest: "connected", IP: "192.168.64.7", Check: "https://infra:6443/readyz", Device: "infra",
		Restarts: machines.Restarts{Count: restarts}, CheckStatus: machines.CheckStatus{State: machines.CheckPassing}}
}

func watch(t *testing.T, w *world, clock *steps, subject string) <-chan any {
	t.Helper()
	return watchWith(t, w, clock, engine.FeedInput{Subject: subject})
}

func watchWith(t *testing.T, w *world, clock *steps, in engine.FeedInput, live ...*feeds.Live) <-chan any {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	o := feeds.Menubar{Calls: w, Clock: clock}
	if len(live) > 0 {
		o.Live = live[0]
	}
	return o.Machine()(ctx, in)
}

func until(t *testing.T, pages <-chan any, what string, ok func(page.Page) bool) page.Page {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	for {
		select {
		case v := <-pages:
			d, isPage := v.(page.Document)
			if !isPage {
				t.Fatalf("the feed sent %T, want a page", v)
			}
			if ok(d.Page) {
				return d.Page
			}
		case <-ctx.Done():
			t.Fatalf("the page never showed %s", what)
		}
	}
}

func find(items []page.Item, id string) *page.Row {
	for _, it := range items {
		if it.Row != nil && it.Row.ID == id {
			return it.Row
		}
		if it.Group != nil {
			if it.Group.Head.ID == id {
				return &it.Group.Head
			}
			if r := find(it.Group.Items, id); r != nil {
				return r
			}
		}
	}
	return nil
}

func row(p page.Page, section, id string) *page.Row {
	for _, s := range p.Sections {
		if s.ID == section {
			return find(s.Items, id)
		}
	}
	return nil
}

func needs(p page.Page) []string {
	var out []string
	for _, s := range p.Sections {
		if s.ID != "needs-you" {
			continue
		}
		for _, it := range s.Items {
			if it.Row != nil {
				out = append(out, it.Row.Label)
			}
		}
	}
	return out
}

func TestTheMacMinisPageOpenFromThisMacIsTheMinisOwnAnswer(t *testing.T) {
	w := &world{infra: []machines.VM{infraAt(machines.Running, 0)}, asked: map[string][]string{}, activity: map[string]uint64{}}
	facts := func(p page.Page) []string {
		var out []string
		for _, s := range p.Sections {
			if s.ID == "summary" {
				for _, v := range s.Caption {
					out = append(out, v.Text)
				}
			}
		}
		return out
	}
	p := until(t, watch(t, w, newSteps(), mini), "mac-mini's time up", func(p page.Page) bool { return len(facts(p)) > 0 })
	if got := facts(p); !slices.Equal(got, []string{"up 21 days"}) {
		t.Errorf("mac-mini's page says %q, want the mini's own time up", got)
	}
	if got := w.askedOf(overviewcalls.Read.Name); !slices.Equal(got, []string{mini}) {
		t.Errorf("the overview was read on %q, want on mac-mini only", got)
	}
}

func TestInfrasPageFollowsItLiveWhileItRestarts(t *testing.T) {
	w := &world{infra: []machines.VM{infraAt(machines.Running, 0), infraAt(machines.Stopped, 0), infraAt(machines.Running, 1)}, asked: map[string][]string{}, activity: map[string]uint64{}}
	clock := newSteps()
	live := &feeds.Live{Calls: w, Clock: clock, ThisMac: thisMac}
	pages := watchWith(t, w, clock, engine.FeedInput{Subject: "infra"}, live)
	state := func(p page.Page) page.Mark {
		if r := row(p, "machine", "state"); r != nil {
			return r.Mark
		}
		return ""
	}
	p := until(t, pages, "infra running", func(p page.Page) bool { return state(p) == page.OK })
	if !slices.Equal(p.Live, []string{"cpu:infra", "memory:infra", "disk:infra"}) {
		t.Errorf("infra's page reads the live keys %q, want its CPU, memory and disk", p.Live)
	}
	if got := needs(p); len(got) != 0 {
		t.Errorf("infra's page draws Needs you %q", got)
	}
	first := live.Sample(t.Context(), p.Live)
	memory, _ := first.Values["memory:infra"].(page.Value)
	if memory.Kind != page.KindSpark || memory.Items[0].Kind != page.KindRatio || memory.Items[0].Of != 24*gb {
		t.Errorf("the first tick reads infra's memory as %+v, want infra's own 24 GB", memory)
	}
	if _, ok := first.Values["cpu:infra"]; ok {
		t.Errorf("the first tick reads infra's CPU from one reading, want it once a second one came")
	}
	second := live.Sample(t.Context(), p.Live)
	if cpu, _ := second.Values["cpu:infra"].(page.Value); cpu.Kind != page.KindSpark || cpu.Items[0].Kind != page.KindPercent {
		t.Errorf("the second tick reads infra's CPU as %+v, want a percent led into its series", cpu)
	}
	if clock.asleep(2 * time.Second) {
		t.Errorf("infra's page waits 2 s to be rebuilt, want its activity read live and never rebuilt on a tick")
	}
	if got := w.askedOf(machinescalls.List.Name); len(got) != 1 {
		t.Errorf("two ticks re-read the machines %d times, want them read once, at the page's opening", len(got))
	}
	clock.pass(t, 30*time.Second)
	p = until(t, pages, "infra down while it restarts", func(p page.Page) bool { return state(p) == page.Broken })
	if r := row(p, "activity", "memory"); r == nil || r.Cells[0].Kind != page.KindNone || r.Cells[0].Live != "" {
		t.Errorf("infra stopped draws its memory as %+v, want none, read live no more", r)
	}
	clock.pass(t, 30*time.Second)
	p = until(t, pages, "infra back with the memory the ticks kept", func(p page.Page) bool {
		r := row(p, "activity", "memory")
		return state(p) == page.OK && r != nil && r.Cells[0].Kind == page.KindRatio
	})
	if r := row(p, "up", "restarts"); r == nil || r.Cells[0].Items[0].N != 1 {
		t.Errorf("infra back counts %+v restarts, want 1", r)
	}
	for _, name := range []string{activitycalls.Sample.Name, overviewcalls.Read.Name} {
		for _, at := range w.askedOf(name) {
			if at != "infra" {
				t.Errorf("%s was read on %s, want every read on infra, through its Mac", name, at)
			}
		}
	}
}

func TestTheSwitcherDrawsEachMacsServersThenWorkspacesAndRereadsEveryThirtySeconds(t *testing.T) {
	build := machines.VM{Name: "build", Mac: mini, Lifetime: machines.Throwaway, State: machines.Running}
	w := &world{infra: []machines.VM{infraAt(machines.Running, 0), infraAt(machines.Stopped, 0)}, beside: []machines.VM{build}, asked: map[string][]string{}, activity: map[string]uint64{}}
	clock := newSteps()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	pages := feeds.Menubar{Calls: w, Clock: clock}.Switcher()(ctx, engine.FeedInput{})
	p := until(t, pages, "every Mac", func(p page.Page) bool { return row(p, "machines", "infra") != nil })
	var groups []string
	for _, it := range p.Sections[0].Items {
		if it.Group != nil && it.Group.Head.ID == mini {
			for _, g := range it.Group.Items {
				groups = append(groups, g.Group.Head.Label)
			}
		}
	}
	if !slices.Equal(groups, []string{"Servers", "Workspaces"}) {
		t.Errorf("mac-mini's groups are %q, want its Servers then its Workspaces", groups)
	}
	if r := row(p, "machines", "infra"); r.Mark != page.OK {
		t.Errorf("infra is drawn %q before it stopped", r.Mark)
	}
	clock.pass(t, 30*time.Second)
	until(t, pages, "infra down thirty seconds later", func(p page.Page) bool { return row(p, "machines", "infra").Mark == page.Broken })
}

func lentTo(vm string, providers ...string) accounts.Report {
	until := time.Date(2026, 11, 4, 6, 30, 0, 0, time.UTC)
	m := accounts.Machine{Name: vm, Kind: accounts.KindVM}
	for _, p := range providers {
		m.Accounts = append(m.Accounts, accounts.Account{Provider: p, Shared: &accounts.Loan{Name: p, Until: until}})
	}
	return accounts.Report{Machines: []accounts.Machine{m}}
}

func shared(p page.Page) []string {
	var out []string
	for _, s := range p.Sections {
		if s.ID != "shared" {
			continue
		}
		for _, it := range s.Items {
			if it.Row != nil {
				out = append(out, it.Row.ID)
			}
		}
	}
	return out
}

func TestAnAccountTakenBackLeavesLinux1sPageOnceTheCallReturns(t *testing.T) {
	linux := machines.VM{Name: "linux-1", Mac: thisMac, Lifetime: machines.Permanent, State: machines.Running, Guest: "connected"}
	w := &world{infra: []machines.VM{infraAt(machines.Running, 0)}, here: []machines.VM{linux}, lent: lentTo("linux-1", catalog.ProviderGitHub, catalog.ProviderClaude), asked: map[string][]string{}, activity: map[string]uint64{}}
	refresh := make(chan struct{}, 1)
	pages := watchWith(t, w, newSteps(), engine.FeedInput{Subject: "linux-1", Refresh: refresh})
	until(t, pages, "GitHub and Claude shared with linux-1", func(p page.Page) bool {
		return slices.Equal(shared(p), []string{catalog.ProviderGitHub, catalog.ProviderClaude})
	})
	w.mu.Lock()
	w.lent = lentTo("linux-1", catalog.ProviderClaude)
	w.mu.Unlock()
	refresh <- struct{}{}
	p := until(t, pages, "GitHub gone from linux-1's shared accounts", func(p page.Page) bool { return slices.Equal(shared(p), []string{catalog.ProviderClaude}) })
	for _, a := range p.Actions {
		if a.ID == "take-back/github" {
			t.Errorf("linux-1's page still offers %q after GitHub was taken back", a.Label)
		}
	}
}

func TestAMacAddedReachesTheSwitcherOnceTheCallReturns(t *testing.T) {
	w := &world{infra: []machines.VM{infraAt(machines.Running, 0)}, asked: map[string][]string{}, activity: map[string]uint64{}}
	refresh := make(chan struct{}, 1)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	pages := feeds.Menubar{Calls: w, Clock: newSteps()}.Switcher()(ctx, engine.FeedInput{Refresh: refresh})
	until(t, pages, "every Mac", func(p page.Page) bool { return row(p, "machines", mini) != nil })
	w.mu.Lock()
	w.added = []machines.MacView{{Name: "studio-1", Reachable: true, Engine: machines.EngineView{Known: true, Running: true}}}
	w.mu.Unlock()
	refresh <- struct{}{}
	until(t, pages, "studio-1 in the switcher", func(p page.Page) bool { return row(p, "machines", "studio-1") != nil })
}

func TestAWorkspaceStartedOnThisMacReachesItsPageWhenTheEngineMoves(t *testing.T) {
	linux := machines.VM{Name: "linux-1", Mac: thisMac, Lifetime: machines.Permanent, State: machines.Stopped}
	w := &world{infra: []machines.VM{infraAt(machines.Running, 0)}, here: []machines.VM{linux}, asked: map[string][]string{}, activity: map[string]uint64{}}
	changed := make(chan struct{}, 1)
	pages := watchWith(t, w, newSteps(), engine.FeedInput{Subject: "linux-1", Changed: changed})
	until(t, pages, "linux-1 stopped", func(p page.Page) bool { r := row(p, "machine", "state"); return r != nil && r.Mark != page.OK })
	w.mu.Lock()
	linux.State = machines.Running
	linux.Guest = "connected"
	w.here = []machines.VM{linux}
	w.mu.Unlock()
	changed <- struct{}{}
	until(t, pages, "linux-1 running", func(p page.Page) bool { r := row(p, "machine", "state"); return r != nil && r.Mark == page.OK })
}

func TestAWorkspaceCreatedFromTheMenuBarAppearsInHomeWhenTheEngineMoves(t *testing.T) {
	w := &world{infra: []machines.VM{infraAt(machines.Running, 0)}, asked: map[string][]string{}, activity: map[string]uint64{}}
	changed := make(chan struct{}, 1)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	o := feeds.Menubar{Calls: w, Clock: newSteps(), Report: func(awake.View) *awake.Report { return nil }}
	in := engine.FeedInput{Changed: changed}
	in.Awake = nothing(in.Awake)
	pages := o.Feed()(ctx, in)
	until(t, pages, "every Mac", func(p page.Page) bool { return row(p, "machines", mini) != nil })
	w.mu.Lock()
	w.here = []machines.VM{{Name: "linux-2", Mac: thisMac, Lifetime: machines.Permanent, State: machines.Running, Guest: "connected"}}
	w.mu.Unlock()
	changed <- struct{}{}
	p := until(t, pages, "linux-2 in Home", func(p page.Page) bool { return row(p, "machines", "linux-2") != nil })
	if row(p, "machines", "infra") == nil {
		t.Error("Home lost mac-mini's infra when This Mac was read again")
	}
}

func TestAWorkspaceCreatedFromTheMenuBarAppearsInHomeOnceTheCallReturns(t *testing.T) {
	for _, on := range []string{thisMac, mini} {
		t.Run(on, func(t *testing.T) {
			w := &world{infra: []machines.VM{infraAt(machines.Running, 0)}, asked: map[string][]string{}, activity: map[string]uint64{}}
			refresh := make(chan struct{}, 1)
			ctx, cancel := context.WithCancel(context.Background())
			t.Cleanup(cancel)
			o := feeds.Menubar{Calls: w, Clock: newSteps(), Report: func(awake.View) *awake.Report { return nil }}
			in := engine.FeedInput{Refresh: refresh}
			in.Awake = nothing(in.Awake)
			pages := o.Feed()(ctx, in)
			until(t, pages, "every Mac", func(p page.Page) bool { return row(p, "machines", mini) != nil })
			created := machines.VM{Name: "linux-2", Mac: on, Lifetime: machines.Permanent, State: machines.Running, Guest: "connected"}
			w.mu.Lock()
			if on == thisMac {
				w.here = []machines.VM{created}
			} else {
				w.beside = []machines.VM{created}
			}
			w.mu.Unlock()
			refresh <- struct{}{}
			until(t, pages, "linux-2 in Home", func(p page.Page) bool { return row(p, "machines", "linux-2") != nil })
		})
	}
}

func TestUpdateAllLeavesHomesNeedsYouOnceTheCallReturns(t *testing.T) {
	w := &world{infra: []machines.VM{infraAt(machines.Running, 0)}, updates: true, asked: map[string][]string{}, activity: map[string]uint64{}}
	refresh := make(chan struct{}, 1)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	o := feeds.Menubar{Calls: w, Clock: newSteps(), Report: func(awake.View) *awake.Report { return nil }}
	in := engine.FeedInput{Refresh: refresh}
	in.Awake = nothing(in.Awake)
	pages := o.Feed()(ctx, in)
	until(t, pages, "2 app updates in Needs you", func(p page.Page) bool { return slices.Contains(needs(p), "2 app updates") })
	w.mu.Lock()
	w.updates = false
	w.mu.Unlock()
	refresh <- struct{}{}
	until(t, pages, "2 app updates gone from Needs you", func(p page.Page) bool { return len(needs(p)) > 0 && !slices.Contains(needs(p), "2 app updates") })
}

func nothing[T any](func() T) func() T { return func() T { var zero T; return zero } }

// invariant: SPEC A4, B5.7: a Mac's page reads that Mac's Awake through
// its engine and the copies every Mac shared, so its care card says it
// never sleeps and its Needs you holds the copy it shared that ends soon,
// not another Mac's; its times
// are in the owner's own clock.
func TestTheMacMinisPageSaysItsAwakeTheCopiesEndingAndItsTimesInTheOwnersClock(t *testing.T) {
	zone := time.Local
	time.Local = time.FixedZone("CET", 3600)
	t.Cleanup(func() { time.Local = zone })
	clock := newSteps()
	soon := accounts.Report{Machines: []accounts.Machine{{Name: mini}},
		Shares: []accounts.Shared{{Account: catalog.ProviderGitHub, VM: "review-bot", At: clock.Now().Add(-time.Hour), Until: clock.Now().Add(20 * time.Minute)}}}
	here := accounts.Report{Machines: []accounts.Machine{{Name: thisMac}},
		Shares: []accounts.Shared{{Account: catalog.ProviderClaude, VM: "linux-1", At: clock.Now().Add(-time.Hour), Until: clock.Now().Add(10 * time.Minute)}}}
	down := infraAt(machines.Stopped, 0)
	down.CheckStatus.Since = clock.Now().Add(-25 * time.Minute)
	w := &world{infra: []machines.VM{down}, asked: map[string][]string{}, activity: map[string]uint64{},
		signedIn: map[string]accounts.Report{mini: soon, thisMac: here},
		awake:    map[string]awake.Report{mini: {Engine: awake.EngineRunning, NeverSleeps: true, NeverHeld: true, Servers: []string{"infra"}}}}
	ending := "ending:" + mini + "/review-bot/" + catalog.ProviderGitHub
	p := until(t, watch(t, w, clock, mini), "mac-mini's Awake and the copy ending", func(p page.Page) bool {
		return row(p, "care", "awake") != nil && row(p, "needs-you", ending) != nil
	})
	if r := row(p, "care", "awake"); r.Label != "Never sleeps" || r.Detail.Text != "hosts infra, which pauses if it sleeps" {
		t.Errorf("mac-mini's Awake reads %q, %q", r.Label, r.Detail.Text)
	}
	if got := w.askedOf(awakecalls.Read.Name); !slices.Contains(got, mini) || slices.Contains(got, thisMac) {
		t.Errorf("Awake was read on %q, want on mac-mini", got)
	}
	if r := row(p, "needs-you", ending); r.Detail.Text != "GitHub ends in 20 min" {
		t.Errorf("the copy ending reads %q", r.Detail.Text)
	}
	if r := row(p, "needs-you", "ending:"+thisMac+"/linux-1/"+catalog.ProviderClaude); r != nil {
		t.Errorf("mac-mini's Needs you holds This Mac's copy %q", r.Detail.Text)
	}
	if r := row(p, "vms", "infra"); r == nil || len(r.Detail.Items) == 0 || r.Detail.Items[0].Text != "down since 11:05" {
		t.Errorf("infra reads %+v, want down since 11:05 in the owner's clock", r)
	}
}

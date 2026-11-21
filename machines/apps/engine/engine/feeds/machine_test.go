package feeds_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	accountscalls "github.com/jterrazz/studio-station/apps/engine/accounts/calls"
	"github.com/jterrazz/studio-station/apps/engine/activity"
	activitycalls "github.com/jterrazz/studio-station/apps/engine/activity/calls"
	"github.com/jterrazz/studio-station/apps/engine/awake"
	awakecalls "github.com/jterrazz/studio-station/apps/engine/awake/calls"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/finding"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/engine"
	"github.com/jterrazz/studio-station/apps/engine/engine/feeds"
	grantscalls "github.com/jterrazz/studio-station/apps/engine/grants/calls"
	"github.com/jterrazz/studio-station/apps/engine/machines"
	machinescalls "github.com/jterrazz/studio-station/apps/engine/machines/calls"
	"github.com/jterrazz/studio-station/apps/engine/overview"
	overviewcalls "github.com/jterrazz/studio-station/apps/engine/overview/calls"
	"github.com/jterrazz/studio-station/apps/engine/page"
	"github.com/jterrazz/studio-station/apps/engine/ports"
	portscalls "github.com/jterrazz/studio-station/apps/engine/ports/calls"
	vaultcalls "github.com/jterrazz/studio-station/apps/engine/vault/calls"
)

const (
	thisMac = "macbook-pro"
	mini    = "mac-mini"
	gb      = int64(1) << 30
)

type steps struct {
	mu    sync.Mutex
	wakes map[time.Duration]chan struct{}
	zone  *time.Location
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
	// value: the programs each machine reads as signed in, by the Mac asked
	// and the VM read through it ("mac/vm", empty vm the Mac's own).
	programs map[string][]accounts.ProgramRow
	// value: This Mac's vault, and what grants.new says of each entry.
	vault    accounts.VaultRead
	offers   map[string]accounts.GrantOffer
	read     []string
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
		return machines.AllMachines{Macs: []machines.MacView{{Name: thisMac, ThisMac: true, Reachable: true, Engine: machines.EngineView{State: machines.EngineRunning}, VMs: w.here}}}
	}
	return machines.AllMachines{Macs: append([]machines.MacView{
		{Name: thisMac, ThisMac: true, Reachable: true, Engine: machines.EngineView{State: machines.EngineRunning}, VMs: w.here},
		{Name: mini, Reachable: true, Engine: machines.EngineView{State: machines.EngineRunning}, VMs: append([]machines.VM{infra}, w.beside...)},
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
		return machinescalls.ListOut{All: machines.Judged(w.machines(listed.ThisMac))}
	case portscalls.List.Name:
		return ports.Listing{}
	case accountscalls.Programs.Name:
		w.mu.Lock()
		defer w.mu.Unlock()
		of, _ := in.(accountscalls.ProgramsIn)
		w.read = append(w.read, at+"/"+of.Machine)
		return accounts.ProgramsRead{Programs: w.programs[at+"/"+of.Machine]}
	case vaultcalls.VaultList.Name:
		w.mu.Lock()
		defer w.mu.Unlock()
		return w.vault
	case grantscalls.GrantNew.Name:
		w.mu.Lock()
		defer w.mu.Unlock()
		entry, _ := in.(grantscalls.GrantNewIn)
		return w.offers[entry.Entry]
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
	o := feeds.Menubar{Calls: w, Clock: clock, Zone: clock.zone}
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
	p := until(t, watch(t, w, newSteps(), mini), "mac-mini's time up", func(p page.Page) bool { return slices.Contains(facts(p), "up 21 days") })
	if got := facts(p); !slices.Equal(got, []string{"online", "up 21 days"}) {
		t.Errorf("mac-mini's page says %q, want its standing, then the mini's own time up", got)
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
		for _, s := range p.Sections {
			if s.ID == "health" && len(s.Items) > 0 && s.Items[0].Group != nil {
				return s.Items[0].Group.Head.Mark
			}
		}
		return ""
	}
	p := until(t, pages, "infra running, its activity read", func(p page.Page) bool {
		w.mu.Lock()
		defer w.mu.Unlock()
		return state(p) == page.OK && len(w.reads["infra"]) > 0
	})
	if !slices.Equal(p.Live, []string{"cpu:infra", "free:infra", "net:infra"}) {
		t.Errorf("infra's page reads the live keys %q, want its CPU, free memory and network", p.Live)
	}
	w.mu.Lock()
	kept := w.reads["infra"]
	w.mu.Unlock()
	if len(kept) == 0 || !kept[0].History || kept[0].Graph != 6*time.Minute || !kept[0].Quick {
		t.Errorf("infra's activity was read %+v, want its day and its last 6 min as its engine keeps them, without waiting", kept)
	}
	if got := needs(p); len(got) != 0 {
		t.Errorf("infra's page draws Needs you %q", got)
	}
	first := live.Sample(t.Context(), p.Live)
	memory, _ := first.Values["free:infra"].(page.Value)
	if memory.Kind != page.KindText || memory.Text != "12 GB free" {
		t.Errorf("the first tick reads infra's free memory as %+v, want infra's own 12 GB free, in the gauge's words", memory)
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
	if g := gaugeOf(p, "free:infra"); g != nil || slices.Contains(p.Live, "free:infra") {
		t.Errorf("infra stopped draws its memory as %+v, live %q, want it read live no more", g, p.Live)
	}
	clock.pass(t, 30*time.Second)
	p = until(t, pages, "infra back with its memory read live again", func(p page.Page) bool {
		return state(p) == page.OK && gaugeOf(p, "free:infra") != nil
	})
	var facts []string
	for _, s := range p.Sections {
		if s.ID == "summary" {
			for _, v := range s.Caption {
				facts = append(facts, v.Text)
			}
		}
	}
	if !slices.Contains(facts, "restarted 1 time") {
		t.Errorf("infra back says %q, want it restarted 1 time", facts)
	}
	for _, name := range []string{activitycalls.Sample.Name, overviewcalls.Read.Name} {
		for _, at := range w.askedOf(name) {
			if at != "infra" {
				t.Errorf("%s was read on %s, want every read on infra, through its Mac", name, at)
			}
		}
	}
}

func programNamed(provider string) accounts.Program {
	for _, p := range catalog.Providers {
		if p.ID != provider {
			continue
		}
		for _, known := range catalog.LoginPrograms {
			if known.Label == p.Label {
				return accounts.Program(known.Name)
			}
		}
	}
	return ""
}

func grantedTo(providers ...string) []accounts.ProgramRow {
	rows := make([]accounts.ProgramRow, 0, len(providers))
	for _, p := range providers {
		program := programNamed(p)
		rows = append(rows, accounts.Describe(accounts.Facts{Program: program, State: accounts.ProgramSignedIn, How: accounts.HowGranted, Entry: "pw-" + p, EntryLabel: p + " token"}, accounts.Where{}))
	}
	return rows
}

func granted(p page.Page) []string {
	var out []string
	for _, s := range p.Sections {
		if s.ID != "shared" {
			continue
		}
		for _, it := range s.Items {
			if it.Row != nil && it.Row.ID != "grant-password" {
				out = append(out, it.Row.ID[strings.LastIndex(it.Row.ID, "/")+1:])
			}
		}
	}
	return out
}

func TestAGrantRemovedLeavesLinux1sPageOnceItIsRead(t *testing.T) {
	linux := machines.VM{Name: "linux-1", Mac: thisMac, Lifetime: machines.Persistent, State: machines.Running, Guest: "connected"}
	w := &world{infra: []machines.VM{infraAt(machines.Running, 0)}, here: []machines.VM{linux}, asked: map[string][]string{}, activity: map[string]uint64{},
		programs: map[string][]accounts.ProgramRow{thisMac + "/linux-1": grantedTo(catalog.ProviderGitHub, catalog.ProviderClaude)}}
	refresh := make(chan struct{}, 1)
	pages := watchWith(t, w, newSteps(), engine.FeedInput{Subject: "linux-1", Refresh: refresh})
	both := []string{string(programNamed(catalog.ProviderGitHub)), string(programNamed(catalog.ProviderClaude))}
	until(t, pages, "GitHub and Claude granted to linux-1", func(p page.Page) bool { return slices.Equal(granted(p), both) })
	w.mu.Lock()
	w.programs[thisMac+"/linux-1"] = grantedTo(catalog.ProviderClaude)
	w.mu.Unlock()
	refresh <- struct{}{}
	p := until(t, pages, "GitHub gone from linux-1's accounts", func(p page.Page) bool { return slices.Equal(granted(p), both[1:]) })
	for _, a := range p.Actions {
		if strings.HasPrefix(a.ID, "remove-grant:") && strings.HasSuffix(a.ID, "/"+both[0]) {
			t.Errorf("linux-1's page still offers %q after GitHub was removed", a.Label)
		}
	}
}

func TestAWorkspaceStartedOnThisMacReachesItsPageWhenTheEngineMoves(t *testing.T) {
	linux := machines.VM{Name: "linux-1", Mac: thisMac, Lifetime: machines.Persistent, State: machines.Stopped}
	w := &world{infra: []machines.VM{infraAt(machines.Running, 0)}, here: []machines.VM{linux}, asked: map[string][]string{}, activity: map[string]uint64{}}
	changed := make(chan struct{}, 1)
	pages := watchWith(t, w, newSteps(), engine.FeedInput{Subject: "linux-1", Changed: changed})
	stopped := func(p page.Page) bool {
		return slices.ContainsFunc(p.Sections, func(s page.Section) bool { return s.ID == "stopped" })
	}
	until(t, pages, "linux-1 stopped", stopped)
	w.mu.Lock()
	linux.State = machines.Running
	linux.Guest = "connected"
	w.here = []machines.VM{linux}
	w.mu.Unlock()
	changed <- struct{}{}
	until(t, pages, "linux-1 running", func(p page.Page) bool { return !stopped(p) && len(p.Segments) > 0 })
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
	w.here = []machines.VM{{Name: "linux-2", Mac: thisMac, Lifetime: machines.Persistent, State: machines.Running, Guest: "connected"}}
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
			created := machines.VM{Name: "linux-2", Mac: on, Lifetime: machines.Persistent, State: machines.Running, Guest: "connected"}
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
// its engine, so its care card says it never sleeps; its times are in the
// owner's own clock.
func TestTheMacMinisPageSaysItsAwakeAndItsTimesInTheOwnersClock(t *testing.T) {
	clock := newSteps()
	clock.zone = time.FixedZone("CET", 3600)
	down := infraAt(machines.Stopped, 0)
	down.CheckStatus.Since = clock.Now().Add(-25 * time.Minute)
	w := &world{infra: []machines.VM{down}, asked: map[string][]string{}, activity: map[string]uint64{},
		awake: map[string]awake.Report{mini: {Engine: awake.EngineRunning, NeverSleeps: true, NeverHeld: true, Servers: []string{"infra"}}}}
	p := until(t, watch(t, w, clock, mini), "mac-mini's Awake", func(p page.Page) bool { return row(p, "care", "awake") != nil })
	if r := row(p, "care", "awake"); r.Label != "Never sleeps" || r.Detail.Text != "hosts infra, which pauses if it sleeps" {
		t.Errorf("mac-mini's Awake reads %q, %q", r.Label, r.Detail.Text)
	}
	if got := w.askedOf(awakecalls.Read.Name); !slices.Contains(got, mini) || slices.Contains(got, thisMac) {
		t.Errorf("Awake was read on %q, want on mac-mini", got)
	}
	if r := row(p, "vms", "infra"); r == nil || len(r.Detail.Items) == 0 || r.Detail.Items[0].Text != "down since 11:05" {
		t.Errorf("infra reads %+v, want down since 11:05 in the owner's clock", r)
	}
}

func gaugeOf(p page.Page, key string) *page.Value {
	for _, s := range p.Sections {
		for _, it := range s.Items {
			if it.Gauge != nil && it.Gauge.Value.Live == key {
				return &it.Gauge.Value
			}
		}
	}
	return nil
}

// value: a clock that moves only when the test leaps it; a sleeper parks
// until its deadline, and the test sees how many are parked.
type virtual struct {
	mu       sync.Mutex
	at       time.Duration
	sleepers map[*nap]bool
	moved    chan struct{}
}

type nap struct {
	until time.Duration
	wake  chan struct{}
}

func newVirtual() *virtual {
	return &virtual{sleepers: map[*nap]bool{}, moved: make(chan struct{})}
}

func (v *virtual) Now() time.Time {
	v.mu.Lock()
	defer v.mu.Unlock()
	return time.Date(2026, 11, 3, 10, 30, 0, 0, time.UTC).Add(v.at)
}

func (v *virtual) since() time.Duration {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.at
}

func (v *virtual) told() {
	close(v.moved)
	v.moved = make(chan struct{})
}

func (v *virtual) Sleep(ctx context.Context, d time.Duration) error {
	v.mu.Lock()
	n := &nap{until: v.at + d, wake: make(chan struct{})}
	v.sleepers[n] = true
	v.told()
	v.mu.Unlock()
	select {
	case <-n.wake:
		return nil
	case <-ctx.Done():
		v.mu.Lock()
		delete(v.sleepers, n)
		v.told()
		v.mu.Unlock()
		return ctx.Err()
	}
}

// value: the deadlines of the sleepers parked once at least n are, within
// 5 s of real time.
func (v *virtual) parked(t *testing.T, n int, what string) []time.Duration {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	for {
		v.mu.Lock()
		until := make([]time.Duration, 0, len(v.sleepers))
		for s := range v.sleepers {
			until = append(until, s.until)
		}
		moved := v.moved
		v.mu.Unlock()
		if len(until) >= n {
			slices.Sort(until)
			return until
		}
		select {
		case <-moved:
		case <-ctx.Done():
			t.Fatalf("%d reads wait on the clock, want %d: %s", len(until), n, what)
		}
	}
}

func (v *virtual) leap(to time.Duration) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.at = to
	for s := range v.sleepers {
		if s.until <= to {
			delete(v.sleepers, s)
			close(s.wake)
		}
	}
	v.told()
}

// value: the world answering each call after its lag on the virtual
// clock; every call at the silent Mac waits until the watch ends, and the
// machines.list it holds back answers This Mac's row first.
type lagged struct {
	w      *world
	clock  *virtual
	lag    map[string]time.Duration
	silent string
}

func (l lagged) wait(ctx context.Context, name string, target machine.Ref) error {
	if l.silent != "" && target.Name == l.silent {
		<-ctx.Done()
		return ctx.Err()
	}
	if d := l.lag[name]; d > 0 {
		return l.clock.Sleep(ctx, d)
	}
	return nil
}

func (l lagged) answer(name string, target machine.Ref, in any) any {
	switch name {
	case portscalls.List.Name:
		of, _ := in.(portscalls.ListIn)
		macs := []ports.MacPorts{{Name: thisMac, ThisMac: true, Reachable: true}}
		if of.AllMachines {
			macs = append(macs, ports.MacPorts{Name: mini, Reachable: true})
		}
		return ports.Listing{AllMachines: ports.AllMachines{Macs: macs}}
	case overviewcalls.Read.Name:
		if target.Name == "" || target.Name == thisMac {
			return overviewcalls.ReadOut{Summary: overview.Summary{Head: overview.Head{Name: thisMac, Up: 3 * 24 * time.Hour}}}
		}
	}
	return l.w.answer(name, target, in)
}

func (l lagged) Invoke(ctx context.Context, target machine.Ref, name string, in, out any, _ call.Progress) error {
	everyMac, _ := in.(portscalls.ListIn)
	if l.silent != "" && (name == machinescalls.List.Name || name == portscalls.List.Name && everyMac.AllMachines) {
		<-ctx.Done()
		return ctx.Err()
	}
	if err := l.wait(ctx, name, target); err != nil {
		return err
	}
	v := l.answer(name, target, in)
	if v == nil {
		return errors.New(name + " is not answered in this world")
	}
	reflect.ValueOf(out).Elem().Set(reflect.ValueOf(v))
	return nil
}

func (l lagged) Watch(ctx context.Context, target machine.Ref, name string, in any, p call.Progress) (<-chan call.Update[any], error) {
	if name != machinescalls.List.Name {
		out := make(chan call.Update[any], 1)
		go func() {
			defer close(out)
			var v any
			if err := l.Invoke(ctx, target, name, in, &v, p); err == nil {
				out <- call.Update[any]{Value: v}
			}
		}()
		return out, nil
	}
	out := make(chan call.Update[any])
	go func() {
		defer close(out)
		if l.wait(ctx, name, target) != nil {
			return
		}
		all, _ := l.w.answer(name, target, in).(machinescalls.ListOut)
		own := all
		own.All.Macs = own.All.Macs[:1]
		for _, u := range []machinescalls.ListOut{own, all} {
			select {
			case out <- call.Update[any]{Value: u}:
			case <-ctx.Done():
				return
			}
			if l.silent != "" {
				<-ctx.Done()
				return
			}
		}
	}()
	return out, nil
}

// value: the machines Home keeps once it drew every Mac.
func homeRead(t *testing.T, w *world) *feeds.HomeRead {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	read := &feeds.HomeRead{}
	in := engine.FeedInput{}
	in.Awake = nothing(in.Awake)
	o := feeds.Menubar{Calls: w, Clock: newSteps(), HomeRead: read, ThisMachine: page.Context{Machine: thisMac}, Report: func(awake.View) *awake.Report { return nil }}
	until(t, o.Feed()(ctx, in), "every Mac", func(p page.Page) bool { return row(p, "machines", mini) != nil })
	return read
}

func laggedPage(t *testing.T, l lagged, home *feeds.HomeRead, subject string) <-chan any {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	o := feeds.Menubar{Calls: l, Clock: l.clock, HomeRead: home, ThisMachine: page.Context{Machine: thisMac}}
	return o.Machine()(ctx, engine.FeedInput{Subject: subject})
}

func readingRows(items []page.Item) []string {
	var out []string
	for _, it := range items {
		if it.Row != nil && it.Row.ID == "reading" {
			out = append(out, it.Row.Label)
		}
		if it.Group != nil {
			out = append(out, readingRows(it.Group.Items)...)
		}
	}
	return out
}

// value: the sections of a page that still draw a part being read.
func stillReading(p page.Page) []string {
	var out []string
	for _, s := range p.Sections {
		if len(readingRows(s.Items)) > 0 {
			out = append(out, s.ID)
		}
	}
	return out
}

func caption(p page.Page) []string {
	for _, s := range p.Sections {
		if s.ID == "summary" {
			out := make([]string, 0, len(s.Caption))
			for _, v := range s.Caption {
				out = append(out, v.Text)
			}
			return out
		}
	}
	return nil
}

func twoMacsAwake() *world {
	return &world{infra: []machines.VM{infraAt(machines.Running, 0)}, asked: map[string][]string{}, activity: map[string]uint64{},
		awake: map[string]awake.Report{thisMac: {Engine: awake.EngineRunning}, mini: {Engine: awake.EngineRunning}}}
}

// invariant: arch L2: with Home watched, a machine's first frame is drawn
// from the machines Home read before any read of the page answers: its header, its
// kind and its standing at the clock's start.
func TestAMachinesPageOpensWithItsHeaderFromHomesRead(t *testing.T) {
	w := twoMacsAwake()
	home := homeRead(t, w)
	l := lagged{w: w, clock: newVirtual(), lag: map[string]time.Duration{}}
	for _, name := range []string{machinescalls.List.Name, portscalls.List.Name, overviewcalls.Read.Name, activitycalls.Read.Name, awakecalls.Read.Name, vaultcalls.VaultList.Name, accountscalls.Programs.Name} {
		l.lag[name] = time.Hour
	}
	first, _ := (<-laggedPage(t, l, home, mini)).(page.Document)
	if first.Header == nil || first.Header.Title != mini || first.Context.Kind != "computer" {
		t.Fatalf("mac-mini's first frame has header %+v and kind %q, want mac-mini's header and a computer's kind", first.Header, first.Context.Kind)
	}
	if got := caption(first.Page); len(got) == 0 || got[0] != "online" {
		t.Errorf("mac-mini's first frame says %q, want its standing first", got)
	}
	if at := l.clock.since(); at != 0 {
		t.Errorf("the first frame came at %s, want at once", at)
	}
	alone, _ := (<-laggedPage(t, l, nil, mini)).(page.Document)
	if alone.Header != nil {
		t.Errorf("with no Home read, mac-mini's first frame has header %+v, want it reading", alone.Header)
	}
}

// invariant: arch L2: every read of a machine's page runs at once and the
// page is drawn after each answer, so it is whole at its slowest read's
// lag, not at the sum of them.
func TestThisMacsPageIsWholeOnceItsSlowestReadAnswers(t *testing.T) {
	w := twoMacsAwake()
	home := homeRead(t, w)
	clock := newVirtual()
	l := lagged{w: w, clock: clock, lag: map[string]time.Duration{
		machinescalls.List.Name: 3 * time.Second, portscalls.List.Name: 2 * time.Second,
		overviewcalls.Read.Name: time.Second, activitycalls.Read.Name: time.Second,
		awakecalls.Read.Name: time.Second, vaultcalls.VaultList.Name: time.Second,
	}}
	pages := laggedPage(t, l, home, thisMac)
	clock.parked(t, 7, "every read of the page asked at once: the machines, both ports, the overview, the activity, the Awake and the vault")
	clock.leap(time.Second)
	p := until(t, pages, "This Mac up 3 days", func(p page.Page) bool { return slices.Contains(caption(p), "up 3 days") })
	if !slices.Contains(stillReading(p), "private") {
		t.Errorf("at 1 s This Mac's ports read %q, want them still reading: they answer at 2 s", stillReading(p))
	}
	clock.leap(2 * time.Second)
	until(t, pages, "This Mac's ports", func(p page.Page) bool { return !slices.Contains(stillReading(p), "private") })
	clock.leap(3 * time.Second)
	until(t, pages, "This Mac's page whole", func(p page.Page) bool { return len(stillReading(p)) == 0 })
	if next := clock.parked(t, 1, "the page's rest"); !slices.Equal(next, []time.Duration{33 * time.Second}) {
		t.Errorf("after its reads the page waits until %v, want its next read 30 s after the slowest answered at 3 s", next)
	}
}

// invariant: arch L2: a Mac that does not answer holds back its own parts
// of This Mac's page alone: This Mac's row, its overview, its ports and
// its Awake are drawn while the clock has not moved.
func TestASilentMacDoesNotHoldBackThisMacsPage(t *testing.T) {
	w := twoMacsAwake()
	clock := newVirtual()
	l := lagged{w: w, clock: clock, silent: mini}
	p := until(t, laggedPage(t, l, nil, thisMac), "This Mac's page while mac-mini is silent", func(p page.Page) bool {
		return p.Header != nil && slices.Contains(caption(p), "up 3 days") && !slices.Contains(stillReading(p), "private") && row(p, "care", "awake") != nil
	})
	if p.Header.Title != "This Mac" && p.Header.Title != thisMac {
		t.Errorf("This Mac's page heads %q", p.Header.Title)
	}
	if at := clock.since(); at != 0 {
		t.Errorf("This Mac's page came at %s, want with the clock unmoved", at)
	}
}

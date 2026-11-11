package process

import (
	"context"
	"errors"
	"path"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jterrazz/studio-station/src/agents"
	"github.com/jterrazz/studio-station/src/awake"
	awakecontract "github.com/jterrazz/studio-station/src/awake/contract"

	"github.com/jterrazz/studio-station/src/activity"
	awakecalls "github.com/jterrazz/studio-station/src/awake/calls"
	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/engine"
	model "github.com/jterrazz/studio-station/src/machines"
	"github.com/jterrazz/studio-station/src/ports"
	portscalls "github.com/jterrazz/studio-station/src/ports/calls"

	enginecalls "github.com/jterrazz/studio-station/src/engine/calls"
)

const (
	stopWait   = 60 * time.Second
	hardWait   = 10 * time.Second
	pollEvery  = 200 * time.Millisecond
	serveEvery = 30
	seenEvery  = time.Minute
	guestReady = 60 * time.Second
	certBudget = 120 * time.Second
)

type Machines interface {
	Records() ([]model.VMRecord, error)
	Save(model.VMRecord) error
	Remove(ctx context.Context, name string) (model.DeleteReport, error)
	PowerOff(ctx context.Context, name string) error
	EngineRunning(ctx context.Context, name string) bool
	ThisMacName() string
	ThisMacLabel() string
	// value: the engine-only machines.uptime-mark: what the keep job marks
	// on a server's record.
	MarkUptime(ctx context.Context, m model.UptimeMark) error
}

type Tailnet interface {
	// value: this Mac's node on the tailnet, and the Audience its shares
	// are opened to (ports/contract.Audience): Share and Unshare.
	Ready(ctx context.Context) (ports.SharingState, error)
	Share(ctx context.Context, s ports.Share) (string, error)
	Unshare(ctx context.Context, s ports.Share) error
	Shared(ctx context.Context) ([]ports.Share, error)
	// value: the ports this node's Funnel opens, by station or anyone.
	Funneled(ctx context.Context) ([]int, error)
	Warm(ctx context.Context) error
}

type Options struct {
	// value: keeps whether the owner stopped sharing a VM's port, so the
	// next listen there is not shared until a share; nil keeps nothing.
	Remember func(vm string, port int, stopped bool) error
	// value: the engine's own state folder: its socket and its children.
	State       string
	ThisMachine catalog.MachineKind
	// value: the ports the catalog's items serve on: a reserved port's HTTP
	// and label, the env a VM's run sets.
	Ports []catalog.ItemPort
	// value: the ports a VM's Machinefile says it may open, read by its
	// name; nil when it says none (or there is none), and nil Rights reads
	// none for every VM.
	Rights    func(ctx context.Context, vm string) (*catalog.PortRights, error)
	Version   string
	Page      int
	PID       int
	Tick      time.Duration
	GuestPort int
	// value: this Mac's power, whose sessions the daemon holds; nil is none.
	Power awakecontract.Power
	// value: the never_sleeps preference, read at start and every 30 ticks.
	NeverSleeps func() bool
	// value: the power page a subscription streams, built by the caller's pages.
	KeepAwakePage func(awake.View) any
	// value: the other pages a subscription streams (the menu bar's), by id.
	Feeds map[string]engine.Feed
	// value: the reading of the live keys the streamed pages list; nil sends
	// no sample.
	Sample engine.Sampler
	// value: the station binary hello names, the one a terminal opens.
	Executable string
	// value: this engine as a client sees it: a second engine asks it first.
	Peer Peer
	// value: the lend records and the recall a lend past its until goes through.
	AccountShares Shares
	// value: the Notifier the engine tells, built over the engine's own log.
	Notifier func(log func(subject, text string)) Notifier
	// value: what runs a server's health check from this Mac; nil runs none.
	Checks CheckRunner
	// value: how the parts of what the engine tells are joined, as the
	// presentation joins them.
	Join func(parts ...string) string
	// value: what listens on this Mac, and what the owner declared public;
	// nil reads nothing, and the exposure job does not run.
	Listening    func(ctx context.Context) ([]ports.Listener, error)
	Declarations func(ctx context.Context) ([]ports.Declaration, error)
	// value: when the engine process started, for a stepped engine that
	// stands for one which outlived its command; zero is the Begin.
	StartedAt time.Time
	// value: the jobs of engine/jobs, each run at every tick: each decides
	// by its own schedule whether it is due.
	Jobs []Tick
	// value: SPEC B5.14: the Macs the owner asked to be told of once they
	// answer again, whether one answers now, and the ask cleared once told;
	// nil tells of none.
	TellBack func() map[string]bool
	Answers  func(ctx context.Context, mac string) bool
	ToldBack func(mac string) error
	Label    func(mac string) string
	// value: the names every machine's agents and workspaces hold, read
	// across every Mac; nil holds none.
	AgentsHeld func(ctx context.Context) (agents.Held, error)
}

type Engine struct {
	h      Mac
	hv     Hypervisor
	guests Guests
	m      Machines
	tn     Tailnet
	st     Stores
	o      Options

	mu  sync.Mutex
	vms map[string]*vmState
	// value: when the Macs the owner waits on were last asked.
	askedBack time.Time
	svcs      map[string]*svc
	// value: every address each running VM listens on, as its guest
	// reported it, shared or not.
	heard map[string][]ports.Listener
	// value: what each VM that stopped was last heard listening on, until
	// its guest reports again or the VM is forgotten.
	lastHeard map[string][]ports.Listener
	// value: when the exposure job last read, what each VM listened on
	// then, and the red ports it told.
	exposedAt    time.Time
	scanned      map[string][]ports.Listener
	red          []string
	reservations []ports.Reservation
	// value: each VM's rights, as its last run read them.
	rights map[string]ports.Rights
	// value: the HTTPS mode this engine's shares were judged and served
	// in: its start's, then each tick's while Serve is on.
	servedMode string
	self       ports.SharingState
	read       bool
	issuing    bool
	started    time.Time
	note       func(line string)
	awakeJob   *awakecalls.Keeper
	awake      *awake.Kept
	hub        hub
	history    *activity.History
	watchers   watchers
	settings   engine.SettingsKept

	notifier Notifier
	serve    func(ctx context.Context, conn Stream)
	memory   engine.Memory
	// value: the engine's own lifetime, from Begin: the guest sessions it
	// holds live in it, never in the request that started a VM.
	life context.Context

	shareMu        sync.Mutex
	sharesAt       time.Time
	takeBackFailed map[string]time.Time
}

type Stores struct {
	Ports        Policy
	Reservations Reservations
}

type Policy interface {
	Get() (*ports.PortsGroup, error)
}

type Reservations interface {
	// value: the reserved ports, kept by their store; the engine is their one
	// writer while it runs.
	Get(ctx context.Context, vm string, guestPort int) (ports.Reservation, error)
	Put(ctx context.Context, r ports.Reservation) error
	List(ctx context.Context) ([]ports.Reservation, error)
	Delete(ctx context.Context, vm string, guestPort int) error
}

func New(h Mac, hv Hypervisor, guests Guests, m Machines, tn Tailnet, st Stores, o Options) *Engine {
	if o.Tick == 0 {
		o.Tick = time.Second
	}
	if o.GuestPort == 0 {
		o.GuestPort = catalog.GuestSSH
	}
	d := &Engine{h: h, hv: hv, guests: guests, m: m, tn: tn, st: st, o: o, vms: map[string]*vmState{}, svcs: map[string]*svc{}, heard: map[string][]ports.Listener{}, lastHeard: map[string][]ports.Listener{}, rights: map[string]ports.Rights{}, takeBackFailed: map[string]time.Time{}}
	d.awakeJob = d.newAwake()
	if o.Notifier != nil {
		d.notifier = o.Notifier(d.logf)
	}
	return d
}

func (d *Engine) ServeWith(serve func(ctx context.Context, conn Stream)) { d.serve = serve }

func (d *Engine) Check() error {
	// value: the server of station.api/v2, run on each connection of the
	// socket; it is the engine's, built over this daemon, so it is handed after New.
	if d.o.ThisMachine != catalog.Mac {
		return &enginecalls.MacOnlyError{Kind: d.o.ThisMachine}
	}
	return nil
}

func (d *Engine) policy() ports.Policy {
	g, err := d.st.Ports.Get()
	if err != nil {
		return ports.PolicyOf(nil)
	}
	return ports.PolicyOf(g)
}

func (d *Engine) saveReservations() {
	_ = portscalls.SaveReservations(context.Background(), d.st.Reservations, append([]ports.Reservation(nil), d.reservations...))
}

func (d *Engine) Run(ctx context.Context, note func(line string)) error {
	// invariant: the socket's liveness is the lock: a daemon that answers ping makes
	// a second one exit; a socket file nobody answers is removed and made again.
	// The socket is made only once adoption and serve reconciliation are done,
	// so its file means the daemon is ready.
	if err := d.Check(); err != nil {
		return err
	}
	sock := engine.SocketPath(d.o.State)
	if d.o.Peer != nil {
		if p, err := d.o.Peer.Hello(ctx); err == nil {
			return &enginecalls.RunningError{PID: p.PID}
		}
	}
	if err := d.Begin(ctx, note); err != nil {
		return err
	}
	_ = d.h.Remove(sock)
	l, err := d.h.ListenUnix(sock)
	if err != nil {
		return err
	}
	defer func() {
		_ = l.Close()
		_ = d.h.Remove(sock)
	}()
	go d.accept(ctx, l)
	go d.watchAwake(ctx)
	for tick := 1; ; tick++ {
		if err := d.h.Sleep(ctx, d.o.Tick); err != nil {
			d.awakeJob.Quit()
			d.closeAll()
			return nil
		}
		d.step(ctx, tick)
	}
}

func (d *Engine) Begin(ctx context.Context, note func(line string)) error {
	// invariant: Begin is the engine up to its socket: the clamshell set back
	// to normal (a killed daemon's mask is cleared by its successor), the
	// children a killed one left reaped, the runs adopted, the lends swept and
	// Serve reconciled. Run follows it with the socket and a tick a second; a
	// stepped engine (the spec world) follows it with Step, then Serve per
	// connection, and Stop.
	if err := d.Check(); err != nil {
		return err
	}
	d.note, d.life = note, ctx
	d.awakeJob.ClearClamshell()
	if d.awake != nil {
		d.awakeJob.Restore(*d.awake)
	}
	if n := d.h.Reap(engine.ChildrenDir(d.o.State)); n > 0 {
		d.logf("children", "killed "+strconv.Itoa(n)+" process(es) a killed engine left running")
	}
	if err := d.h.MkdirAll(d.o.State); err != nil {
		return err
	}
	d.started = d.h.Now().UTC().Truncate(time.Second)
	if !d.o.StartedAt.IsZero() {
		d.started = d.o.StartedAt.UTC().Truncate(time.Second)
	}
	d.mu.Lock()
	d.history = activity.NewHistory(d.started)
	d.mu.Unlock()
	d.neverSleeps(d.started)
	d.reservations, _ = d.st.Reservations.List(ctx)
	// invariant: the reservations are judged in the mode of this start, at
	// adoption, so a switch made before Serve is ever on is judged from it.
	d.servedMode = d.policy().HTTPS
	d.refreshSelf(ctx)
	d.adopt(ctx)
	d.sweepShares(ctx)
	d.reconcileServe(ctx)
	return nil
}

func (d *Engine) Step(ctx context.Context, tick int) bool {
	// value: one tick of a stepped engine; it reports whether it is busy: a VM
	// changed, a run waits for its delay, or a health check runs.
	before := d.vmViews()
	d.step(ctx, tick)
	return !reflect.DeepEqual(before, d.vmViews()) || d.waiting()
}

func (d *Engine) waiting() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, v := range d.vms {
		if !v.rerunAt.IsZero() || v.checking {
			return true
		}
	}
	return false
}

func (d *Engine) Memory() engine.Memory {
	d.mu.Lock()
	defer d.mu.Unlock()
	out := engine.Memory{}
	for name, v := range d.vms {
		heard, ok := d.heard[name]
		if !ok {
			heard = d.lastHeard[name]
		}
		out[name] = engine.Remembered{
			State: v.state, Detail: v.detail, Reruns: v.reruns, RerunAt: v.rerunAt, Restarts: v.restarts, BootAt: v.bootAt, Check: v.check,
			Latency: v.latency, Restart: v.walk, Down: v.down, Cause: v.cause, LastLines: v.lastLines, LastHeard: heard, Agents: keptReport(v.agents), Reported: v.reported,
			ReadyAt: v.readyAt,
		}
	}
	return out
}

func (d *Engine) Remember(m engine.Memory) {
	// invariant: Remember comes before Begin: a VM it holds broken, or waiting
	// for its delay, is not run again by the adoption.
	d.memory = m
}

func (d *Engine) Serve(ctx context.Context, conn Stream) {
	defer func() { _ = conn.Close() }()
	if d.serve != nil {
		d.serve(ctx, conn)
	}
}

func (d *Engine) Stop() {
	d.awakeJob.Quit()
	d.closeAll()
}

func (d *Engine) step(ctx context.Context, tick int) {
	if tick%serveEvery == 0 {
		d.refreshSelf(ctx)
		d.neverSleeps(time.Time{})
	}
	d.tick(ctx)
	d.sweepShares(ctx)
	d.watchExposure(ctx)
	d.awakeJob.Tick()
	d.tellBack(ctx)
	d.watchState()
	for _, job := range d.o.Jobs {
		job(ctx)
	}
}

func (d *Engine) logf(subject, message string) {
	if d.note != nil {
		d.note(d.h.Now().UTC().Format(time.RFC3339) + " " + subject + ": " + message)
	}
}

// invariant: Serve's state is read at start and every serveEvery ticks: on,
// every published reservation is served and the entries no reservation justifies go; off,
// no serve runs and every row says so, logged once per change.
func (d *Engine) refreshSelf(ctx context.Context) {
	self, err := d.tn.Ready(ctx)
	if err != nil {
		self = ports.SharingState{}
	}
	d.mu.Lock()
	was, first := d.self.On, !d.read
	d.self, d.read = self, true
	d.mu.Unlock()
	if !first && self.On == was {
		return
	}
	if self.On {
		d.warmCert(ctx, self.DNSName)
	}
	switch {
	case self.On && !first:
		d.logf("tailnet", "Serve is on: every reserved port is shared")
		// invariant: the HTTPS mode is judged first and the entries no
		// reservation justifies go before any share is served, so none is
		// served on a Funnel port the mode no longer gives it.
		mode := d.policy().HTTPS
		d.mu.Lock()
		d.serveInModeLocked(ctx, mode)
		d.mu.Unlock()
		d.reconcileServe(ctx)
		d.serveAll(ctx)
	case !self.On && self.Backend != ports.Running:
		d.logf("tailnet", catalog.SignedOutDetail+": ports are local only")
		d.unserveAll()
	case !self.On:
		d.logf("tailnet", catalog.ServeOffDetail+": ports are local only")
		d.unserveAll()
	}
}

// invariant: the tailnet issues the host's certificate at the first https
// request, 34.5 s measured, so the daemon asks for it as Serve turns on, in
// the background: never blocking a publish, one at a time, within its budget.
func (d *Engine) warmCert(ctx context.Context, name string) {
	d.mu.Lock()
	if d.issuing || name == "" {
		d.mu.Unlock()
		return
	}
	d.issuing = true
	d.mu.Unlock()
	go func() {
		start := d.h.Now()
		wait, cancel := context.WithDeadline(ctx, start.Add(certBudget))
		err := d.tn.Warm(wait)
		late := wait.Err() == context.DeadlineExceeded
		cancel()
		d.mu.Lock()
		d.issuing = false
		d.mu.Unlock()
		switch {
		case ctx.Err() != nil:
		case err == nil:
			d.logf("tailnet", "certificate for "+name+" ready in "+d.h.Now().Sub(start).Round(time.Millisecond).String())
		case late || errors.Is(err, context.DeadlineExceeded):
			d.logf("tailnet", "certificate for "+name+" not issued: no answer within "+certBudget.String())
		default:
			d.logf("tailnet", "certificate for "+name+" not issued: "+err.Error())
		}
	}()
}

func (d *Engine) accept(ctx context.Context, l Listener) {
	for {
		conn, err := l.Accept()
		if err != nil {
			return
		}
		go d.Serve(ctx, conn)
	}
}

const accountsBudget = 12 * time.Second

// invariant: each running VM is asked by its own guest, all at once, each
// within accountsBudget; a VM whose guest is not connected is "absent" with
// no rows. Only the guest's parsed rows travel: never a provider's output.
func (d *Engine) accounts(ctx context.Context) engine.AccountsView {
	d.mu.Lock()
	var running []*vmState
	for _, v := range d.vms {
		if v.state == engine.StateRunning {
			running = append(running, v)
		}
	}
	d.mu.Unlock()
	out := make([]engine.VMAccounts, len(running))
	var wg sync.WaitGroup
	for i, v := range running {
		wg.Add(1)
		go func() {
			defer wg.Done()
			d.mu.Lock()
			guest := v.guest
			d.mu.Unlock()
			out[i] = engine.VMAccounts{Name: v.rec.Name, Guest: engine.GuestAbsent}
			if guest != engine.GuestConnected {
				return
			}
			rows, err := d.askGuest(ctx, v, "accounts", accountsBudget)
			if err != nil {
				d.logf(v.rec.Name+" accounts", err.Error())
				return
			}
			out[i] = engine.VMAccounts{Name: v.rec.Name, Guest: engine.GuestConnected, Accounts: rows}
		}()
	}
	wg.Wait()
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return engine.AccountsView{VMs: out}
}

// invariant: stop_all stops every VM of this host that runs, at once, with
// the wish stopped and its disk kept, whatever its lifetime: only a delete
// destroys. Each answers its own result, so a VM that would not stop keeps
// the engine that supervises it.
func (d *Engine) stopAll(ctx context.Context) []engine.Stopped {
	records, _ := d.m.Records()
	var wg sync.WaitGroup
	var mu sync.Mutex
	var out []engine.Stopped
	for _, r := range records {
		v, err := d.find(r.Name)
		if err != nil {
			continue
		}
		d.mu.Lock()
		running := v.state == engine.StateRunning
		d.mu.Unlock()
		if !running || r.Lifetime == model.AlwaysOn {
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			s := engine.Stopped{Name: r.Name, Lifetime: string(r.Lifetime), Result: engine.ResultStopped}
			err := d.halt(ctx, v, true)
			if err == nil && d.m.EngineRunning(ctx, r.Name) {
				err = errors.New("the engine still runs it")
			}
			if err != nil {
				s.Result, s.Detail = engine.ResultFailed, err.Error()
				d.logf(r.Name, "stop_all: "+err.Error())
			}
			mu.Lock()
			out = append(out, s)
			mu.Unlock()
		}()
	}
	wg.Wait()
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (d *Engine) sharedVMs() map[string]bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	out := map[string]bool{}
	for _, l := range d.reservations {
		out[l.VM] = true
	}
	return out
}

func (d *Engine) closeAll() {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, s := range d.svcs {
		for _, l := range s.listeners {
			_ = l.Close()
		}
		s.listeners = nil
	}
	for _, v := range d.vms {
		if v.cancel != nil {
			v.cancel()
		}
	}
}

func (d *Engine) hostKeys() []string {
	home := d.h.Home()
	var keys []string
	if pub, err := d.h.ReadFile(path.Join(home, ".ssh", "id_ed25519.pub")); err == nil {
		if line := strings.TrimSpace(string(pub)); line != "" {
			keys = append(keys, line)
		}
	}
	if auth, err := d.h.ReadFile(path.Join(home, ".ssh", "authorized_keys")); err == nil {
		for _, line := range strings.Split(string(auth), "\n") {
			if line = strings.TrimSpace(line); line != "" && !strings.HasPrefix(line, "#") {
				keys = append(keys, line)
			}
		}
	}
	return keys
}

//go:build world

package wire

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/call/lanes"
	"github.com/jterrazz/studio-station/apps/engine/engine/process"
	model "github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/ports"
	"github.com/jterrazz/studio-station/apps/engine/world"
)

type quietMac struct{}

func (quietMac) Now() time.Time                               { return time.Unix(1, 0) }
func (quietMac) Sleep(context.Context, time.Duration) error   { return nil }
func (quietMac) Home() string                                 { return "/home" }
func (quietMac) ReadFile(string) ([]byte, error)              { return nil, context.Canceled }
func (quietMac) Exists(string) bool                           { return false }
func (quietMac) MkdirAll(string) error                        { return nil }
func (quietMac) Remove(string) error                          { return nil }
func (quietMac) ListenUnix(string) (process.Listener, error)  { return nil, context.Canceled }
func (quietMac) ListenLoopback(int) (process.Listener, error) { return nil, context.Canceled }
func (quietMac) Join(process.Stream, process.Stream)          {}
func (quietMac) Reap(string) int                              { return 0 }

type quietMachines struct{}

func (quietMachines) Records() ([]model.VMRecord, error) { return nil, nil }
func (quietMachines) Save(model.VMRecord) error          { return nil }
func (quietMachines) Remove(context.Context, string) (model.DeleteReport, error) {
	return model.DeleteReport{}, nil
}
func (quietMachines) PowerOff(context.Context, string) error             { return nil }
func (quietMachines) EngineRunning(context.Context, string) bool         { return false }
func (quietMachines) ThisMacName() string                                { return "here" }
func (quietMachines) ThisMacLabel() string                               { return "here" }
func (quietMachines) MarkUptime(context.Context, model.UptimeMark) error { return nil }

type quietTailnet struct{}

func (quietTailnet) Ready(context.Context) (ports.SharingState, error) {
	return ports.SharingState{}, nil
}
func (quietTailnet) Share(context.Context, ports.Share) (string, error) { return "", nil }
func (quietTailnet) Unshare(context.Context, ports.Share) error         { return nil }
func (quietTailnet) Shared(context.Context) ([]ports.Share, error)      { return nil, nil }
func (quietTailnet) Funneled(context.Context) ([]int, error)            { return nil, nil }
func (quietTailnet) Warm(context.Context) error                         { return nil }

type quietPolicy struct{}

func (quietPolicy) Get() (*ports.PortsGroup, error) { return &ports.PortsGroup{}, nil }

type quietReservations struct{ process.Reservations }

func (quietReservations) List(context.Context) ([]ports.Reservation, error) { return nil, nil }

func quietEngine(kind catalog.MachineKind, r process.Reservations) *process.Engine {
	return process.New(quietMac{}, nil, nil, quietMachines{}, quietTailnet{}, process.Stores{Ports: quietPolicy{}, Reservations: r}, process.Options{ThisMachine: kind, State: "/state"})
}

func begunEngine(t *testing.T) *process.Engine {
	t.Helper()
	d := quietEngine(catalog.Mac, quietReservations{})
	if err := d.Begin(context.Background(), func(string) {}); err != nil {
		t.Fatal(err)
	}
	return d
}

func coldMac(start func(m *worldMac) *process.Engine) *worldMac {
	w := &world.World{ThisMac: "here", Macs: map[string]*world.Mac{"here": {Engine: &world.Engine{StartedAt: time.Unix(1, 0)}}}}
	m := (&worldRun{w: w, macs: map[string]*worldMac{}}).mac("here")
	m.start = func() *process.Engine { return start(m) }
	return m
}

type dialingReservations struct {
	quietReservations
	dial func(ctx context.Context)
}

func (r dialingReservations) List(ctx context.Context) ([]ports.Reservation, error) {
	r.dial(ctx)
	return nil, nil
}

// value: a caller's context that says when the caller waits on it.
type askedContext struct {
	context.Context
	asked chan<- struct{}
}

func (c askedContext) Done() <-chan struct{} {
	select {
	case c.asked <- struct{}{}:
	default:
	}
	return c.Context.Done()
}

func dialMac(ctx context.Context, m *worldMac) error {
	conn, err := worldSockets{m: m}.DialUnix(ctx, "", 0)
	if err == nil {
		_ = conn.Close()
	}
	return err
}

func TestWorldMacUpNeverAnswersAnEngineThatEnded(t *testing.T) {
	for range 200 {
		w := &world.World{ThisMac: "here", Macs: map[string]*world.Mac{"here": {Engine: &world.Engine{StartedAt: time.Unix(1, 0)}}}}
		run := &worldRun{w: w, macs: map[string]*worldMac{}}
		m := run.mac("here")
		m.hold(begunEngine(t), m.engineStarted())

		var (
			wg  sync.WaitGroup
			got *process.Engine
			ok  bool
		)
		wg.Add(2)
		go func() { defer wg.Done(); got, ok = m.up(context.Background()) }()
		go func() { defer wg.Done(); m.down(false) }()
		wg.Wait()
		if ok && got == nil {
			t.Fatal("up answered ok with a nil engine: it read the engine twice while down ended it")
		}
	}
}

func TestWorldMacCallersThatMeetAStartWaitForItsEngine(t *testing.T) {
	const callers = 4
	for range 200 {
		m := coldMac(func(*worldMac) *process.Engine { return quietEngine(catalog.Mac, quietReservations{}) })
		barrier := make(chan struct{})
		errs := make([]error, callers)
		var wg sync.WaitGroup
		for i := range callers {
			wg.Go(func() {
				<-barrier
				errs[i] = dialMac(context.Background(), m)
			})
		}
		close(barrier)
		wg.Wait()
		for i, err := range errs {
			if err != nil {
				t.Fatalf("caller %d of %d dialed a cold Mac and was refused: %v", i+1, callers, err)
			}
		}
	}
}

func TestWorldMacStartThatDialsItsOwnMacIsRefusedNotWaited(t *testing.T) {
	var (
		once  sync.Once
		inner error
	)
	m := coldMac(func(m *worldMac) *process.Engine {
		return quietEngine(catalog.Mac, dialingReservations{dial: func(ctx context.Context) {
			once.Do(func() { inner = dialMac(ctx, m) })
		}})
	})
	done := make(chan error, 1)
	go func() { done <- dialMac(context.Background(), m) }()
	bound, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("the start that dialed its own Mac failed: %v", err)
		}
	case <-bound.Done():
		t.Fatal("the start dialed its own Mac and waited for itself")
	}
	if inner == nil {
		t.Fatal("the engine's own dial during its start was answered, not refused")
	}
}

func TestWorldMacStartBegunInAnEnginesLifetimeLeapsTheClockItself(t *testing.T) {
	started, release := make(chan struct{}), make(chan struct{})
	m := coldMac(func(*worldMac) *process.Engine {
		close(started)
		<-release
		return quietEngine(catalog.Mac, quietReservations{})
	})
	inside := context.WithValue(context.Background(), waits{}, true)
	go func() { _ = dialMac(inside, m) }()
	<-started
	bound, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- dialMac(context.Background(), m) }()
	close(release)
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("the command that met an engine's start was refused: %v", err)
		}
	case <-bound.Done():
		t.Fatal("the start begun in an engine's lifetime waited on a clock only the command it blocks moves")
	}
}

func TestWorldMacWaiterLeavesWhenItsContextEnds(t *testing.T) {
	started, release := make(chan struct{}), make(chan struct{})
	m := coldMac(func(*worldMac) *process.Engine {
		close(started)
		<-release
		return quietEngine(catalog.Mac, quietReservations{})
	})
	defer close(release)
	go func() { _ = dialMac(context.Background(), m) }()
	<-started
	bound, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	asked := make(chan struct{}, 1)
	ctx, leave := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- dialMac(askedContext{Context: ctx, asked: asked}, m) }()
	select {
	case <-asked:
	case err := <-done:
		t.Fatalf("a caller that met a start did not wait for it: %v", err)
	case <-bound.Done():
		t.Fatal("a caller waited for the start without its context")
	}
	leave()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("a waiter whose context ended was answered an engine")
		}
	case <-bound.Done():
		t.Fatal("a waiter whose context ended still waited for the start")
	}
}

func TestWorldMacStartThatFailsFailsEveryWaiter(t *testing.T) {
	const waiters = 3
	started, asked := make(chan struct{}), make(chan struct{}, 4*waiters)
	bound, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var starts atomic.Int32
	m := coldMac(func(*worldMac) *process.Engine {
		if starts.Add(1) > 1 {
			return quietEngine(catalog.Mac, quietReservations{})
		}
		close(started)
		for range waiters {
			select {
			case <-asked:
			case <-bound.Done():
				t.Error("a waiter never asked for the start it waited on")
			}
		}
		return quietEngine(catalog.Linux, quietReservations{})
	})
	errs := make([]error, waiters+1)
	var wg sync.WaitGroup
	wg.Go(func() { errs[waiters] = dialMac(context.Background(), m) })
	<-started
	for i := range waiters {
		wg.Go(func() { errs[i] = dialMac(askedContext{Context: context.Background(), asked: asked}, m) })
	}
	wg.Wait()
	if n := starts.Load(); n != 1 {
		t.Fatalf("a start that failed was begun %d times: a waiter began its own", n)
	}
	for i, err := range errs {
		if err == nil {
			t.Fatalf("caller %d was answered an engine by a start that failed", i)
		}
	}
	if err := dialMac(context.Background(), m); err != nil {
		t.Fatalf("a caller after the failed start was refused instead of beginning its own: %v", err)
	}
	if n := starts.Load(); n != 2 {
		t.Fatalf("the caller after a failed start left %d starts, want 2", n)
	}
}

func TestWorldMacFirstCallersServeAFarMacOnce(t *testing.T) {
	const callers = 16
	w := &world.World{ThisMac: "here", Macs: map[string]*world.Mac{
		"here":  {Engine: &world.Engine{StartedAt: time.Unix(1, 0)}},
		"there": {Engine: &world.Engine{StartedAt: time.Unix(1, 0)}},
	}}
	var built atomic.Int32
	run := &worldRun{w: w, macs: map[string]*worldMac{}}
	run.farFrom(func(*worldMac) farMac {
		built.Add(1)
		return farMac{start: func() *process.Engine { return quietEngine(catalog.Mac, quietReservations{}) }}
	})
	m := run.mac("there")
	barrier := make(chan struct{})
	var wg sync.WaitGroup
	for range callers {
		wg.Go(func() {
			<-barrier
			if _, ok := m.up(context.Background()); !ok {
				t.Error("a first caller of a far Mac was refused")
			}
		})
	}
	close(barrier)
	wg.Wait()
	if n := built.Load(); n != 1 {
		t.Fatalf("a far Mac was built %d times by concurrent first callers, want 1", n)
	}
}

func TestWorldMacStartThatJoinsAPoolJobWaitingOnItEnds(t *testing.T) {
	bound, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	type frame struct{ dialed error }
	var pool *lanes.Pool
	read := func(ctx context.Context, m *worldMac) <-chan frame {
		return lanes.Run(ctx, pool, frame{}, []lanes.Job[frame]{{Key: "overview/machine/daemon",
			Do: func(ctx context.Context) func(*frame) {
				err := dialMac(ctx, m)
				return func(f *frame) { f.dialed = err }
			},
			Late: func(*frame) {},
		}}, func(f frame) frame { return f })
	}
	drained := func(answers <-chan frame) bool {
		for {
			select {
			case _, open := <-answers:
				if !open {
					return true
				}
			case <-bound.Done():
				return false
			}
		}
	}
	started, release := make(chan struct{}), make(chan struct{})
	var (
		once   sync.Once
		joined atomic.Bool
	)
	m := coldMac(func(m *worldMac) *process.Engine {
		close(started)
		select {
		case <-release:
		case <-bound.Done():
		}
		return quietEngine(catalog.Mac, dialingReservations{dial: func(ctx context.Context) {
			once.Do(func() { joined.Store(drained(read(ctx, m))) })
		}})
	})
	pool = lanes.NewPool(m.poolContext(context.Background()), m.budgets(), 1, 1)
	direct := make(chan error, 1)
	go func() { direct <- dialMac(context.Background(), m) }()
	select {
	case <-started:
	case <-bound.Done():
		t.Fatal("the direct caller never began the start")
	}
	outside := read(context.Background(), m)
	close(release)
	select {
	case err := <-direct:
		if err != nil {
			t.Fatalf("the direct caller was refused the engine it began: %v", err)
		}
	case <-bound.Done():
		t.Fatal("the start joined the pool job that waited on it, and neither ended")
	}
	if !joined.Load() {
		t.Fatal("the start's read through the pool joined a job that waited on the start")
	}
	if !drained(outside) {
		t.Fatal("the pool job that met the start never ended")
	}
}

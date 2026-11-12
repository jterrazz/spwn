//go:build world

package wire

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/engine/process"
	model "github.com/jterrazz/studio-station/src/machines"
	"github.com/jterrazz/studio-station/src/ports"
	"github.com/jterrazz/studio-station/src/world"
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

func begunEngine(t *testing.T) *process.Engine {
	t.Helper()
	d := process.New(quietMac{}, nil, nil, quietMachines{}, quietTailnet{}, process.Stores{Ports: quietPolicy{}, Reservations: quietReservations{}}, process.Options{ThisMachine: catalog.Mac, State: "/state"})
	if err := d.Begin(context.Background(), func(string) {}); err != nil {
		t.Fatal(err)
	}
	return d
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

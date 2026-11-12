package jobs

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/src/activity"
	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/core/machine"
)

type clock struct{ now time.Time }

func (c *clock) Now() time.Time { return c.now }

type machines struct {
	clock *clock
	mu    sync.Mutex
	asked map[string]int
}

func (m *machines) Invoke(_ context.Context, target machine.Ref, _ string, _, out any, _ call.Progress) error {
	m.mu.Lock()
	m.asked[target.Name]++
	m.mu.Unlock()
	r, ok := out.(*activity.Reading)
	if !ok {
		return errors.New("activity.sample answers a Reading")
	}
	*r = activity.Reading{Sample: activity.Sample{At: m.clock.Now()}}
	return nil
}

func (m *machines) Watch(context.Context, machine.Ref, string, any, call.Progress) (<-chan call.Update[any], error) {
	return nil, nil
}

func TestTwentyIdleVMsAreSampledOnceAMinuteEach(t *testing.T) {
	c := &clock{now: time.Date(2026, 11, 4, 10, 0, 0, 0, time.UTC)}
	guests := &machines{clock: c, asked: map[string]int{}}
	seen := make([]Seen, 0, 21)
	seen = append(seen, Seen{Ref: machine.Ref{Name: "mac"}, Kind: machine.Computer})
	for i := range 20 {
		seen = append(seen, Seen{Ref: machine.Ref{Name: "vm-" + strconv.Itoa(i), On: "mac"}, Kind: machine.Workspace, PID: 1000 + i, Guest: true})
	}
	watched := ""
	h := activity.NewHistory(c.now)
	s := &Sample{
		Calls: guests, Clock: c, Seen: func() []Seen { return seen }, Watched: func(m machine.Ref) bool { return m.Name == watched },
		History: func() *activity.History { return h }, Tell: func(string, string, string) {},
	}
	tick := func(seconds int) {
		for range seconds {
			s.Run(t.Context())
			c.now = c.now.Add(time.Second)
		}
	}
	tick(600)
	for _, m := range seen {
		if got := guests.asked[m.Ref.Name]; got != 10 {
			t.Errorf("%s was sampled %d times in 10 idle minutes; want 10, once a minute", m.Ref.Name, got)
		}
	}
	watched = "vm-3"
	tick(60)
	for _, m := range seen {
		want, every := 1, "once, the others unwatched"
		if m.Ref.Name == watched {
			want, every = 30, "30, every 2 s"
		}
		if got := guests.asked[m.Ref.Name] - 10; got != want {
			t.Errorf("%s was sampled %d times in the minute vm-3 was watched; want %s", m.Ref.Name, got, every)
		}
	}
}

type silentMeter struct{}

func (silentMeter) Sample(context.Context) (activity.Sample, error) { return activity.Sample{}, nil }
func (silentMeter) Top(context.Context) (activity.Busiest, error)   { return activity.Busiest{}, nil }
func (silentMeter) Process(context.Context, int) (activity.Usage, error) {
	return activity.Usage{CPU: 0.18, Memory: 4 << 30}, nil
}

func TestAVMWhoseGuestIsSilentKeepsAGapInItsHistoryNeverZeros(t *testing.T) {
	c := &clock{now: time.Date(2026, 11, 4, 10, 0, 0, 0, time.UTC)}
	h := activity.NewHistory(c.now)
	s := &Sample{
		Calls: &machines{clock: c, asked: map[string]int{}}, Meter: silentMeter{}, Clock: c,
		Seen: func() []Seen {
			return []Seen{{Ref: machine.Ref{Name: "linux-1", On: "mac"}, Kind: machine.Workspace, PID: 4242}}
		},
		Watched: func(machine.Ref) bool { return true },
		History: func() *activity.History { return h }, Tell: func(string, string, string) {},
	}
	for range 10 {
		s.Run(t.Context())
		c.now = c.now.Add(time.Second)
	}
	kept, _ := h.Of("linux-1")
	if kept.Last.Run == nil || kept.Last.Run.CPU != 0.18 {
		t.Errorf("linux-1's last reading is %+v; want its process's usage on its Mac", kept.Last)
	}
	if fine, coarse := kept.Series.Fine(), kept.Series.Coarse(); len(fine)+len(coarse) != 0 {
		t.Errorf("linux-1's history holds %d and %d points while its guest was silent; want none, a gap", len(fine), len(coarse))
	}
}

type blindMeter struct{ silentMeter }

func (blindMeter) Process(context.Context, int) (activity.Usage, error) {
	return activity.Usage{}, errors.New("no such process")
}

type lateGuest struct {
	machines
	from time.Time
}

func (l *lateGuest) Invoke(ctx context.Context, target machine.Ref, name string, in, out any, p call.Progress) error {
	if l.clock.now.Before(l.from) {
		return errors.New("the guest is not connected yet")
	}
	return l.machines.Invoke(ctx, target, name, in, out, p)
}

func TestAVMWhoseGuestConnectsAfterTheFirstTickGetsItsPointAtOnceNotAWindowLater(t *testing.T) {
	c := &clock{now: time.Date(2026, 11, 4, 10, 0, 0, 0, time.UTC)}
	h := activity.NewHistory(c.now)
	guests := &lateGuest{machines: machines{clock: c, asked: map[string]int{}}, from: c.now.Add(time.Second)}
	s := &Sample{
		Calls: guests, Meter: blindMeter{}, Clock: c,
		Seen: func() []Seen {
			return []Seen{{Ref: machine.Ref{Name: "linux-1", On: "mac"}, Kind: machine.Workspace, PID: 4242, Guest: !c.now.Before(guests.from)}}
		},
		Watched: func(machine.Ref) bool { return false },
		History: func() *activity.History { return h }, Tell: func(string, string, string) {},
	}
	for range 5 {
		s.Run(t.Context())
		c.now = c.now.Add(time.Second)
	}
	kept, _ := h.Of("linux-1")
	if got := len(kept.Series.Coarse()); got != 1 {
		t.Errorf("linux-1 has %d coarse points 5 s after its guest connected; want 1, not a window's wait", got)
	}
}

type hung struct {
	machines
	waiting  sync.WaitGroup
	inFlight sync.Mutex
	at       int
	most     int
}

func (h *hung) Invoke(ctx context.Context, target machine.Ref, name string, in, out any, p call.Progress) error {
	if target.On == "" {
		return h.machines.Invoke(ctx, target, name, in, out, p)
	}
	h.inFlight.Lock()
	h.at++
	h.most = max(h.most, h.at)
	h.inFlight.Unlock()
	h.waiting.Done()
	<-ctx.Done()
	h.inFlight.Lock()
	h.at--
	h.inFlight.Unlock()
	return ctx.Err()
}

func TestTwentyHungGuestsCostATickOneBudgetNotTwenty(t *testing.T) {
	c := &clock{now: time.Date(2026, 11, 4, 10, 0, 0, 0, time.UTC)}
	seen := make([]Seen, 0, 21)
	seen = append(seen, Seen{Ref: machine.Ref{Name: "mac"}, Kind: machine.Computer})
	for i := range 20 {
		seen = append(seen, Seen{Ref: machine.Ref{Name: "vm-" + strconv.Itoa(i), On: "mac"}, Kind: machine.Workspace, Guest: true})
	}
	h := activity.NewHistory(c.now)
	guests := &hung{machines: machines{clock: c, asked: map[string]int{}}}
	guests.waiting.Add(20)
	s := &Sample{
		Calls: guests, Clock: c,
		Seen: func() []Seen { return seen }, Watched: func(machine.Ref) bool { return false },
		History: func() *activity.History { return h }, Tell: func(string, string, string) {},
	}
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		defer close(done)
		s.Run(ctx)
	}()
	guests.waiting.Wait()
	cancel()
	<-done
	if guests.most != 20 {
		t.Errorf("at most %d of the 20 hung guests were waited on at once; want all 20, one budget for the tick", guests.most)
	}
	if _, ok := h.Of("mac"); !ok {
		t.Error("This Mac was not sampled in the tick its VMs hung")
	}
}

type patientMeter struct{ silentMeter }

func (patientMeter) Process(ctx context.Context, _ int) (activity.Usage, error) {
	if err := ctx.Err(); err != nil {
		return activity.Usage{}, err
	}
	return activity.Usage{CPU: 0.18, Memory: 15 << 30}, nil
}

func TestAVMWhoseGuestHangsIsMeasuredOnItsMacInTheSameTick(t *testing.T) {
	c := &clock{now: time.Date(2026, 11, 4, 10, 0, 0, 0, time.UTC)}
	h := activity.NewHistory(c.now)
	guests := &hung{machines: machines{clock: c, asked: map[string]int{}}}
	guests.waiting.Add(1)
	s := &Sample{
		Calls: guests, Meter: patientMeter{}, Clock: c,
		Seen: func() []Seen {
			return []Seen{{Ref: machine.Ref{Name: "infra", On: "mac"}, Kind: machine.Server, PID: 4242, Guest: true}}
		},
		Watched: func(machine.Ref) bool { return false },
		History: func() *activity.History { return h }, Tell: func(string, string, string) {},
	}
	s.Run(t.Context())
	kept, _ := h.Of("infra")
	if kept.Last.Run == nil || kept.Last.Run.Memory != 15<<30 {
		t.Errorf("after a tick its guest hung through, infra's last reading is %+v; want its process's usage on its Mac", kept.Last)
	}
}

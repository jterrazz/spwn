package jobs

import (
	"context"
	"sync"
	"time"

	"github.com/jterrazz/studio-station/src/activity"
	activitycalls "github.com/jterrazz/studio-station/src/activity/calls"
	"github.com/jterrazz/studio-station/src/activity/contract"
	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/core/finding"
	"github.com/jterrazz/studio-station/src/core/machine"
)

// value: what a sample may take: a guest or a Meter that does not answer
// in it is a sample missed, never a wait.
const budget = 2 * time.Second

type Seen struct {
	Ref  machine.Ref
	Kind machine.Kind
	// value: a VM's run on this Mac: its process, and whether station's
	// guest answers inside it.
	PID   int
	Guest bool
}

type Clock interface{ Now() time.Time }

type Sample struct {
	// value: what reaches activity.sample on a machine: this Mac's own, a
	// VM's over its guest connection.
	Calls call.Router
	// value: this Mac's Meter, which reads a VM's process while its guest
	// is silent.
	Meter contract.Meter
	// value: this Mac first, then each VM of it that runs.
	Seen func() []Seen
	// value: a surface watches this machine now (SPEC §6.3).
	Watched func(m machine.Ref) bool
	Clock   Clock
	History func() *activity.History
	// value: tells the owner a finding that broke, once until it clears:
	// the machine, the finding's subject, its words.
	Tell func(machine, finding, text string)

	mu   sync.Mutex
	last map[string]time.Time
}

func Every(watched bool) time.Duration {
	// invariant: SPEC §6.3: every 2 s while a surface watches, every 60 s
	// otherwise.
	if watched {
		return activity.FineEvery
	}
	return activity.CoarseEvery
}

func (s *Sample) due(h *activity.History, name string, now time.Time, every time.Duration) bool {
	// invariant: a machine is sampled once its last point of the series, or
	// its last try, is every old: an engine that starts again goes on from its
	// history.
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.last == nil {
		s.last = map[string]time.Time{}
	}
	last := s.last[name]
	if kept, ok := h.Of(name); ok {
		if at, has := kept.Series.Last(); has && at.At.After(last) {
			last = at.At
		}
	}
	if !last.IsZero() && now.Sub(last) < every {
		return false
	}
	s.last[name] = now
	return true
}

// invariant: a VM tried while its guest was not connected gave the series no
// point: it is tried again a fine step later, so the guest that connects
// meanwhile is sampled then, not a whole window later.
func (s *Sample) retrySoon(name string, now time.Time, every time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.last[name] = now.Add(activity.FineEvery - every)
}

func (s *Sample) Run(ctx context.Context) {
	h := s.History()
	if h == nil {
		return
	}
	now := s.Clock.Now()
	seen := s.Seen()
	type read struct {
		r     activity.Reading
		ok    bool
		tried bool
	}
	reads := make([]read, len(seen))
	var all sync.WaitGroup
	for i, m := range seen {
		if !s.due(h, m.Ref.Name, now, Every(s.Watched(m.Ref))) {
			continue
		}
		all.Go(func() {
			r, ok := s.read(ctx, m, now)
			reads[i] = read{r, ok, true}
		})
	}
	// invariant: the machines are read together, so a tick costs one
	// budget however many guests hang; what they answer is kept in order.
	all.Wait()
	for i, m := range seen {
		r, ok := reads[i].r, reads[i].ok
		if reads[i].tried && m.Ref.On != "" && !m.Guest {
			s.retrySoon(m.Ref.Name, now, Every(s.Watched(m.Ref)))
		}
		if !ok {
			continue
		}
		h.Add(m.Ref.Name, r)
		kept, _ := h.Of(m.Ref.Name)
		s.judge(h, m, activity.Findings(m.Ref, m.Kind, &kept.Series), r.Sample)
	}
}

func (s *Sample) read(ctx context.Context, m Seen, now time.Time) (activity.Reading, bool) {
	// invariant: a guest that hangs spends its own budget, and its run is
	// still measured on this Mac in the same tick, with a budget of its own.
	if m.Ref.On == "" || m.Guest {
		r, err := s.guest(ctx, m.Ref)
		if err == nil {
			return r, true
		}
	}
	if m.PID == 0 || ctx.Err() != nil {
		return activity.Reading{}, false
	}
	ctx, cancel := context.WithTimeout(ctx, budget)
	defer cancel()
	u, err := s.Meter.Process(ctx, m.PID)
	if err != nil {
		return activity.Reading{}, false
	}
	return activity.Reading{Sample: activity.Sample{At: now.UTC(), Thermal: activity.ThermalUnknown}, Top: activity.Busiest{CPU: []activity.Process{}, Memory: []activity.Process{}}, Run: &u}, true
}

func (s *Sample) guest(ctx context.Context, ref machine.Ref) (activity.Reading, error) {
	ctx, cancel := context.WithTimeout(ctx, budget)
	defer cancel()
	return call.Invoke(ctx, s.Calls, ref, activitycalls.Sample, activitycalls.SampleIn{}, nil)
}

func (s *Sample) judge(h *activity.History, m Seen, found []finding.Finding, last activity.Sample) {
	var broken []string
	bySubject := map[string]finding.Finding{}
	for _, f := range found {
		if f.Severity == finding.Broken {
			broken = append(broken, f.Subject)
			bySubject[f.Subject] = f
		}
	}
	for _, subject := range h.Raise(m.Ref.Name, broken) {
		s.Tell(m.Ref.Name, subject, told(m, bySubject[subject], last))
	}
}

func told(m Seen, f finding.Finding, last activity.Sample) string {
	name := m.Ref.Name
	if m.Ref.On == "" {
		name = "This Mac"
	}
	if f.Subject == activity.SubjectDisk && last.DiskTotal > 0 {
		return activity.DiskFreeTold(name, last)
	}
	return name + ": " + f.Text
}

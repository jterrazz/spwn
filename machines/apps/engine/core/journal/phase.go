package journal

import (
	"context"
	"sync"
	"time"
)

type Phase struct {
	// value: the step's name, in the engine's words ("clone", "Packages").
	Name string
	// value: the Mac's clock when the step began.
	At time.Time
	// value: how long the step ran: to the next step's beginning, the last
	// to the end of the act.
	Took time.Duration
}

type recording struct {
	mu    sync.Mutex
	marks []Phase
}

type phasesKey struct{}

func Recording(ctx context.Context) context.Context {
	// invariant: an act that declares its lines runs on a context that
	// records its steps, once: a gesture it runs in turn marks on the same
	// record.
	if ctx.Value(phasesKey{}) != nil {
		return ctx
	}
	return context.WithValue(ctx, phasesKey{}, &recording{})
}

func Mark(ctx context.Context, at time.Time, name string) {
	// invariant: a step marked begins at the instant given and ends the one
	// before it; a step named as the one before it is the same step, and a
	// context that records nothing is left as it is.
	p, _ := ctx.Value(phasesKey{}).(*recording)
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if n := len(p.marks); n > 0 && p.marks[n-1].Name == name {
		return
	}
	p.marks = append(p.marks, Phase{Name: name, At: at})
}

func Recorded(ctx context.Context, end time.Time) []Phase {
	// value: the steps an act marked, in order, each with how long it took,
	// the last ending at end; none when nothing was marked.
	p, _ := ctx.Value(phasesKey{}).(*recording)
	if p == nil {
		return nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.marks) == 0 {
		return nil
	}
	out := make([]Phase, len(p.marks))
	for i, m := range p.marks {
		next := end
		if i+1 < len(p.marks) {
			next = p.marks[i+1].At
		}
		m.Took = max(next.Sub(m.At), 0)
		out[i] = m
	}
	return out
}

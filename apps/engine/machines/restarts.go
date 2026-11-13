package machines

import (
	"strconv"
	"time"
)

// value: SPEC §7.4: the waits before a server's restarts, the run that
// counts as steady, and the loop.
func restartWaits() [5]time.Duration {
	return [5]time.Duration{5 * time.Second, 10 * time.Second, 20 * time.Second, 40 * time.Second, 60 * time.Second}
}

const (
	// value: the detail of a server something inside powered off.
	PoweredOffInside = "powered off from inside"
	SteadyRun        = 10 * time.Minute
	LoopWindow       = 10 * time.Minute
	LoopBudget       = 5
)

type Restarts struct {
	// value: the restarts since its Mac's engine started, the last one, and
	// the wait the last one took.
	Count int           `json:"count,omitempty"`
	Last  time.Time     `json:"last,omitzero"`
	Wait  time.Duration `json:"wait,omitempty"`
	// value: the restarts after an unexpected end within the loop window.
	Recent []time.Time `json:"recent,omitempty"`
	// value: the unexpected ends since a run last stayed up SteadyRun.
	Streak  int  `json:"streak,omitempty"`
	Looping bool `json:"looping,omitempty"`
}

func (r Restarts) After(up time.Duration, now time.Time) Restarts {
	// invariant: a server whose run ended unexpectedly waits 5 s, 10 s,
	// 20 s, 40 s, then 60 s each time before it boots again; a run that
	// stayed up 10 min sends it back to the first wait; the fifth restart
	// within 10 min is a loop, which keeps trying every 60 s (SPEC §7.4).
	if up >= SteadyRun {
		r.Streak = 0
	}
	recent := make([]time.Time, 0, len(r.Recent)+1)
	for _, at := range r.Recent {
		if now.Sub(at) < LoopWindow {
			recent = append(recent, at)
		}
	}
	recent = append(recent, now)
	r.Recent = recent
	waits := restartWaits()
	r.Wait = waits[min(r.Streak, len(waits)-1)]
	r.Streak++
	r.Looping = len(r.Recent) >= LoopBudget
	if r.Looping {
		r.Wait = waits[len(waits)-1]
	}
	r.Count++
	r.Last = now
	return r
}

func (r Restarts) Rebooted(now time.Time) Restarts {
	// invariant: a reboot from inside boots again at once, counted, with no
	// wait and no step of the backoff.
	r.Count++
	r.Last, r.Wait = now, 0
	return r
}

func (r Restarts) Steady(up time.Duration, now time.Time) Restarts {
	// invariant: a loop ends once its window holds fewer than five restarts
	// or a run stayed up 10 min.
	if up >= SteadyRun {
		r.Streak, r.Recent = 0, nil
	}
	recent := r.Recent[:0:0]
	for _, at := range r.Recent {
		if now.Sub(at) < LoopWindow {
			recent = append(recent, at)
		}
	}
	r.Recent = recent
	r.Looping = r.Looping && len(recent) >= LoopBudget
	return r
}

func (r Restarts) Words(now time.Time) []string {
	// value: SPEC §7.4's count, in parts: `↺ 0`, `none since boot`; `↺ 2`,
	// `last 3 h ago`; a loop as `↺ 5 in 10 min`, `waiting 60 s`.
	switch {
	case r.Count == 0:
		return []string{"↺ 0", "none since boot"}
	case r.Looping:
		return []string{r.InALoop(), "waiting " + strconv.Itoa(int(r.Wait.Seconds())) + " s"}
	}
	return []string{"↺ " + strconv.Itoa(r.Count), "last " + ago(now.Sub(r.Last))}
}

func (r Restarts) InALoop() string {
	return "↺ " + strconv.Itoa(len(r.Recent)) + " in " + strconv.Itoa(int(LoopWindow.Minutes())) + " min"
}

func ago(d time.Duration) string {
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return strconv.Itoa(int(d.Minutes())) + " min ago"
	case d < 48*time.Hour:
		return strconv.Itoa(int(d.Hours())) + " h ago"
	}
	return strconv.Itoa(int(d.Hours()/24)) + " d ago"
}

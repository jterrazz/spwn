package activity

import (
	"time"

	"github.com/jterrazz/studio-station/src/core/finding"
)

type Source string

const (
	// value: the history of the engine that keeps the machine's activity.
	FromEngine Source = "engine"
	// value: two samples 1 s apart, read now: the engine keeps none.
	FromDirect Source = "direct"
	// value: a VM read from its Mac while its guest does not answer: its
	// process there, nothing of its own counters.
	FromMacProcess Source = "mac-process"
	// value: the engine of this Mac keeps no history of the machine asked.
	NotKept Source = ""
)

const (
	// value: the window a rate is the mean of when the engine holds
	// history (SPEC §6.2), to its 2 s cadence: what a page says; a direct
	// read's two samples are DirectGap apart.
	RateWindow = 10 * time.Second
	DirectGap  = time.Second
	// value: what a read waits for a Meter before it says it did not answer.
	Budget = 2 * time.Second
	// value: what a reader waits on a read inside a VM, its two samples
	// and the way in, before it shows the VM's run from its Mac.
	Inside = 3*Budget + DirectGap
)

type Report struct {
	Source Source `json:"source"`
	// value: when the engine that keeps the history started; zero on a
	// direct read.
	Since time.Time `json:"since,omitzero"`
	Last  Reading   `json:"last"`
	// value: the means over Window; Window zero when no two points span one.
	Rates  Rates         `json:"rates"`
	Window time.Duration `json:"window"`
	// value: the Meter did not answer in Budget: no value is known.
	Silent bool `json:"silent,omitempty"`
	// value: the coarse series, when the read asked for the history.
	History []Sample `json:"history,omitempty"`
	// value: the points of the window a screen's graphs draw, when the read
	// asked for one.
	Graph []Sample `json:"graph,omitempty"`
	// value: the window's CPU as its chart draws it (SPEC B5.3), when the
	// read asked for a window.
	Chart *Chart `json:"chart,omitempty"`
	// invariant: never nil: SPEC §6.5's findings over the series read.
	Findings []finding.Finding `json:"findings"`
}

func SeriesOf(fine, coarse []Sample) Series {
	// value: the series a fine and a coarse window hold, oldest first, as
	// the engine hands them over its API.
	var s Series
	for _, x := range fine {
		s.fine.add(x, FinePoints)
	}
	for _, x := range coarse {
		s.coarse.add(x, CoarsePoints)
	}
	return s
}

func Recent(s *Series, now time.Time) (Rates, time.Duration) {
	// invariant: SPEC §6.2: the mean of the last RateWindow, from the 2 s
	// points of an engine a surface watches: the newest no older than
	// FineEvery and a second of jitter, its base the point nearest
	// RateWindow before it, within FineEvery under and twice over (a sample
	// a cadence skipped). Points a minute apart, or a newest gone stale, span
	// no such window: no rate, and the reader samples twice itself.
	fine := s.Fine()
	if len(fine) < 2 {
		return Rates{}, 0
	}
	newest := fine[len(fine)-1]
	if now.Sub(newest.At) > FineEvery+time.Second {
		return Rates{}, 0
	}
	var base *Sample
	for i := len(fine) - 2; i >= 0; i-- {
		gap := newest.At.Sub(fine[i].At)
		if gap < RateWindow-FineEvery {
			continue
		}
		if gap >= RateWindow+2*FineEvery {
			break
		}
		if base == nil || (gap-RateWindow).Abs() < (newest.At.Sub(base.At)-RateWindow).Abs() {
			base = &fine[i]
		}
	}
	if base == nil {
		return Rates{}, 0
	}
	return Between(*base, newest), newest.At.Sub(base.At)
}

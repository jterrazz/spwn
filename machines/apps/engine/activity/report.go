package activity

import (
	"cmp"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/core/finding"
	"github.com/jterrazz/studio-station/apps/engine/core/verdict"
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
	// invariant: the head of the page: Nothing to look at, or the first
	// finding's text under its subject.
	Verdict verdict.Verdict `json:"verdict"`
	// value: when the engine that keeps the history started; zero on a
	// direct read.
	Since time.Time `json:"since,omitzero"`
	Last  Reading   `json:"last"`
	// value: the means over Window; Window zero when no two points span one.
	Rates  Rates         `json:"rates"`
	Window time.Duration `json:"window"`
	// value: the Meter did not answer in Budget: no value is known.
	Silent bool `json:"silent,omitempty"`
	// value: the last 24 h a minute apart, when the read asked for the
	// history (Series.History).
	History []Sample `json:"history,omitempty"`
	// value: the points of the window a screen's graphs draw, when the read
	// asked for one.
	Graph []Sample `json:"graph,omitempty"`
	// value: the window's CPU as its chart draws it (SPEC B5.3), when the
	// read asked for a window.
	Chart *Chart `json:"chart,omitempty"`
	// value: the window's other measures in the same bars, when the read
	// asked for a window.
	Measures *Measures `json:"measures,omitempty"`
	// invariant: never nil: SPEC §6.5's findings over the series read.
	Findings []finding.Finding `json:"findings"`
}

const (
	SentenceCalm   = "Nothing to look at."
	SentenceSilent = "No answer in 2 s."
	LineKept       = "history kept 24 h, since the engine started"
)

func VerdictOf(r Report) verdict.Verdict {
	// invariant: a silent Meter is said so, never calm; the first finding
	// of the ranked list leads otherwise; the line says the history is kept
	// only when an engine keeps it.
	switch {
	case r.Silent:
		return verdict.Verdict{Tone: verdict.Warn, Sentence: SentenceSilent}
	case len(r.Findings) > 0:
		f := r.Findings[0]
		tone := verdict.Warn
		if f.Severity == finding.Broken {
			tone = verdict.Bad
		}
		return verdict.Verdict{Tone: tone, Sentence: f.Text, Line: cmp.Or(f.Detail, f.Subject)}
	}
	v := verdict.Verdict{Tone: verdict.OK, Sentence: SentenceCalm}
	if !r.Since.IsZero() {
		v.Line = LineKept
	}
	return v
}

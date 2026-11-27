package calls

import (
	"context"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/activity"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/finding"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/core/section"
)

type ReadIn struct {
	// value: a VM of this Mac whose history its engine keeps, by name;
	// empty reads this machine.
	Of string `json:"of,omitempty"`
	// value: the machine as the reader names it, and its kind: what the
	// findings' fixes say; empty is this machine.
	Machine machine.Ref  `json:"machine,omitzero"`
	Kind    machine.Kind `json:"kind,omitempty"`
	// value: the last 24 h the engine keeps, a minute apart, for scripts.
	History bool `json:"history,omitempty"`
	// value: the window a screen's graphs draw (6 min, 1 h, 24 h): the
	// points of it the engine keeps.
	Graph time.Duration `json:"graph,omitempty"`
	// value: what the engine keeps, else one sample now: a read that never
	// waits (an Overview's line).
	Quick bool `json:"quick,omitempty"`
	// value: the VM's run as this Mac measures it now, for a reader whose
	// read inside the VM failed: its guest does not answer.
	Run bool `json:"run,omitempty"`
}

const (
	// value: the widest window a graph is drawn over: a wider one would
	// fold into more buckets than any screen draws.
	MaxGraph = 30 * 24 * time.Hour
)

var Read = call.Watch[Deps, ReadIn, activity.Report]{
	Name: "activity.read", Section: section.Activity, Where: call.AtTarget, Applies: activity.Applies,
	Errors: []fail.Code{call.TooLarge},
	Run: func(ctx context.Context, d Deps, in ReadIn, _ call.Progress) (<-chan call.Update[activity.Report], error) {
		if in.Graph > MaxGraph {
			return nil, call.OverBound("activity.read", "graph", "30 days")
		}
		out := make(chan activity.Report, 1)
		r := d.read(ctx, in)
		r.Verdict = activity.VerdictOf(r)
		out <- r
		close(out)
		return call.Updates(ctx, out), nil
	},
}

func (d Deps) read(ctx context.Context, in ReadIn) activity.Report {
	// invariant: SPEC §6.2: the history of the engine that keeps it when it
	// holds the last 10 s; else, on the machine itself, two samples 1 s
	// apart. A VM this Mac's engine holds no such window of is NotKept, with
	// the series it keeps and the findings over it: its reader asks the VM
	// for the values.
	ref, kind, name := in.Machine, in.Kind, in.Of
	if ref.Name == "" {
		ref.Name = d.Name
	}
	if kind == "" {
		kind = d.Kind
	}
	if name == "" {
		name = d.Name
	}
	var kept *activity.Kept
	var since time.Time
	if d.Kept != nil {
		if k, at, ok, err := d.Kept(ctx, name, activity.Want{Graph: in.Graph, History: in.History}); err == nil && ok {
			if r, usable := fromHistory(k, at, in, d.Now()); usable {
				r.Findings = findings(ref, kind, &k.Series, k.Last.Top)
				return graphed(r, &k.Series, in, d.Now())
			}
			kept, since = &k, at
		}
	}
	r := activity.Report{Source: activity.NotKept, Findings: []finding.Finding{}}
	switch {
	case in.Of == "":
		r = d.direct(ctx, ref, kind, in.Quick, kept)
	case in.Run:
		r = d.run(ctx, in.Of)
		if kept != nil {
			r.Findings = findings(ref, kind, &kept.Series, kept.Last.Top)
		}
	case kept != nil:
		r.Findings = findings(ref, kind, &kept.Series, kept.Last.Top)
	}
	if kept != nil {
		r.Since = since
		if in.History {
			r.History = kept.Series.History(d.Now())
		}
		r = graphed(r, &kept.Series, in, d.Now())
	}
	return r
}

func graphed(r activity.Report, s *activity.Series, in ReadIn, now time.Time) activity.Report {
	// invariant: ADR-107: every measure of a window is read from the one
	// tier TierOf picks for it, each bar its mean and its busiest.
	if in.Graph > 0 {
		r.Graph = s.Graph(in.Graph, now)
		chart, measures := s.Chart(in.Graph, now), s.Measures(in.Graph, now)
		r.Chart, r.Measures = &chart, &measures
	}
	return r
}

func fromHistory(k activity.Kept, since time.Time, in ReadIn, now time.Time) (activity.Report, bool) {
	r := activity.Report{Source: activity.FromEngine, Since: since, Last: k.Last}
	if k.Last.Run != nil {
		r.Source = activity.FromMacProcess
		return r, true
	}
	r.Rates, r.Window = activity.Recent(&k.Series, now)
	if in.History {
		r.History = k.Series.History(now)
	}
	return r, r.Window > 0
}

func (d Deps) direct(ctx context.Context, ref machine.Ref, kind machine.Kind, quick bool, kept *activity.Kept) activity.Report {
	// invariant: the findings judge the engine's series, when it keeps one,
	// with the samples read now: a CPU busy for 10 min stays told.
	r := activity.Report{Source: activity.FromDirect, Findings: []finding.Finding{}}
	first, err := d.budgeted(ctx)
	if err != nil {
		r.Silent = true
		return r
	}
	r.Last = first
	points := []activity.Sample{first.Sample}
	if !quick && d.Sleep != nil && d.Sleep(ctx, activity.DirectGap) == nil {
		second, err := d.budgeted(ctx)
		if err != nil {
			r.Silent = true
			return r
		}
		r.Last = second
		points = append(points, second.Sample)
		r.Rates, r.Window = activity.Between(first.Sample, second.Sample), second.Sample.At.Sub(first.Sample.At)
	}
	var s activity.Series
	if kept != nil {
		s = kept.Series.Clone()
	}
	for _, x := range points {
		s.Add(x)
	}
	r.Findings = findings(ref, kind, &s, r.Last.Top)
	return r
}

func (d Deps) run(ctx context.Context, vm string) activity.Report {
	r := activity.Report{Source: activity.NotKept, Findings: []finding.Finding{}}
	if d.Runs == nil || d.Now == nil {
		return r
	}
	runs, err := d.Runs(ctx)
	if err != nil {
		return r
	}
	pid := 0
	for _, run := range runs {
		if run.VM == vm {
			pid = run.Holder
		}
	}
	if pid == 0 {
		return r
	}
	ctx, cancel := context.WithTimeout(ctx, activity.Budget)
	defer cancel()
	u, err := d.Meter.Process(ctx, pid)
	if err != nil {
		return r
	}
	r.Source = activity.FromMacProcess
	r.Last = activity.Reading{
		Sample: activity.Sample{At: d.Now().UTC(), Thermal: activity.ThermalUnknown},
		Top:    activity.Busiest{CPU: []activity.Process{}, Memory: []activity.Process{}},
		Run:    &u,
	}
	return r
}

func (d Deps) budgeted(ctx context.Context) (activity.Reading, error) {
	ctx, cancel := context.WithTimeout(ctx, activity.Budget)
	defer cancel()
	return compose(ctx, d)
}

func findings(ref machine.Ref, kind machine.Kind, s *activity.Series, top activity.Busiest) []finding.Finding {
	out := activity.Findings(ref, kind, s, top.CPU)
	if out == nil {
		return []finding.Finding{}
	}
	return out
}

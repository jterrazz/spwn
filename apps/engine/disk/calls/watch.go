package calls

import (
	"context"
	"maps"
	"path"
	"slices"
	"sync"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/call/lanes"
	"github.com/jterrazz/studio-station/apps/engine/disk"
)

const (
	subject     = "disk"
	pending     = int64(-2)
	globWorkers = 8
)

type part struct {
	root int
	sign int64
}

type pathPlan struct {
	report disk.PathReport
	parts  []part
}

type misplacedPlan struct {
	report disk.MisplacedReport
	root   int
}

type entryPlan struct {
	entry         disk.Entry
	paths         []pathPlan
	install       []int
	misplacedOnes []misplacedPlan
	containers    bool
	exists        bool
}

type plan struct {
	entries []entryPlan
	roots   []string
	other   []int
}

type reading struct {
	sizes      []int64
	known      []int64
	installed  map[string]bool
	listed     bool
	containers int64
	// value: no walk has answered on this machine yet, so no row has a
	// last size to show.
	fresh bool
}

func (s *Surveyor) Watch(ctx context.Context) <-chan disk.Survey {
	return lanes.Stream(ctx, s.o.Pool, subject, s.watch)
}

func (s *Surveyor) Forget() { s.o.Pool.Forget(subject) }

// invariant: the frame is every placed entry whose declared paths exist, read
// with a glob and a stat each; one walk then sizes every root of every row and
// the other caches together, and a row's size fills when its last root
// answers, its mark when its own tool answers the inventory.
func (s *Surveyor) watch(ctx context.Context) <-chan disk.Survey {
	p := s.plan()
	frame := reading{sizes: make([]int64, len(p.roots)), known: make([]int64, len(p.roots)), containers: pending}
	known := s.loadKnown()
	frame.fresh = len(known) == 0
	for i, root := range p.roots {
		frame.sizes[i], frame.known[i] = pending, pending
		if k, ok := known[root]; ok {
			frame.known[i] = k.Bytes
		}
	}
	budget := s.walkBudget()
	jobs := []lanes.Job[reading]{
		{
			Key: subject + "/walk", Lane: lanes.Walks, Budget: budget,
			Partial: func(ctx context.Context, emit func(func(*reading))) func(*reading) {
				var mu sync.Mutex
				answered := slices.Repeat([]int64{unmeasured}, len(p.roots))
				s.o.Disk.Sizes(ctx, p.roots, func(i int, bytes int64) {
					mu.Lock()
					answered[i] = bytes
					mu.Unlock()
					emit(func(r *reading) { r.sizes[i] = bytes })
				})
				mu.Lock()
				defer mu.Unlock()
				s.saveKnown(p.roots, answered)
				return func(*reading) {}
			},
			Late: func(r *reading) {
				for i, b := range r.sizes {
					if b == pending {
						r.sizes[i] = unmeasured
					}
				}
			},
		},
		{
			Key: subject + "/installed", Lane: lanes.Inline,
			Partial: func(ctx context.Context, emit func(func(*reading))) func(*reading) {
				s.eachInstalled(ctx, func(installed map[string]bool) {
					emit(func(r *reading) { r.installed = installed })
				})
				return func(r *reading) { r.listed = true }
			},
			Late: func(r *reading) { r.listed = true },
		},
		{
			Key: subject + "/containers", Lane: lanes.Inline,
			Do: func(ctx context.Context) func(*reading) {
				bytes := int64(0)
				if name := containersEntry(p); name != "" && s.isInstalled(ctx, name) {
					bytes = s.containerBytes(ctx)
				}
				return func(r *reading) { r.containers = bytes }
			},
			Late: func(r *reading) { r.containers = unmeasured },
		},
	}
	readings := lanes.Run(ctx, s.o.Pool, frame, jobs, func(r reading) reading {
		return reading{sizes: slices.Clone(r.sizes), known: r.known, fresh: r.fresh, installed: maps.Clone(r.installed), listed: r.listed, containers: r.containers}
	})
	out := make(chan disk.Survey)
	go func() {
		defer close(out)
		for r := range readings {
			select {
			case out <- s.assemble(p, r):
			case <-ctx.Done():
				return
			}
		}
	}()
	return out
}

func containersEntry(p plan) string {
	for _, e := range p.entries {
		if e.containers {
			return e.entry.Name
		}
	}
	return ""
}

func (s *Surveyor) plan() plan {
	var p plan
	index := map[string]int{}
	root := func(abs string) int {
		if i, ok := index[abs]; ok {
			return i
		}
		index[abs] = len(p.roots)
		p.roots = append(p.roots, abs)
		return index[abs]
	}
	placed := s.Placed()
	globbed := s.globAll(placed)
	glob := func(pattern string) []string { return globbed[s.o.Expand.Expand(pattern)] }
	for _, e := range placed {
		ep := entryPlan{entry: e, containers: e.Disk.Size == catalog.SizeContainers}
		for _, st := range e.Disk.Paths {
			if !st.PlacedOn(s.o.ThisMachine) {
				continue
			}
			pp := pathPlan{report: disk.PathReport{Path: s.o.Expand.Display(s.o.Expand.Expand(st.Path)), Level: st.Level}}
			for _, m := range glob(st.Path) {
				pp.parts = append(pp.parts, part{root: root(m), sign: 1})
			}
			for _, memo := range st.Memos {
				for _, m := range glob(path.Join(st.Path, memo)) {
					pp.parts = append(pp.parts, part{root: root(m), sign: -1})
				}
			}
			ep.exists = ep.exists || len(pp.parts) > 0
			ep.paths = append(ep.paths, pp)
		}
		for _, pattern := range e.Disk.Installed {
			for _, m := range glob(pattern) {
				ep.install = append(ep.install, root(m))
			}
		}
		ep.exists = ep.exists || len(ep.install) > 0
		for _, st := range e.Disk.Misplaced {
			abs := s.o.Expand.Expand(st.Path)
			if !s.m.Exists(abs) {
				continue
			}
			ep.misplacedOnes = append(ep.misplacedOnes, misplacedPlan{report: s.misplaced(e, st, abs, 0), root: root(abs)})
		}
		ep.exists = ep.exists || len(ep.misplacedOnes) > 0
		p.entries = append(p.entries, ep)
	}
	for _, other := range s.otherRoots() {
		p.other = append(p.other, root(other))
	}
	return p
}

// value: the frame's globs are independent reads of the disk, so they run at
// once, a few at a time, and the frame costs its slowest, not their sum.
func (s *Surveyor) globAll(placed []disk.Entry) map[string][]string {
	var patterns []string
	for _, e := range placed {
		for _, st := range e.Disk.Paths {
			if !st.PlacedOn(s.o.ThisMachine) {
				continue
			}
			patterns = append(patterns, s.o.Expand.Expand(st.Path))
			for _, memo := range st.Memos {
				patterns = append(patterns, s.o.Expand.Expand(path.Join(st.Path, memo)))
			}
		}
		for _, pattern := range e.Disk.Installed {
			patterns = append(patterns, s.o.Expand.Expand(pattern))
		}
	}
	out := make(map[string][]string, len(patterns))
	var mu sync.Mutex
	var wg sync.WaitGroup
	work := make(chan string)
	for range globWorkers {
		wg.Go(func() {
			for pattern := range work {
				matches := s.m.Glob(pattern)
				mu.Lock()
				out[pattern] = matches
				mu.Unlock()
			}
		})
	}
	for _, pattern := range patterns {
		work <- pattern
	}
	close(work)
	wg.Wait()
	return out
}

func (s *Surveyor) assemble(p plan, r reading) disk.Survey {
	report := disk.Survey{Rows: []disk.Tally{}, Misplaced: []disk.MisplacedReport{}}
	for _, ep := range p.entries {
		row, show := s.row(ep, r)
		if !show {
			continue
		}
		report.Rows = append(report.Rows, row)
		report.Misplaced = append(report.Misplaced, row.Misplaced...)
	}
	for _, i := range p.other {
		if b := r.sizes[i]; b > 0 {
			report.Other += b
		}
	}
	return report
}

func (s *Surveyor) row(ep entryPlan, r reading) (disk.Tally, bool) {
	e := ep.entry
	installed := r.installed[e.Name] || e.Name == disk.StationName
	_, answered := r.installed[e.Name]
	marked := r.listed || answered || e.Name == disk.StationName
	waiting, containersWait := false, false
	var within time.Duration
	value := func(i int) int64 {
		switch b := r.sizes[i]; b {
		case pending:
			waiting = true
			return max(r.known[i], 0)
		case unmeasured:
			within = max(within, s.walkBudget())
			return 0
		default:
			return b
		}
	}
	row := disk.Tally{Entry: e, Installed: installed, Bytes: map[catalog.Level]int64{}, Paths: []disk.PathReport{}, Misplaced: []disk.MisplacedReport{}}
	for _, pp := range ep.paths {
		report := pp.report
		for _, pt := range pp.parts {
			report.Bytes += pt.sign * value(pt.root)
		}
		row.Bytes[pp.report.Level] += report.Bytes
		row.Paths = append(row.Paths, report)
	}
	for _, i := range ep.install {
		row.Install += value(i)
	}
	for _, st := range ep.misplacedOnes {
		report := st.report
		report.Bytes = value(st.root)
		row.Misplaced = append(row.Misplaced, report)
	}
	if ep.containers && installed {
		switch r.containers {
		case pending:
			waiting, containersWait = true, true
		case unmeasured:
			within = max(within, containersBudget)
		default:
			row.Bytes[catalog.LevelCache] += r.containers
		}
	}
	switch {
	case len(row.Misplaced) > 0:
		row.State = catalog.StateDegraded
	case installed:
		row.State = catalog.StateOK
	case !marked:
		row.State = catalog.StatePending
	default:
		row.State = catalog.StateAbsent
	}
	if within > 0 && !waiting {
		unknown := s.unknown(e, installed, within)
		if !marked {
			unknown.State = catalog.StatePending
		}
		return unknown, true
	}
	if waiting {
		row.Pending, row.Unmeasured = true, containersWait || r.fresh
		return row, e.Name == disk.StationName || ep.exists || (ep.containers && installed)
	}
	return row, e.Name == disk.StationName || row.Total() != 0
}

func (s *Surveyor) walkBudget() time.Duration {
	if s.o.Budget > 0 {
		return s.o.Budget
	}
	return sizeBudget
}

func (s *Surveyor) unknown(e disk.Entry, installed bool, within time.Duration) disk.Tally {
	row := disk.Tally{Entry: e, Installed: installed, Unknown: true, Within: within, Bytes: map[catalog.Level]int64{}, Install: unmeasured, Paths: []disk.PathReport{}, Misplaced: []disk.MisplacedReport{}, State: catalog.StateAbsent}
	if installed {
		row.State = catalog.StateOK
	}
	for _, level := range []catalog.Level{catalog.LevelCache, catalog.LevelData, catalog.LevelSettings, catalog.LevelKept} {
		row.Bytes[level] = unmeasured
	}
	for _, p := range e.Disk.Paths {
		if p.PlacedOn(s.o.ThisMachine) {
			row.Paths = append(row.Paths, disk.PathReport{Path: s.o.Expand.Display(s.o.Expand.Expand(p.Path)), Level: p.Level, Bytes: unmeasured})
		}
	}
	return row
}

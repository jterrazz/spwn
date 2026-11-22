package calls

import (
	"context"
	"maps"
	"path"
	"slices"
	"strings"
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
	systemRoot  = "/usr"
)

// value: the folders macOS keeps apps in, every app in them an app whoever
// installed it.
func appFolders() []string { return []string{"/Applications", "~/Applications"} }

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
	// value: each entry of a macOS app folder that no catalog install
	// names: an app the catalog does not carry.
	apps []int
	// value: the root the system is walked at, -1 where the volume tells it,
	// and the apps' install roots inside it, which are apps, not system.
	system      int
	underSystem []int
}

type reading struct {
	sizes      []int64
	known      []int64
	installed  map[string]bool
	listed     bool
	containers int64
	system     int64
	// value: no walk has answered on this machine yet, so no row has a
	// last size to show.
	fresh bool
}

func (s *Surveyor) Watch(ctx context.Context) <-chan disk.Survey {
	return lanes.Stream(ctx, s.o.Pool, subject, s.watch)
}

func (s *Surveyor) Forget() { s.o.Pool.Forget(subject) }

func (s *Surveyor) Held(ctx context.Context) <-chan disk.HeldReading {
	// invariant: each reader asks the hypervisor itself, never through the
	// survey a pool shares, so a machine or an image made or deleted since
	// the last read is seen; the frame is pending, holding the last answer
	// the pool heard, which stands too when this one fails or is late, so a
	// read asked again never draws less than the last; a machine with no
	// hypervisor to ask answers none at once.
	if s.o.Station == nil {
		out := make(chan disk.HeldReading, 1)
		out <- disk.HeldReading{State: disk.HeldNone}
		close(out)
		return out
	}
	key := subject + "/station"
	frame := disk.HeldReading{State: disk.HeldPending}
	frame.Held, frame.Known = lanes.Remembered[disk.Held](s.o.Pool, key)
	job := lanes.Job[disk.HeldReading]{
		Key: key, Lane: lanes.Commands, Fresh: true,
		Do: func(ctx context.Context) func(*disk.HeldReading) {
			held, err := s.o.Station.Held(ctx)
			if err == nil && held != nil {
				lanes.Remember(s.o.Pool, key, *held)
			}
			return func(r *disk.HeldReading) {
				switch {
				case err != nil:
					r.State = disk.HeldFailed
				case held == nil:
					r.State, r.Held, r.Known = disk.HeldNone, disk.Held{}, false
				default:
					r.State, r.Held, r.Known = disk.HeldAnswered, *held, false
				}
			}
		},
		Late: func(r *disk.HeldReading) { r.State = disk.HeldFailed },
	}
	return lanes.Run(ctx, s.o.Pool, frame, []lanes.Job[disk.HeldReading]{job}, func(r disk.HeldReading) disk.HeldReading { return r })
}

// invariant: the frame is every placed entry whose declared paths exist, read
// with a glob and a stat each; one walk then sizes every root of every row and
// the other caches together, and a row's size fills when its last root
// answers, its mark when its own tool answers the inventory.
func (s *Surveyor) watch(ctx context.Context) <-chan disk.Survey {
	p := s.plan()
	frame := reading{sizes: make([]int64, len(p.roots)), known: make([]int64, len(p.roots)), containers: pending, system: -1}
	if p.system < 0 {
		frame.system = s.o.Disk.System(ctx)
	}
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
		return reading{sizes: slices.Clone(r.sizes), known: r.known, fresh: r.fresh, installed: maps.Clone(r.installed), listed: r.listed, containers: r.containers, system: r.system}
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
	p := plan{system: -1}
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
			if !s.m.Exists(abs) || !s.movable(e, st) {
				continue
			}
			ep.misplacedOnes = append(ep.misplacedOnes, misplacedPlan{report: s.misplaced(e, st, abs, 0), root: root(abs)})
		}
		ep.exists = ep.exists || len(ep.misplacedOnes) > 0
		p.entries = append(p.entries, ep)
	}
	p.apps = s.appRoots(p, root)
	for _, other := range s.otherRoots() {
		p.other = append(p.other, root(other))
	}
	if s.o.ThisMachine == catalog.Linux {
		// value: one volume holds a Linux machine's system and data alike,
		// so its system is what its programs and libraries hold.
		p.system = root(systemRoot)
		seen := map[int]bool{}
		for _, ep := range p.entries {
			for _, i := range ep.install {
				if !seen[i] && strings.HasPrefix(p.roots[i], systemRoot+"/") {
					seen[i] = true
					p.underSystem = append(p.underSystem, i)
				}
			}
		}
	}
	return p
}

func (s *Surveyor) appRoots(p plan, root func(string) int) []int {
	// invariant: on macOS every entry of an app folder is an app, whoever
	// installed it, counted once: one a catalog install names, or holds, is
	// that install's; Linux keeps no app folder.
	if s.o.ThisMachine == catalog.Linux {
		return nil
	}
	var installed []string
	for _, ep := range p.entries {
		for _, i := range ep.install {
			installed = append(installed, p.roots[i])
		}
	}
	var out []int
	for _, folder := range appFolders() {
		dir := s.o.Expand.Expand(folder)
		names, err := s.m.Names(dir)
		if err != nil {
			continue
		}
		for _, name := range names {
			if full := path.Join(dir, name); !claims(installed, full) {
				out = append(out, root(full))
			}
		}
	}
	return out
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
	report := disk.Survey{Rows: []disk.Tally{}, Misplaced: []disk.MisplacedReport{}, Hypervisor: s.o.Hypervisor}
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
	report.System, report.SystemPending = r.system, false
	if p.system >= 0 {
		report.System, report.SystemPending = walked(r, append([]int{p.system}, p.underSystem...))
		if report.System >= 0 {
			report.System = r.sizes[p.system]
			for _, i := range p.underSystem {
				report.System -= r.sizes[i]
			}
			report.System = max(report.System, 0)
		}
	}
	report.Apps, report.CachesUnknown = s.kinds(p, r)
	report.Logs = s.logs(p, r)
	return report
}

// invariant: a size the walk has not answered, pending or past its
// budget, is never read as 0: what it is part of is -1, pending while one
// of its roots still is.
func walked(r reading, roots []int) (bytes int64, waiting bool) {
	for _, i := range roots {
		switch b := r.sizes[i]; {
		case b == pending:
			bytes, waiting = -1, true
		case b < 0:
			bytes = -1
		case bytes >= 0:
			bytes += b
		}
	}
	return bytes, waiting
}

func (s *Surveyor) kinds(p plan, r reading) (apps int64, cachesUnknown bool) {
	seen := map[int]bool{}
	var installs []int
	for _, ep := range p.entries {
		for _, i := range ep.install {
			if !seen[i] {
				seen[i] = true
				installs = append(installs, i)
			}
		}
		for _, pp := range ep.paths {
			if pp.report.Level != catalog.LevelCache {
				continue
			}
			for _, pt := range pp.parts {
				cachesUnknown = cachesUnknown || r.sizes[pt.root] < 0
			}
		}
		cachesUnknown = cachesUnknown || (ep.containers && r.containers < 0)
	}
	for _, i := range p.apps {
		if !seen[i] {
			seen[i] = true
			installs = append(installs, i)
		}
	}
	apps, _ = walked(r, installs)
	return apps, cachesUnknown
}

func (s *Surveyor) logs(p plan, r reading) int64 {
	logs := s.o.Expand.Display(s.o.Expand.Expand(catalog.StationLogs))
	for _, ep := range p.entries {
		if ep.entry.Name != disk.StationName {
			continue
		}
		for _, pp := range ep.paths {
			if pp.report.Path != logs {
				continue
			}
			var bytes int64
			for _, pt := range pp.parts {
				if r.sizes[pt.root] < 0 {
					return -1
				}
				bytes += pt.sign * r.sizes[pt.root]
			}
			return max(bytes, 0)
		}
	}
	return 0
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
		if holdsNothing(r, st.root) {
			continue
		}
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

// invariant: a misplaced folder the walk answers empty, or last answered
// empty while it walks it again, is no misplaced folder: there is nothing to
// move.
func holdsNothing(r reading, i int) bool {
	b := r.sizes[i]
	return b == 0 || (b == pending && r.known[i] == 0)
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

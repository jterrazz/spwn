package calls

import (
	"cmp"
	"context"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/core/call/lanes"
	"github.com/jterrazz/studio-station/src/disk"
	"github.com/jterrazz/studio-station/src/disk/contract"
)

const (
	containersBudget = 5 * time.Second
	sizeBudget       = 10 * time.Second
	commandLane      = 16
	walkLane         = 1
	unmeasured       = int64(-1)
)

type Inventory interface {
	// value: each call holds every tool answered so far, so a row's mark fills
	// as soon as its own tool answers; the last call holds them all.
	Installed(ctx context.Context, each func(map[string]bool))
}

type SurveyorOptions struct {
	// value: this machine's disk: sizes, erases, and which apps run.
	Disk        contract.DiskUsage
	Pool        *lanes.Pool
	ThisMachine catalog.MachineKind
	Expand      disk.Expander
	Files       map[catalog.AppFile]string
	Inventory   Inventory
	Budget      time.Duration
	// value: the space this machine's container runtime holds, or none.
	Containers ContainerSpace
	// value: applies the settings a moved folder needs (tart-launcher).
	Configurer Configurer
}

type Configurer interface {
	Configure(ctx context.Context, item string, interactive bool) error
}

type Surveyor struct {
	m    Machine
	apps []catalog.App
	o    SurveyorOptions
}

func NewSurveyor(m Machine, apps []catalog.App, o SurveyorOptions) *Surveyor {
	if o.Pool == nil {
		o.Pool = lanes.NewPool(context.Background(), m, commandLane, walkLane)
	}
	return &Surveyor{m: m, apps: apps, o: o}
}

func (s *Surveyor) Entries() []disk.Entry {
	everywhere := []catalog.MachineKind{catalog.Mac, catalog.Linux, catalog.MacOS}
	out := []disk.Entry{
		{Name: disk.StationName, Category: disk.StationCategory, Disk: catalog.StationDisk, Kinds: everywhere},
		{Name: catalog.KeysRow, Category: disk.StationCategory, Disk: catalog.SSHDisk, Kinds: everywhere},
	}
	for _, t := range s.apps {
		if t.Disk.Declared() {
			out = append(out, disk.Entry{Name: t.Name, Category: string(t.Category), Disk: t.Disk, Kinds: t.LivesOn()})
		}
	}
	return out
}

func (s *Surveyor) Placed() []disk.Entry {
	var out []disk.Entry
	for _, e := range s.Entries() {
		if s.placed(e) {
			out = append(out, e)
		}
	}
	return out
}

func (s *Surveyor) Names() []string {
	placed := s.Placed()
	out := make([]string, 0, len(placed))
	for _, e := range placed {
		out = append(out, e.Name)
	}
	return out
}

func (s *Surveyor) placed(e disk.Entry) bool {
	for _, k := range e.Kinds {
		if k == s.o.ThisMachine {
			return true
		}
	}
	return false
}

func (s *Surveyor) find(name string) (disk.Entry, error) {
	for _, e := range s.Entries() {
		if e.Name != name {
			continue
		}
		if !s.placed(e) {
			return e, &disk.ElsewhereError{Name: name, Kinds: e.Kinds}
		}
		return e, nil
	}
	return disk.Entry{}, &disk.UnknownError{Name: name}
}

func (s *Surveyor) installed(ctx context.Context) map[string]bool {
	out := map[string]bool{}
	s.eachInstalled(ctx, func(got map[string]bool) { out = got })
	out[disk.StationName] = true
	return out
}

func (s *Surveyor) eachInstalled(ctx context.Context, each func(map[string]bool)) {
	if s.o.Inventory != nil {
		s.o.Inventory.Installed(ctx, each)
	}
}

func (s *Surveyor) isInstalled(ctx context.Context, name string) bool {
	ctx, stop := context.WithCancel(ctx)
	defer stop()
	var installed bool
	s.eachInstalled(ctx, func(got map[string]bool) {
		if v, ok := got[name]; ok {
			installed = v
			stop()
		}
	})
	return installed
}

func (s *Surveyor) List(ctx context.Context) (disk.Survey, error) {
	return lanes.Last(s.Watch(ctx)), ctx.Err()
}

func (s *Surveyor) otherRoots() []string {
	var claimed []string
	for _, e := range s.Placed() {
		for _, p := range e.Disk.Paths {
			claimed = append(claimed, s.o.Expand.Expand(p.Path))
		}
		for _, p := range e.Disk.Installed {
			claimed = append(claimed, s.o.Expand.Expand(p))
		}
	}
	roots := []string{s.o.Expand.Roots.Cache}
	if s.o.ThisMachine != catalog.Linux {
		roots = append(roots, path.Join(s.o.Expand.Home, "Library", "Caches"))
	}
	var out []string
	for _, root := range roots {
		entries, err := s.m.Names(root)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if strings.HasPrefix(entry, "com.apple.") {
				continue
			}
			if full := path.Join(root, entry); !claims(claimed, full) {
				out = append(out, full)
			}
		}
	}
	return out
}

func claims(patterns []string, entry string) bool {
	for _, p := range patterns {
		if p == entry || strings.HasPrefix(p, entry+"/") || strings.HasPrefix(entry, strings.TrimSuffix(p, "/*")+"/") {
			return true
		}
		if ok, _ := path.Match(p, entry); ok {
			return true
		}
	}
	return false
}

func (s *Surveyor) size(pattern string) int64 {
	matches := s.m.Glob(s.o.Expand.Expand(pattern))
	var total int64
	for _, m := range matches {
		total += s.dirSize(m)
	}
	return total
}

func (s *Surveyor) dirSize(p string) int64 {
	var bytes int64
	s.o.Disk.Sizes(context.Background(), []string{p}, func(_ int, b int64) { bytes = b })
	return bytes
}

func (s *Surveyor) exists(pattern string) bool {
	matches := s.m.Glob(s.o.Expand.Expand(pattern))
	return len(matches) > 0
}

func (s *Surveyor) misplacedOnes(e disk.Entry) []disk.MisplacedReport {
	out := []disk.MisplacedReport{}
	for _, st := range e.Disk.Misplaced {
		abs := s.o.Expand.Expand(st.Path)
		if !s.m.Exists(abs) {
			continue
		}
		out = append(out, s.misplaced(e, st, abs, s.dirSize(abs)))
	}
	return out
}

func (s *Surveyor) misplaced(e disk.Entry, st catalog.Misplaced, abs string, bytes int64) disk.MisplacedReport {
	r := disk.MisplacedReport{Name: e.Name, Path: s.o.Expand.Display(abs), Bytes: bytes, Repair: "station disk move " + s.o.Expand.Display(abs)}
	if len(st.Move) > 0 {
		r.To = s.o.Expand.Display(s.o.Expand.Expand(st.Move[0].To))
	}
	return r
}

func (s *Surveyor) Relocated(st catalog.Disk) bool {
	for _, r := range st.Relocate {
		if r.Pending != "" {
			continue
		}
		file := s.o.Files[r.File]
		if file == "" {
			continue
		}
		data, err := s.m.ReadFile(file)
		if err != nil {
			return false
		}
		value := disk.LineValue
		if r.File.Tabled() {
			value = disk.TableValue
		}
		if _, ok := value(string(data), r.Key); !ok {
			return false
		}
	}
	return true
}

func (s *Surveyor) Free(ctx context.Context, names []string) ([]disk.Outcome, error) {
	var entries []disk.Entry
	named := len(names) > 0
	for _, name := range dedupe(names) {
		e, err := s.find(name)
		if err != nil {
			return nil, err
		}
		if len(e.Disk.Free) == 0 && len(e.Disk.Misplaced) == 0 {
			return nil, &disk.NothingToFreeError{Name: name}
		}
		entries = append(entries, e)
	}
	installed := s.installed(ctx)
	if !named {
		// invariant: a bare free empties every Can rebuild row, the rows
		// whose caches hold anything, and only those.
		for _, e := range s.Placed() {
			if len(e.Disk.Free) > 0 && s.levelBytes(e, catalog.LevelCache) > 0 {
				entries = append(entries, e)
			}
		}
	}
	outcomes := make([]disk.Outcome, 0, len(entries))
	for _, e := range entries {
		present := installed[e.Name]
		before := s.cacheBytes(e)
		if !present && before == 0 && len(s.misplacedOnes(e)) == 0 {
			outcomes = append(outcomes, disk.Outcome{Name: e.Name, Result: catalog.StateAbsent, Detail: "not installed"})
			continue
		}
		failure := s.actions(ctx, e.Disk.Free, present)
		if !present && failure == nil {
			failure = s.deleteCaches(e)
		}
		for _, st := range e.Disk.Misplaced {
			if s.m.Exists(s.o.Expand.Expand(st.Path)) {
				failure = first(failure, s.actions(ctx, st.Free, present))
			}
		}
		outcomes = append(outcomes, s.outcome(e.Name, failure, before-s.cacheBytes(e), false))
	}
	return outcomes, nil
}

func (s *Surveyor) CheckErase(name string) (disk.Entry, error) {
	// invariant: an erase names one row whose data station can erase: a row
	// of keys, tokens or settings only is never touched, whatever it is asked.
	e, err := s.find(name)
	if err != nil {
		return e, err
	}
	if len(e.Disk.Erase) > 0 {
		return e, nil
	}
	if kept := s.keptOnly(e); kept != "" {
		return e, &disk.NeverTouchedError{Name: name, Path: kept}
	}
	return e, &disk.NothingToEraseError{Name: name, Hint: e.Disk.NoErase}
}

func (s *Surveyor) keptOnly(e disk.Entry) string {
	first := ""
	for _, p := range e.Disk.Paths {
		if !p.PlacedOn(s.o.ThisMachine) {
			continue
		}
		if p.Level != catalog.LevelKept && p.Level != catalog.LevelSettings {
			return ""
		}
		if first == "" && !strings.Contains(p.Path, "*") {
			first = s.o.Expand.Display(s.o.Expand.Expand(p.Path))
		}
	}
	return cmp.Or(first, e.Name)
}

func (s *Surveyor) CheckRunning(ctx context.Context, name string) error {
	e, err := s.CheckErase(name)
	if err != nil {
		return err
	}
	if e.Disk.Quit != "" && s.running(ctx, e.Disk.Quit) {
		return &disk.RunningError{Name: name, Process: e.Disk.Quit, Verb: "erase", Subject: name}
	}
	return nil
}

func (s *Surveyor) Plan(name string) []disk.PathReport {
	e, err := s.CheckErase(name)
	if err != nil {
		return nil
	}
	var out []disk.PathReport
	for _, p := range e.Disk.Paths {
		if !p.PlacedOn(s.o.ThisMachine) || (p.Level != catalog.LevelCache && p.Level != catalog.LevelData) {
			continue
		}
		out = append(out, disk.PathReport{Path: s.o.Expand.Display(s.o.Expand.Expand(p.Path)), Level: p.Level, Bytes: s.storedBytes(p)})
	}
	return out
}

func (s *Surveyor) Erase(ctx context.Context, name string) (disk.Outcome, error) {
	// invariant: the caller checked the erase first (CheckRunning).
	e, err := s.CheckErase(name)
	if err != nil {
		return disk.Outcome{}, err
	}
	before := s.levelBytes(e, catalog.LevelCache) + s.levelBytes(e, catalog.LevelData)
	present := s.isInstalled(ctx, e.Name)
	failure := first(s.actions(ctx, e.Disk.Free, present), s.actions(ctx, e.Disk.Erase, present))
	if !present && failure == nil {
		failure = s.deleteCaches(e)
	}
	after := s.levelBytes(e, catalog.LevelCache) + s.levelBytes(e, catalog.LevelData)
	return s.outcome(e.Name, failure, before-after, true), nil
}

func (s *Surveyor) Move(ctx context.Context, folder string) (disk.Outcome, error) {
	// invariant: a move takes a misplaced folder where its app's own config
	// says, by renames only, then applies the settings that make the app look
	// there; the app running refuses it before anything moves.
	abs := folder
	if strings.HasPrefix(folder, "~") {
		abs = s.o.Expand.Expand(folder)
	}
	for _, e := range s.Placed() {
		for _, st := range e.Disk.Misplaced {
			if s.o.Expand.Expand(st.Path) != abs || !s.exists(st.Path) {
				continue
			}
			return s.move(ctx, e, st)
		}
	}
	return disk.Outcome{}, &disk.NotMisplacedError{Path: s.o.Expand.Display(abs)}
}

func (s *Surveyor) move(ctx context.Context, e disk.Entry, st catalog.Misplaced) (disk.Outcome, error) {
	shown := s.o.Expand.Display(s.o.Expand.Expand(st.Path))
	if st.Busy != "" && s.running(ctx, st.Busy) {
		return disk.Outcome{}, &disk.RunningError{Name: e.Name, Process: st.Busy, Verb: "move", Subject: shown}
	}
	var to string
	for _, m := range st.Move {
		from, dest := s.o.Expand.Expand(m.From), s.o.Expand.Expand(m.To)
		pairs := [][2]string{{from, dest}}
		if strings.HasSuffix(m.From, "/*") {
			matches := s.m.Glob(from)
			pairs = pairs[:0]
			for _, src := range matches {
				pairs = append(pairs, [2]string{src, path.Join(dest, path.Base(src))})
			}
		}
		for _, p := range pairs {
			if err := s.o.Disk.Move(ctx, p[0], p[1]); err != nil {
				return disk.Outcome{}, err
			}
			to = s.o.Expand.Display(dest)
		}
	}
	for _, a := range st.Free {
		if a.Run == nil {
			_ = s.action(ctx, a)
		}
	}
	for _, dir := range st.Prune {
		s.m.Prune(s.o.Expand.Expand(dir))
	}
	for _, item := range st.Then {
		if s.o.Configurer == nil {
			continue
		}
		if err := s.o.Configurer.Configure(ctx, item, false); err != nil {
			return disk.Outcome{}, err
		}
	}
	if to == "" {
		return disk.Outcome{Name: shown, Result: catalog.StateOK, Detail: "nothing to move: its rebuildable files freed"}, nil
	}
	return disk.Outcome{Name: shown, Result: catalog.StateOK, Detail: "moved to " + to}, nil
}

func (s *Surveyor) Volume(ctx context.Context) (total, free int64, err error) {
	return s.o.Disk.Volume(ctx)
}

func (s *Surveyor) outcome(name string, f *actionFailure, freed int64, erase bool) disk.Outcome {
	if f != nil {
		return disk.Outcome{Name: name, Result: catalog.StateBroken, Detail: f.detail, Reason: f.reason, Erase: erase}
	}
	return disk.Outcome{Name: name, Result: catalog.StateOK, Freed: max(freed, 0), Erase: erase}
}

func (s *Surveyor) running(ctx context.Context, process string) bool {
	return s.o.Disk.Running(ctx, process)
}

func (s *Surveyor) cacheBytes(e disk.Entry) int64 {
	total := s.levelBytes(e, catalog.LevelCache)
	for _, st := range e.Disk.Misplaced {
		total += s.dirSize(s.o.Expand.Expand(st.Path))
	}
	return total
}

func (s *Surveyor) levelBytes(e disk.Entry, level catalog.Level) int64 {
	var total int64
	for _, p := range e.Disk.Paths {
		if p.Level == level && p.PlacedOn(s.o.ThisMachine) {
			total += s.storedBytes(p)
		}
	}
	return total
}

// invariant: a memo a tool rewrites on every call (mise's bin_paths, written by each
// shim and by station's own reads) is not counted, so a cleaned cache measures empty.
func (s *Surveyor) storedBytes(st catalog.Stored) int64 {
	total := s.size(st.Path)
	for _, m := range st.Memos {
		total -= s.size(path.Join(st.Path, m))
	}
	return total
}

type actionFailure struct{ detail, reason string }

func first(a, b *actionFailure) *actionFailure {
	if a != nil {
		return a
	}
	return b
}

// invariant: every action runs even after one failed, and the first failure is the one
// reported, so a stopped container runtime does not hide an image prune that could run.
// A tool that is not installed has no binary to run, so only its plain deletions run.
func (s *Surveyor) actions(ctx context.Context, actions []catalog.Action, present bool) *actionFailure {
	var failed *actionFailure
	for _, a := range actions {
		if !a.PlacedOn(s.o.ThisMachine) || (!present && len(a.Run) > 0) {
			continue
		}
		failed = first(failed, s.action(ctx, a))
	}
	return failed
}

func (s *Surveyor) action(ctx context.Context, a catalog.Action) *actionFailure {
	if a.Uses != "" {
		if !s.m.Has(a.Uses) {
			return nil
		}
	}
	if a.Exists != "" && !s.exists(a.Exists) {
		return nil
	}
	var failed *actionFailure
	if len(a.Run) > 0 {
		failed = s.run(ctx, a)
	}
	for _, pattern := range a.Delete {
		failed = first(failed, s.delete(pattern))
	}
	for _, dir := range a.Prune {
		s.m.Prune(s.o.Expand.Expand(dir))
	}
	return failed
}

func (s *Surveyor) deleteCaches(e disk.Entry) *actionFailure {
	var failed *actionFailure
	for _, p := range e.Disk.Paths {
		if p.Level == catalog.LevelCache && p.PlacedOn(s.o.ThisMachine) {
			failed = first(failed, s.delete(p.Path))
		}
	}
	return failed
}

func (s *Surveyor) delete(pattern string) *actionFailure {
	matches := s.m.Glob(s.o.Expand.Expand(pattern))
	if err := s.o.Disk.Erase(context.Background(), matches); err != nil {
		return &actionFailure{detail: err.Error()}
	}
	return nil
}

func (s *Surveyor) run(ctx context.Context, a catalog.Action) *actionFailure {
	args := make([]string, 0, len(a.Run)-1)
	for _, w := range a.Run[1:] {
		args = append(args, s.o.Expand.Word(w))
	}
	env := make([]string, 0, len(a.Env))
	for _, kv := range a.Env {
		env = append(env, s.o.Expand.Word(kv))
	}
	label := label(a.Run)
	ran, err := s.m.Run(ctx, a.Run[0], args, env)
	if !ran.Found {
		return nil
	}
	if err != nil {
		return &actionFailure{detail: label + ": " + err.Error()}
	}
	if ran.Code != 0 {
		return &actionFailure{detail: label + " exited " + strconv.Itoa(ran.Code), reason: ran.LastStderr}
	}
	return nil
}

func label(argv []string) string {
	words := []string{argv[0]}
	for _, w := range argv[1:] {
		if strings.HasPrefix(w, "-") {
			break
		}
		words = append(words, w)
	}
	return strings.Join(words, " ")
}

// invariant: a machine without a container runtime measures none of its space.
func (s *Surveyor) containerBytes(ctx context.Context) int64 {
	if s.o.Containers == nil {
		return 0
	}
	return s.o.Containers.Bytes(ctx)
}

func dedupe(names []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, n := range names {
		if !seen[n] {
			seen[n] = true
			out = append(out, n)
		}
	}
	return out
}

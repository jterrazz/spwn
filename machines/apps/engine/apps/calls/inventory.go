package calls

import (
	"cmp"
	"context"
	"errors"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/apps"
	"github.com/jterrazz/studio-station/apps/engine/apps/contract"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/call/lanes"
)

const (
	DetailAppStore = "app-store"
	// value: the detail of an app that updates itself, read present and
	// never behind.
	DetailUpdatesSelf = "updates itself"
	subject           = "apps"
)

const (
	inventoryBudget = 30 * time.Second
	versionBudget   = 5 * time.Second
	applications    = "/Applications"
)

type listing struct {
	keys     map[string]string
	currency map[string]apps.Currency
	missing  bool
	failed   bool
	unlisted bool
}

func (s *Shelf) Inventory(ctx context.Context) ([]apps.Report, error) {
	return lanes.Last(s.Watch(ctx)), ctx.Err()
}

func (s *Shelf) Watch(ctx context.Context) <-chan []apps.Report {
	return lanes.Stream(ctx, s.o.Pool, subject, s.watch)
}

func (s *Shelf) Forget() { s.o.Pool.Forget(subject) }

// invariant: the frame is the catalogue placed here, every row pending; one job
// per method lists its manager once (shared by every reader of the run), and
// each row fills as its own probe answers, so one hung tool holds back no other
// row of its method; a row whose method tells currency waits for it, inside
// the listing.
func (s *Shelf) watch(ctx context.Context) <-chan []apps.Report {
	list := s.Placed()
	frame := make([]apps.Report, len(list))
	byMethod := map[catalog.Method][]int{}
	var methods []catalog.Method
	for i, t := range list {
		m := s.install(t).Method()
		frame[i] = apps.Report{App: t, Method: m, State: catalog.StateUnknown, Pending: true}
		if _, seen := byMethod[m]; !seen {
			methods = append(methods, m)
		}
		byMethod[m] = append(byMethod[m], i)
	}
	jobs := make([]lanes.Job[[]apps.Report], 0, len(methods))
	for _, m := range methods {
		rows := byMethod[m]
		jobs = append(jobs, lanes.Job[[]apps.Report]{
			Key: subject + "/method/" + string(m), Budget: inventoryBudget + versionBudget + contract.CurrencyBudget,
			Partial: func(ctx context.Context, emit func(func(*[]apps.Report))) func(*[]apps.Report) {
				group := make([]catalog.App, len(rows))
				for k, i := range rows {
					group[k] = list[i]
				}
				each := func(k int, r apps.Report) {
					emit(func(out *[]apps.Report) { (*out)[rows[k]] = r })
				}
				reports := s.reportsEach(ctx, group, s.sharedListings(ctx, group), each)
				return func(out *[]apps.Report) {
					for k, i := range rows {
						(*out)[i] = reports[k]
					}
				}
			},
			Late: func(out *[]apps.Report) {
				for _, i := range rows {
					if !(*out)[i].Pending {
						continue
					}
					(*out)[i] = apps.Report{App: list[i], Method: m, State: catalog.StateUnknown, Why: apps.WhyLate, Detail: "no answer in " + (inventoryBudget + versionBudget + contract.CurrencyBudget).String()}
				}
			},
		})
	}
	return lanes.Run(ctx, s.o.Pool, frame, jobs, slices.Clone[[]apps.Report])
}

// invariant: a shared listing is keyed by its method alone, so it asks the
// currency of every placed app of that method, whichever reader starts it.
func (s *Shelf) sharedListings(ctx context.Context, list []catalog.App) map[catalog.Method]listing {
	placed := s.Placed()
	out := map[catalog.Method]listing{}
	for m := range s.needed(list) {
		out[m] = lanes.Share(ctx, s.o.Pool, subject+"/list/"+string(m), func(ctx context.Context) listing { return s.list(ctx, m, placed) })
	}
	return out
}

func (s *Shelf) reports(ctx context.Context, list []catalog.App) []apps.Report {
	return s.reportsWith(ctx, list, s.listings(ctx, list))
}

func (s *Shelf) reportsWith(ctx context.Context, list []catalog.App, listings map[catalog.Method]listing) []apps.Report {
	return s.reportsEach(ctx, list, listings, func(int, apps.Report) {})
}

func (s *Shelf) reportsEach(ctx context.Context, list []catalog.App, listings map[catalog.Method]listing, each func(int, apps.Report)) []apps.Report {
	out := make([]apps.Report, len(list))
	var wg sync.WaitGroup
	for i, t := range list {
		wg.Go(func() {
			out[i] = s.report(ctx, t, listings)
			each(i, out[i])
		})
	}
	wg.Wait()
	return out
}

func (s *Shelf) needed(list []catalog.App) map[catalog.Method]bool {
	needed := map[catalog.Method]bool{}
	for _, t := range list {
		for _, in := range t.InstallsOn(s.o.ThisMachine) {
			needed[in.Method()] = true
		}
		for _, m := range s.managed(t) {
			needed[m] = true
		}
		if old, ok := t.Superseded(); ok {
			needed[old.Method()] = true
		}
	}
	return needed
}

func (s *Shelf) listings(ctx context.Context, list []catalog.App) map[catalog.Method]listing {
	needed := s.needed(list)
	var mu sync.Mutex
	out := map[catalog.Method]listing{}
	var wg sync.WaitGroup
	for m := range needed {
		wg.Go(func() {
			l := s.list(ctx, m, list)
			mu.Lock()
			out[m] = l
			mu.Unlock()
		})
	}
	wg.Wait()
	return out
}

func (s *Shelf) list(ctx context.Context, m catalog.Method, list []catalog.App) listing {
	inst := s.o.Installers[m]
	l := listing{missing: inst.Ready(ctx) != nil}
	listed, cancel := context.WithTimeout(ctx, inventoryBudget)
	keys, err := inst.Inventory(listed)
	cancel()
	switch {
	case errors.Is(err, contract.ErrUnlisted):
		l.unlisted = true
		return l
	case errors.Is(err, contract.ErrMissing):
		l.missing = true
		return l
	case err != nil && l.missing:
		return l
	case err != nil:
		l.failed = true
		return l
	}
	l.keys = keys
	var held []string
	for _, t := range list {
		for _, i := range t.InstallsOn(s.o.ThisMachine) {
			if _, ok := listedAll(keys, i); ok && i.Method() == m && !slices.Contains(held, i.Key()) {
				held = append(held, i.Key())
			}
		}
	}
	if len(held) > 0 {
		told, cancel := context.WithTimeout(ctx, contract.CurrencyBudget)
		l.currency, _ = inst.Outdated(told, held)
		cancel()
	}
	return l
}

func (s *Shelf) report(ctx context.Context, t catalog.App, listings map[catalog.Method]listing) apps.Report {
	i := s.install(t)
	m := i.Method()
	manager := s.installer(t).Manager()
	r := apps.Report{App: t, Method: m, Manager: s.managerKey(m), State: catalog.StateAbsent, Irreversible: s.uninstallable(t) != nil}
	l := listings[m]
	switch {
	case l.unlisted && s.bundled(t):
		r.State, r.Managed = catalog.StateOK, true
		return r
	case l.unlisted:
		return s.readInstalled(ctx, t, r)
	case l.failed:
		r.State, r.Detail, r.Why = catalog.StateUnknown, cmp.Or(manager.Name, string(m))+" did not answer", apps.WhyLate
		return r
	}
	if version, ok := listedAll(l.keys, i); ok {
		r.State, r.Version, r.Managed = catalog.StateOK, version, true
		r.Holds, r.HoldsUnread = s.holds(t, listings)
		if t.UpdatesSelf {
			r.Detail = DetailUpdatesSelf
			return r
		}
		// invariant: an inconclusive currency leaves the app ok, never outdated.
		switch l.currency[i.Key()].Verdict {
		case apps.Unknown:
			r.Detail = "currency unknown"
		case apps.Behind:
			r.State, r.Detail, r.Why = catalog.StateDegraded, "outdated", apps.WhyOutdated
			r.Newest = l.currency[i.Key()].Newest
		case apps.Current:
		}
		return r
	}
	if other, ok := s.heldOtherwise(t, listings); ok {
		held := s.report(ctx, s.as(t, other), listings)
		held.App = t
		return held
	}
	if s.fromAppStore(t) {
		r.State, r.Detail = catalog.StateOK, DetailAppStore
		return r
	}
	if listedOld(t, listings) && !s.bundled(t) {
		r.State, r.Detail, r.Why = catalog.StateDegraded, "formula installed, app absent", apps.WhyPartial
		return r
	}
	if s.present(ctx, t, &r) {
		r.State = catalog.StateDegraded
		if l.missing {
			r.Detail, r.Why, r.Needs = "installed, "+manager.Name+" not installed", apps.WhyMissing, manager.App
		} else {
			r.Detail, r.Why = "on PATH, not managed by "+string(m), apps.WhyUnmanaged
			if s.bundled(t) {
				r.Detail = t.Bundle + ".app in /Applications, not installed by " + string(m)
			}
		}
		return r
	}
	if l.missing {
		r.Detail, r.Why, r.Needs = manager.Name+" not installed", apps.WhyMissing, manager.App
	}
	return r
}

// invariant: an app that another of its methods here lists is held by
// that method, adopted as it is: read, updated and removed by it, never
// installed again by the first.
func (s *Shelf) heldOtherwise(t catalog.App, listings map[catalog.Method]listing) (catalog.Install, bool) {
	all := t.InstallsOn(s.o.ThisMachine)
	if len(all) < 2 {
		return nil, false
	}
	for _, in := range all[1:] {
		if _, ok := listedAll(listings[in.Method()].keys, in); ok {
			return in, true
		}
	}
	return nil, false
}

// invariant: a manager holds what each method it manages lists that no
// app of the catalog placed here installs by that method by default: the
// owner's own packages. A key a catalog app's other method lists may be
// the owner's own install by hand, so it counts as held too, and keeps the
// manager. A list that did not answer, or that its method keeps none of,
// leaves what it holds unread.
func (s *Shelf) holds(t catalog.App, listings map[catalog.Method]listing) (held []string, unread bool) {
	for _, m := range s.managed(t) {
		l, ok := listings[m]
		if !ok || l.keys == nil {
			unread = true
			continue
		}
		ours := map[string]bool{}
		for _, p := range s.Placed() {
			if i := s.install(p); i.Method() == m {
				ours[i.Key()] = true
			}
		}
		for key := range l.keys {
			if !ours[key] {
				held = append(held, key)
			}
		}
	}
	slices.Sort(held)
	return held, unread
}

// value: the methods a manager manages that this machine has an installer
// of; none for an app that manages nothing.
func (s *Shelf) managed(t catalog.App) []catalog.Method {
	it, ok := s.cat.Package(t.Name)
	if !ok || it.Package.Kind != catalog.KindManager {
		return nil
	}
	var out []catalog.Method
	for _, key := range it.Package.Manages {
		if _, wired := s.o.Installers[catalog.Method(key)]; wired {
			out = append(out, catalog.Method(key))
		}
	}
	return out
}

func listedOld(t catalog.App, listings map[catalog.Method]listing) bool {
	old, ok := t.Superseded()
	if !ok {
		return false
	}
	_, listed := listedAll(listings[old.Method()].keys, old)
	return listed
}

// invariant: an install is listed when every key it is listed by is, so
// an app of several packages is held only while all of them are.
func listedAll(keys map[string]string, in catalog.Install) (string, bool) {
	for _, k := range catalog.Keys(in) {
		if _, ok := keys[k]; !ok {
			return "", false
		}
	}
	return keys[in.Key()], true
}

func (s *Shelf) readInstalled(ctx context.Context, t catalog.App, r apps.Report) apps.Report {
	if t.Version.Path != "" {
		matches := s.m.Glob(s.home(t.Version.Path))
		if len(matches) == 0 {
			return r
		}
		r.State, r.Managed = catalog.StateOK, true
		r.Version = path.Base(matches[0])
		if len(t.Version.Args) > 0 {
			if version, code, ok := s.readVersion(ctx, t.Version); ok && code == 0 {
				r.Version = version
			}
		}
		return r
	}
	if check := catalog.CheckOf(s.install(t)); check != "" {
		return s.readChecked(ctx, t, r, check)
	}
	if !s.provided(t) {
		return r
	}
	if len(t.Version.Args) == 0 {
		r.State, r.Managed = catalog.StateOK, true
		return r
	}
	version, code, ok := s.readVersion(ctx, t.Version)
	switch {
	case ok && code == 0:
		r.State, r.Version, r.Managed = catalog.StateOK, version, true
	case ok:
		r.State, r.Why, r.Detail = catalog.StateBroken, apps.WhyFailing, strings.Join(t.Version.Args, " ")+" exited "+strconv.Itoa(code)
	}
	return r
}

// invariant: an install that names a check is there when the check exits 0,
// whatever its commands are; its output is the version a `version-read`
// pattern reads, unless the app reads its version by a command of its own.
// Both run with the folder the install lands its commands in as their whole
// PATH, so they answer for station's copy, never for one found ahead of it
// (an owner's shim). A check that fails or does not start reads the app
// absent.
func (s *Shelf) readChecked(ctx context.Context, t catalog.App, r apps.Report, check string) apps.Report {
	bin := s.home(catalog.ScriptBin)
	text, code, ran := s.m.RunSealed(ctx, bin, "bash", []string{"-c", check}, versionBudget)
	if !ran || code != 0 {
		return r
	}
	r.State, r.Managed = catalog.StateOK, true
	if len(t.Version.Args) > 0 {
		args := make([]string, 0, len(t.Version.Args)-1)
		for _, a := range t.Version.Args[1:] {
			args = append(args, s.home(a))
		}
		name := s.home(t.Version.Args[0])
		if !strings.Contains(name, "/") {
			name = path.Join(bin, name)
		}
		if said, code, ok := s.m.RunSealed(ctx, bin, name, args, versionBudget); ok && code == 0 {
			r.Version = strings.TrimSpace(said)
			if t.Version.Pattern != "" {
				r.Version = matchVersion(t.Version.Pattern, said)
			}
		}
		return r
	}
	if t.Version.Pattern != "" {
		r.Version = matchVersion(t.Version.Pattern, text)
	}
	return r
}

func (s *Shelf) present(ctx context.Context, t catalog.App, r *apps.Report) bool {
	if s.bundled(t) {
		return true
	}
	for _, name := range t.Provides {
		where, ok := s.m.LookPath(name)
		if !ok || s.kept(t, where) {
			continue
		}
		// hazard: stock macOS ships /usr/bin shims (java, python3, xcodebuild, git)
		// that exist without the tool; such a hit counts only when the probe answers.
		if strings.HasPrefix(where, "/usr/bin/") {
			if len(t.Version.Args) == 0 {
				continue
			}
			version, code, ok := s.readVersion(ctx, t.Version)
			if !ok || code != 0 {
				continue
			}
			r.Version = version
			return true
		}
		if len(t.Version.Args) > 0 {
			if version, code, ok := s.readVersion(ctx, t.Version); ok && code == 0 {
				r.Version = version
			}
		}
		return true
	}
	return false
}

// invariant: a hit in a manager's own folder is that manager's only for
// an app it carries here; any other app found there (one the owner put
// there by `mise use -g`, one whose method moved off mise) is present
// another way, adopted rather than installed again.
func (s *Shelf) kept(t catalog.App, where string) bool {
	for _, in := range t.InstallsOn(s.o.ThisMachine) {
		for _, dir := range s.o.Kept[in.Method()] {
			if strings.HasPrefix(where, dir+"/") {
				return true
			}
		}
	}
	return false
}

// value: the key a method's apps are updated by: the first its manager
// package installs by, else the method's own.
func (s *Shelf) managerKey(m catalog.Method) string {
	if owner, ok := s.cat.Manager(string(m)); ok {
		if it, ok := s.cat.Package(owner); ok && len(it.Package.Manages) > 0 {
			return string(it.Package.Manages[0])
		}
	}
	return string(m)
}

func (s *Shelf) provided(t catalog.App) bool {
	for _, name := range t.Provides {
		if _, ok := s.m.LookPath(name); ok {
			return true
		}
	}
	return len(t.Provides) == 0 && len(t.Version.Args) > 0
}

func (s *Shelf) bundled(t catalog.App) bool {
	if t.Bundle == "" {
		return false
	}
	return s.m.Exists(path.Join(s.o.Root, applications, t.Bundle+".app"))
}

// invariant: an App Store copy of an AppStore tool's bundle is its install,
// not station's: present, never managed, upgraded or removed by station.
func (s *Shelf) fromAppStore(t catalog.App) bool {
	if !t.AppStore || t.Bundle == "" {
		return false
	}
	return s.m.Exists(path.Join(s.o.Root, applications, t.Bundle+".app", catalog.AppStoreReceipt))
}

func (s *Shelf) readVersion(ctx context.Context, p catalog.VersionRead) (version string, code int, ran bool) {
	// invariant: the version read runs the binary the lookup found, so it
	// answers for that copy and never for another one station's own search
	// would find first (macOS's /usr/bin stub against a shim).
	args := make([]string, 0, len(p.Args)-1)
	for _, a := range p.Args[1:] {
		args = append(args, s.home(a))
	}
	name := p.Args[0]
	if where, ok := s.m.LookPath(name); ok {
		name = where
	}
	text, code, ran := s.m.RunBriefly(ctx, name, args, versionBudget)
	if !ran {
		return "", 0, false
	}
	if p.Pattern == "" {
		return strings.TrimSpace(text), code, true
	}
	return matchVersion(p.Pattern, text), code, true
}

func matchVersion(pattern, text string) string {
	re, err := regexp.Compile("(?m)" + pattern)
	if err != nil {
		return ""
	}
	if m := re.FindStringSubmatch(text); len(m) > 1 {
		return m[1]
	}
	return ""
}

func (s *Shelf) home(p string) string {
	if p == "~" || strings.HasPrefix(p, "~/") {
		return s.m.Home() + p[1:]
	}
	return p
}

package wire

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/src/apps"
	appsfake "github.com/jterrazz/studio-station/src/apps/adapter/fake"
	"github.com/jterrazz/studio-station/src/apps/adapter/mise"
	appscalls "github.com/jterrazz/studio-station/src/apps/calls"
	"github.com/jterrazz/studio-station/src/apps/contract"
	"github.com/jterrazz/studio-station/src/catalog"
	corefake "github.com/jterrazz/studio-station/src/core/adapter/fake"
	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/core/confirm"
	"github.com/jterrazz/studio-station/src/core/fail"
	coremachine "github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/machinefile"
	mffake "github.com/jterrazz/studio-station/src/machinefile/adapter/fake"
	machinefilecalls "github.com/jterrazz/studio-station/src/machinefile/calls"
	mfcontract "github.com/jterrazz/studio-station/src/machinefile/contract"
	"github.com/jterrazz/studio-station/src/platform/local"
	"github.com/jterrazz/studio-station/src/platform/local/localtest"
	"github.com/jterrazz/studio-station/src/world"
)

// value: each kind of machine as a Machinefile says its system.
func systems() map[catalog.MachineKind]string {
	return map[catalog.MachineKind]string{
		catalog.Mac:   "{ kind: computer, system: { os: macos } }",
		catalog.MacOS: "{ kind: workspace, system: { os: macos } }",
		catalog.Linux: "{ kind: workspace, system: { os: ubuntu, version: '24.04' } }",
	}
}

// value: an item with two methods on macOS, a ranked manager's and
// another's, so a plan may choose either and the machine may hold either.
const twin = "item: station.catalog/v1\nkind: app\nname: twin\ndescription: two ways\nprovides: [twin]\nmethods:\n    - { os: macos, skill: { repo: owner/twin, name: twin } }\n    - { os: macos, cask: twin }\n"

// value: an item the owner may have another way, read by its version.
const probed = "item: station.catalog/v1\nkind: app\nname: probed\ndescription: read by its version\nprovides: [probed]\nversion-read: { args: [probed, --version], pattern: '^probed (\\S+)' }\nmethods:\n    - { os: macos, cask: probed }\n"

// value: an item two lines install on a Mac, each read by its check, the
// first taken back by its undo.
const lined = "item: station.catalog/v1\nkind: app\nname: lined\ndescription: two lines\nprovides: [lined]\nmethods:\n    - { os: macos, script: { run: install-lined, check: lined --version, undo: uninstall-lined } }\n    - { os: macos, script: { url: https://example.com/lined.sh, check: lined version } }\n"

type bench struct {
	w     *world.World
	shelf *appscalls.Shelf
	cat   catalog.Catalog
	lib   machinefile.Library
	// value: the owner's files, which apply reads afresh at each run.
	files map[string]string
	calls *benchCalls
}

// value: this machine's calls as apply reaches them: apps' over the
// bench's shelf, the machinefile family's over the owner's files, the
// built-in source and the world's ledger; the settings read none and set
// nothing.
type benchCalls struct{ registry *call.Registry }

func (c *benchCalls) Invoke(ctx context.Context, _ coremachine.Ref, name string, in, out any, p call.Progress) error {
	if strings.HasPrefix(name, "settings.") {
		return nil
	}
	v, err := c.registry.Run(ctx, benchProfile, name, in, p)
	call.Set(out, v)
	return err
}

func (c *benchCalls) Watch(ctx context.Context, _ coremachine.Ref, name string, in any, p call.Progress) (<-chan call.Update[any], error) {
	if strings.HasPrefix(name, "settings.") {
		none := make(chan call.Update[any])
		close(none)
		return none, nil
	}
	return c.registry.Stream(ctx, benchProfile, name, in, p)
}

var benchProfile = coremachine.Profile{Ref: coremachine.Ref{Name: "here"}, Kind: coremachine.Computer, System: coremachine.System{OS: coremachine.MacOS}}

type ownerFiles struct{ files map[string]string }

func (o ownerFiles) Read(context.Context) (machinefile.Tree, error) {
	user := machinefile.Tree{Layer: machinefile.User, Root: "~/.config/station"}
	for p, body := range o.files {
		user.Files = append(user.Files, machinefile.File{Path: p, Body: []byte(body)})
	}
	slices.SortFunc(user.Files, func(a, b machinefile.File) int { return strings.Compare(a.Path, b.Path) })
	return user, nil
}

func machinefileOf(kind catalog.MachineKind, entries ...string) string {
	return "machinefile: station.machinefile/v1\nname: m\nmachine: " + systems()[kind] + "\napps: [" + strings.Join(entries, ", ") + "]\n"
}

// value: a machine of that kind whose every manager is there and holds
// nothing, the shelf over its fakes, and the library of the built-in
// catalog under the twin and a Machinefile `m` that lists entry; each edit
// sets the machine's runner and the shelf's options before it is built.
func benchOf(t *testing.T, kind catalog.MachineKind, entry string, edits ...func(*localtest.Runner, *appscalls.ShelfOptions)) bench {
	t.Helper()
	builtin, err := builtinSource().Read(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	user := machinefile.Tree{Layer: machinefile.User, Root: "~/.config/station"}
	var listed []string
	if entry != "" {
		listed = []string{entry}
	}
	files := map[string]string{
		"machinefiles/m.yaml":              machinefileOf(kind, listed...),
		"catalog/command-line/twin.yaml":   twin,
		"catalog/command-line/probed.yaml": probed,
		"catalog/command-line/lined.yaml":  lined,
	}
	for p, body := range files {
		user.Files = append(user.Files, machinefile.File{Path: p, Body: []byte(body)})
	}
	lib, err := machinefile.Load([]machinefile.Tree{user, builtin})
	if err != nil {
		t.Fatal(err)
	}
	runner := localtest.NewRunner(t)
	system := localtest.Mac(runner, localtest.NewFS(), localtest.Env{HomeDir: "/home/u", OS: "darwin"}, localtest.NewNet(), &localtest.Clock{})
	adapters, _ := installers(machine{sys: system, vars: map[string]string{}}, false)
	w := &world.World{ThisMac: "here", Managers: map[catalog.Method]world.Named{}}
	w.Mac("here").Apps = map[catalog.Method]*world.Manager{}
	registry := contract.Installers{}
	for method, adapter := range adapters {
		w.Managers[method] = appsfake.Named(adapter)
		w.Mac("here").Apps[method] = &world.Manager{Installed: map[string]string{}}
		_, lined := adapter.(contract.Lined)
		registry[method] = appsfake.InstallerOf(w, "here", method, lined)
	}
	foot := appsfake.FootprintOf(w, "here", lib.Catalog)
	system.Run, system.FS = appsfake.Seen{Runner: system.Run, Foot: foot, Home: "/home/u"}, appsfake.SeenFS{FS: system.FS, Foot: foot, Home: "/home/u"}
	options := appscalls.ShelfOptions{ThisMachine: kind, Root: "/", Installers: registry, Bundler: appsfake.BundlerOf(w, "here")}
	for _, edit := range edits {
		edit(runner, &options)
	}
	shelf := appscalls.NewShelf(waiting{appsMachine{h: system, run: system.Run}}, lib.Catalog, nil, options)
	calls := &benchCalls{}
	deps := machinefilecalls.Deps{
		Sources: func() []mfcontract.Source { return []mfcontract.Source{ownerFiles{files}, builtinSource()} },
		Name:    "m", Kind: coremachine.Computer, Prefs: mfcontract.NewSourcePrefs(corefake.NewPreferences()),
		Ledger: mffake.LedgerOf(w, "here"), Locks: mffake.LocksOf(w, "here"), Password: mffake.PasswordOf(w, "here"), Now: func() time.Time { return time.Unix(0, 0) }, Calls: calls,
	}
	calls.registry, err = call.NewRegistry(appscalls.All(appscalls.Deps{Apps: shelf, Known: lib.Catalog.Apps, Kind: coremachine.Computer}), machinefilecalls.All(deps))
	if err != nil {
		t.Fatal(err)
	}
	return bench{w: w, shelf: shelf, cat: lib.Catalog, lib: lib, files: files, calls: calls}
}

// value: `station apply` of the Machinefile m on a terminal, with
// --remove and its yes when remove says so.
func (b bench) apply(t *testing.T, remove bool) machinefile.Applied {
	t.Helper()
	in := machinefilecalls.ApplyIn{Interactive: true, Remove: remove}
	if remove {
		in.Confirm = confirm.Yes
	}
	applied, err := call.Invoke(context.Background(), b.calls, coremachine.Ref{}, machinefilecalls.Apply, in, nil)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	return applied
}

// value: the line of item in the plan of m as `station plan` reads it.
func (b bench) planned(t *testing.T, item string) machinefile.Line {
	t.Helper()
	b.shelf.Forget()
	p, err := call.Invoke(context.Background(), b.calls, coremachine.Ref{}, machinefilecalls.Plan, machinefilecalls.PlanIn{}, nil)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	i := slices.IndexFunc(p.Lines, func(l machinefile.Line) bool { return l.Step.Item == item })
	if i < 0 {
		t.Fatalf("no line %s in the plan", item)
	}
	return p.Lines[i]
}

func (b bench) recorded(item string) (machinefile.Installed, bool) {
	for _, in := range b.w.Mac("here").Store().Ledger {
		if in.Item == item {
			return in, true
		}
	}
	return machinefile.Installed{}, false
}

func doneOf(t *testing.T, done []machinefile.Done, item string) machinefile.Done {
	t.Helper()
	i := slices.IndexFunc(done, func(d machinefile.Done) bool { return d.Line.Step.Item == item })
	if i < 0 {
		t.Fatalf("no line of %s ran in %+v", item, done)
	}
	return done[i]
}

// value: the machine whose wait lasts until its run ends, so no read's
// budget runs out on a fake clock's instant.
type waiting struct{ appsMachine }

func (waiting) Sleep(ctx context.Context, _ time.Duration) error {
	<-ctx.Done()
	return ctx.Err()
}

func (b bench) step(t *testing.T, item string) machinefile.Step {
	t.Helper()
	r, err := machinefile.Resolve(b.lib, "m", machinefile.Machine{}, machinefile.Lock{})
	if err != nil {
		t.Fatal(err)
	}
	i := slices.IndexFunc(r.Steps, func(s machinefile.Step) bool { return s.Item == item })
	if i < 0 {
		t.Fatalf("no step %s in %+v", item, r.Steps)
	}
	return r.Steps[i]
}

func (b bench) row(t *testing.T, item string) apps.Report {
	t.Helper()
	b.shelf.Forget()
	reports, err := b.shelf.Inventory(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	i := slices.IndexFunc(reports, func(r apps.Report) bool { return r.App.Name == item })
	if i < 0 {
		t.Fatalf("no row %s", item)
	}
	return reports[i]
}

func (b bench) holds(method catalog.Method, key string) bool {
	_, ok := b.w.Mac("here").Apps[method].Installed[key]
	return ok
}

func outcomeOf(t *testing.T, out []apps.Outcome, item string) apps.Outcome {
	t.Helper()
	i := slices.IndexFunc(out, func(o apps.Outcome) bool { return o.Name == item })
	if i < 0 {
		t.Fatalf("no outcome of %s in %+v", item, out)
	}
	return out[i]
}

// value: every entry a Machinefile may write for an app on a kind of
// machine: its name alone, and with each manager key it has a method by
// there, at a version where that method pins one.
func entries(cat catalog.Catalog, a catalog.App, kind catalog.MachineKind) []string {
	out := []string{a.Name}
	it, _ := cat.Item(a.Name)
	for _, m := range it.Item.Methods {
		key := m.Key()
		if !slices.Contains(catalog.ManagerKey("").Enum(), key) {
			continue
		}
		if _, ok := a.InstallBy(kind, catalog.Method(m.Method())); !ok {
			continue
		}
		e := "{ name: " + a.Name + ", manager: " + key + " }"
		if m.Pins() {
			e = "{ name: " + a.Name + ", manager: " + key + ", version: '9.8.7' }"
		}
		if !slices.Contains(out, e) {
			out = append(out, e)
		}
	}
	return out
}

func TestTheMethodApplyRunsIsTheMethodThePlanShows(t *testing.T) {
	// invariant: for every app of the catalog, on every kind of machine,
	// for every entry a Machinefile may write of it, the method its step
	// names is the one `station apply` installs it by at the version
	// asked, the one the ledger records and the read after it names, the
	// one a named update, a bulk update of its manager and a removal
	// reach; and the plan then keeps it, neither installing nor adopting.
	ctx := context.Background()
	base := benchOf(t, catalog.Mac, "")
	cat := base.cat
	for kind := range systems() {
		for _, a := range cat.Apps {
			if in, ok := a.InstallOn(kind); !ok || !registryCan(base, in.Method()).Install {
				continue
			}
			for _, entry := range entries(cat, a, kind) {
				about := string(kind) + " " + entry
				b := benchOf(t, kind, entry)
				step := b.step(t, a.Name)
				method := catalog.Method(step.Method)
				in, _ := a.InstallBy(kind, method)
				done := doneOf(t, b.apply(t, false).Done, a.Name)
				if done.State != catalog.StateOK || !b.holds(method, in.Key()) {
					t.Errorf("%s: the plan showed %s, apply ended %s %q and left %s without it", about, step.Method, done.State, done.Detail, step.Method)
					continue
				}
				if step.Version != "" && done.Version != step.Version {
					t.Errorf("%s: asked at %s, holds %s", about, step.Version, done.Version)
				}
				if rec, ok := b.recorded(a.Name); !ok || rec.Method != step.Method {
					t.Errorf("%s: installed by %s, recorded %+v", about, step.Method, rec)
				}
				row := b.row(t, a.Name)
				if string(row.Method) != step.Method {
					t.Errorf("%s: installed by %s, read back by %s", about, step.Method, row.Method)
				}
				if line := b.planned(t, a.Name); line.Action != machinefile.Keep || line.Method() != step.Method {
					t.Errorf("%s: then planned %s by %s", about, line.Action, line.Method())
				}
				can := registryCan(b, method)
				if can.Update {
					up, err := b.shelf.Upgrade(ctx, apps.UpgradeRequest{Names: []string{a.Name}})
					if err != nil || string(outcomeOf(t, up, a.Name).Method) != step.Method {
						t.Errorf("%s: an update reached %+v (%v)", about, up, err)
					}
				}
				b.w.Mac("here").Apps[method].Behind = map[string]string{in.Key(): "99.0.0"}
				// invariant: a bare update takes the rows its method reads behind, so a
				// method that reads no currency is reached by name only.
				seen, _ := appsfake.InstallerOf(b.w, "here", method, false).Outdated(ctx, []string{in.Key()})
				if can.Update && can.UpdateAll && seen[in.Key()].Verdict == apps.Behind {
					var via []catalog.ManagerKey
					if _, ok := b.cat.Manager(row.Manager); ok {
						via = []catalog.ManagerKey{catalog.ManagerKey(row.Manager)}
					}
					b.shelf.Forget()
					up, err := b.shelf.Upgrade(ctx, apps.UpgradeRequest{All: via == nil, Via: via})
					if i := slices.IndexFunc(up, func(o apps.Outcome) bool { return o.Name == a.Name }); err != nil || i < 0 || string(up[i].Method) != step.Method || b.w.Mac("here").Apps[method].Installed[in.Key()] != "99.0.0" {
						t.Errorf("%s: the update of %v reached %+v (%v)", about, via, up, err)
					}
				}
				if can.Remove && catalog.Undoable(in) {
					if _, err := b.shelf.Uninstall(ctx, []string{a.Name}, true); err != nil || b.holds(method, in.Key()) {
						t.Errorf("%s: the removal left the %s copy (%v)", about, step.Method, err)
					}
				}
			}
		}
	}
}

func registryCan(b bench, m catalog.Method) contract.Abilities {
	return appsfake.InstallerOf(b.w, "here", m, false).Can()
}

func TestAnAppHeldByAnotherOfItsMethodsIsAdoptedNeverInstalledAgain(t *testing.T) {
	// invariant: on every path that meets an app another of its methods
	// holds (its row, the plan, an install by name, an update, a removal),
	// the method that holds it is the one named and reached.
	ctx := context.Background()
	for _, c := range []struct{ item, held, key string }{{"twin", "skill", "twin"}} {
		b := benchOf(t, catalog.Mac, c.item)
		b.w.Mac("here").Apps[catalog.Method(c.held)].Installed[c.key] = "1.0.0"
		step := b.step(t, c.item)
		row := b.row(t, c.item)
		if string(row.Method) != c.held || row.State != catalog.StateOK {
			t.Errorf("%s: reads %s by %s (%s), want ok by %s", c.item, row.State, row.Method, row.Detail, c.held)
		}
		seen := map[string]machinefile.Seen{c.item: {Condition: machinefile.Present, State: catalog.StateOK, Method: string(row.Method)}}
		line := machinefile.Judge(machinefile.Resolved{Name: "m", Steps: []machinefile.Step{step}}, b.cat, seen, nil, false).Lines[0]
		if step.Method == c.held || line.Action != machinefile.Adopt || line.Method() != c.held {
			t.Errorf("%s: the plan chose %s and %s it by %s; want it adopted by %s", c.item, step.Method, line.Action, line.Method(), c.held)
		}
		out, err := b.shelf.Install(ctx, apps.InstallRequest{Names: []string{c.item}}, nil)
		if o := outcomeOf(t, out, c.item); err != nil || o.Detail != apps.AlreadyInstalled || string(o.Method) != c.held || b.holds(catalog.Method(step.Method), c.key) {
			t.Errorf("%s: installed again: %+v (%v)", c.item, out, err)
		}
		up, err := b.shelf.Upgrade(ctx, apps.UpgradeRequest{Names: []string{c.item}})
		if err != nil || string(outcomeOf(t, up, c.item).Method) != c.held {
			t.Errorf("%s: the update reached %+v (%v)", c.item, up, err)
		}
		gone, err := b.shelf.Uninstall(ctx, []string{c.item}, true)
		if err != nil || string(outcomeOf(t, gone, c.item).Method) != c.held || b.holds(catalog.Method(c.held), c.key) {
			t.Errorf("%s: the removal reached %+v (%v)", c.item, gone, err)
		}
	}
}

func TestAMethodAskedThatTheAppHasNoneOfHereIsRefused(t *testing.T) {
	b := benchOf(t, catalog.Mac, "twin")
	_, err := b.shelf.Install(context.Background(), apps.InstallRequest{Names: []string{"twin"}, Method: catalog.MethodShell}, nil)
	if err == nil || !strings.Contains(err.Error(), "no method shell") {
		t.Errorf("installed twin by shell: %v", err)
	}
}

func TestAVersionAMethodCannotPinIsACodedRefusal(t *testing.T) {
	// invariant: a version asked of a method whose install carries none
	// is refused with its code, the words the binary that prints it says.
	b := benchOf(t, catalog.Mac, "twin")
	_, err := call.Invoke(context.Background(), b.calls, coremachine.Ref{}, appscalls.Install, appscalls.InstallIn{Names: []string{"twin"}, Method: "cask", Version: "1.2.3"}, nil)
	var refused *fail.Error
	if !errors.As(err, &refused) || refused.Code != appscalls.Unpinnable || refused.Detail["method"] != "cask" || refused.Detail["version"] != "1.2.3" {
		t.Errorf("installed twin at 1.2.3 by cask: %v", err)
	}
}

func TestAMoveToAMethodTheAppHasNoneOfNamesTheOneThatHoldsIt(t *testing.T) {
	// invariant: a move refused for a method the app has none of here
	// names the method that holds it, so its hint never asks a move to
	// where the app already is.
	b := benchOf(t, catalog.Mac, "probed")
	b.w.Mac("here").Apps[catalog.MethodCask].Installed["probed"] = "1.0.0"
	_, err := call.Invoke(context.Background(), b.calls, coremachine.Ref{}, appscalls.Move, appscalls.MoveIn{Name: "probed", To: "skill"}, nil)
	var refused *fail.Error
	if !errors.As(err, &refused) || refused.Code != appscalls.NoMethod || refused.Detail["held"] != "cask" {
		t.Errorf("moved probed to skill: %v", err)
	}
}

func TestAnUpdateAskedInV30sManagerWordReachesThatManager(t *testing.T) {
	// invariant: apps.update read from a v3.0 station, its manager in
	// v3.0's word (`skills`), updates that manager's apps, as today's key.
	b := benchOf(t, catalog.Mac, "twin")
	b.w.Mac("here").Apps[catalog.MethodSkill].Installed["twin"] = "1.0.0"
	b.w.Mac("here").Apps[catalog.MethodSkill].Behind = map[string]string{"twin": "2.0.0"}
	out, err := call.Invoke(context.Background(), b.calls, coremachine.Ref{}, appscalls.Update, appscalls.UpdateIn{Method: []string{"skills"}}, nil)
	if err != nil || len(out) != 1 || out[0].Name != "twin" || b.w.Mac("here").Apps[catalog.MethodSkill].Installed["twin"] != "2.0.0" {
		t.Errorf("updated %+v (%v)", out, err)
	}
}

func TestAKeyAnotherMethodListsIsHeldByThatManager(t *testing.T) {
	// invariant: whatever a manager's removal would take back, its holds
	// read covers it: twin is held by its default method, so the copy the
	// other method's manager lists is the owner's, and keeps that manager.
	b := benchOf(t, catalog.Mac, "twin")
	b.w.Mac("here").Apps[catalog.MethodCask].Installed["twin"] = "1.0.0"
	b.w.Mac("here").Apps[catalog.MethodSkill].Installed["twin"] = "1.0.0"
	if row := b.row(t, "twin"); row.Method != catalog.MethodCask {
		t.Fatalf("twin reads by %s, want its default", row.Method)
	}
	owner, _ := b.cat.Manager(string(catalog.MethodSkill))
	manager, _ := b.cat.App(owner)
	in, _ := manager.InstallOn(catalog.Mac)
	b.w.Mac("here").Apps[in.Method()].Installed[in.Key()] = "1.0.0"
	if row := b.row(t, owner); !slices.Contains(row.Holds, "twin") {
		t.Errorf("%s holds %v; want the copy its method lists counted as yours", owner, row.Holds)
	}
}

func TestAMoveHandsTheAppToTheMethodAskedOnEveryPath(t *testing.T) {
	// invariant: once moved, the app is held by the method asked alone:
	// the outcome, the read after it and the plan that asks that method
	// name it, and the copy of the method it left is gone.
	ctx := context.Background()
	for _, c := range []struct{ item, from, to, entry string }{
		{"twin", "skill", "cask", "twin"}, {"twin", "cask", "skill", "{ name: twin, manager: skill }"},
	} {
		about := c.item + " " + c.from + " to " + c.to
		b := benchOf(t, catalog.Mac, c.entry)
		b.w.Mac("here").Apps[catalog.Method(c.from)].Installed[c.item] = "1.0.0"
		out, err := b.shelf.Move(ctx, c.item, catalog.Method(c.to), nil)
		if err != nil {
			t.Fatalf("%s: %v", about, err)
		}
		o := outcomeOf(t, out, c.item)
		if string(o.Method) != c.to || !b.holds(catalog.Method(c.to), c.item) || b.holds(catalog.Method(c.from), c.item) {
			t.Errorf("%s: moved to %s; held by %s %v, by %s %v", about, o.Method, c.to, b.holds(catalog.Method(c.to), c.item), c.from, b.holds(catalog.Method(c.from), c.item))
		}
		row := b.row(t, c.item)
		step := b.step(t, c.item)
		seen := map[string]machinefile.Seen{c.item: {Condition: machinefile.Present, State: catalog.StateOK, Method: string(row.Method)}}
		line := machinefile.Judge(machinefile.Resolved{Name: "m", Steps: []machinefile.Step{step}}, b.cat, seen, nil, false).Lines[0]
		if string(row.Method) != c.to || step.Method != c.to || line.Action != machinefile.Keep {
			t.Errorf("%s: reads by %s, the plan asks %s and %s it", about, row.Method, step.Method, line.Action)
		}
	}
}

func TestACommandInAManagersOwnFolderIsThatManagersForTheAppsItCarries(t *testing.T) {
	// invariant: a command found in a manager's own folder (mise's shims
	// and installs) is that manager's for an app it carries here, read by
	// its listing: one its listing holds not is absent, so a Machinefile's
	// global copy is installed, never adopted from a shim or a project's
	// runtime. An app mise carries not here, found there, is present
	// another way: adopted, never installed again.
	kept := mise.Kept("/home/u", map[string]string{})
	carried := ""
	for _, a := range benchOf(t, catalog.Mac, "").cat.Apps {
		if in, ok := a.InstallOn(catalog.Mac); ok && kept[in.Method()] != nil && len(a.InstallsOn(catalog.Mac)) == 1 && len(a.Provides) > 0 {
			carried = a.Name
			break
		}
	}
	if carried == "" {
		t.Fatal("no app of the catalog carried by the kept folders' manager alone")
	}
	for _, dirs := range kept {
		for _, dir := range dirs {
			for _, c := range []struct {
				item string
				want apps.State
			}{{carried, apps.NotInstalled}, {"probed", apps.NotManaged}} {
				b := benchOf(t, catalog.Mac, c.item)
				command := b.lib.Catalog.Apps[slices.IndexFunc(b.lib.Catalog.Apps, func(a catalog.App) bool { return a.Name == c.item })].Provides[0]
				where := dir + "/" + c.item + "/2/bin/" + command
				b = benchOf(t, catalog.Mac, c.item, func(r *localtest.Runner, o *appscalls.ShelfOptions) {
					r.Paths[command] = where
					r.On(where, "--version").Reply(local.Output{Stdout: command + " 2.0.0\n"})
					o.Kept = kept
				})
				if row := b.row(t, c.item); apps.StateOf(row) != c.want {
					t.Errorf("%s at %s reads %s (%s), want %s", c.item, where, apps.StateOf(row), row.Detail, c.want)
				}
			}
		}
	}
}

func TestTheVersionReadRunsTheCopyTheLookupFound(t *testing.T) {
	// invariant: the version is read from the binary the lookup found on
	// the owner's PATH, never from another one station's search finds.
	var runner *localtest.Runner
	b := benchOf(t, catalog.Mac, "probed", func(r *localtest.Runner, _ *appscalls.ShelfOptions) {
		r.Paths["probed"] = "/opt/elsewhere/probed"
		r.On("/opt/elsewhere/probed", "--version").Reply(local.Output{Stdout: "probed 2.1.0\n"})
		runner = r
	})
	row := b.row(t, "probed")
	if row.Version != "2.1.0" || !slices.ContainsFunc(runner.Calls, func(c local.Cmd) bool { return c.Name == "/opt/elsewhere/probed" }) {
		t.Errorf("read %q by %v; want 2.1.0 from /opt/elsewhere/probed", row.Version, runner.Calls)
	}
}

func TestAnExtraHeldByAnotherMethodThanItsRecordIsLeftByApplyRemove(t *testing.T) {
	// invariant: apply --remove takes back an extra only by the method
	// its record names: one moved to another method since is the
	// owner's, left there with its record let go (ADR-034).
	b := benchOf(t, catalog.Mac, "{ name: twin, manager: cask }")
	if done := doneOf(t, b.apply(t, false).Done, "twin"); done.State != catalog.StateOK {
		t.Fatalf("twin by cask ended %s %q", done.State, done.Detail)
	}
	if _, err := b.shelf.Move(context.Background(), "twin", catalog.MethodSkill, nil); err != nil {
		t.Fatal(err)
	}
	b.files["machinefiles/m.yaml"] = machinefileOf(catalog.Mac)
	b.shelf.Forget()
	left := doneOf(t, b.apply(t, true).Removed, "twin")
	if !b.holds(catalog.MethodSkill, "twin") || left.Detail != machinefile.AnotherWay {
		t.Errorf("apply --remove ended twin %s %q; the skill copy there: %v", left.State, left.Detail, b.holds(catalog.MethodSkill, "twin"))
	}
	if rec, ok := b.recorded("twin"); ok {
		t.Errorf("the record %+v stays", rec)
	}
}

func TestAMoveWhoseInstallDoesNotLandKeepsTheOldCopy(t *testing.T) {
	// invariant: a move installs by the method asked before it touches
	// the old copy, so an install that fails or is interrupted leaves the
	// old copy where it was.
	for _, c := range []struct{ from, to, how string }{
		{"skill", "cask", "fails"}, {"cask", "skill", "fails"},
		{"skill", "cask", "interrupted"}, {"cask", "skill", "interrupted"},
	} {
		about := c.from + " to " + c.to + " " + c.how
		ctx, cancel := context.WithCancel(context.Background())
		b := benchOf(t, catalog.Mac, "twin", func(_ *localtest.Runner, o *appscalls.ShelfOptions) {
			if c.how == "interrupted" {
				o.Installers[catalog.Method(c.to)] = interrupting{o.Installers[catalog.Method(c.to)], cancel}
				o.Bundler = interruptingBundle{o.Bundler, cancel}
			}
		})
		b.w.Mac("here").Apps[catalog.Method(c.from)].Installed["twin"] = "1.0.0"
		if c.how == "fails" {
			b.w.Script = []world.Fault{{Call: "Installer.Install"}, {Call: "Bundler.Bundle"}}
		}
		out, _ := b.shelf.Move(ctx, "twin", catalog.Method(c.to), nil)
		cancel()
		if i := slices.IndexFunc(out, func(o apps.Outcome) bool { return o.Name == "twin" }); i >= 0 && out[i].Result == catalog.StateOK {
			t.Errorf("%s: moved %+v", about, out[i])
		}
		if !b.holds(catalog.Method(c.from), "twin") {
			t.Errorf("%s: the %s copy is gone", about, c.from)
		}
	}
}

// value: an install that lands, then is caught by ctrl-c before it answers.
type interrupting struct {
	contract.Installer
	cancel func()
}

func (i interrupting) Install(ctx context.Context, in []catalog.Install, logs contract.Logs) error {
	_ = i.Installer.Install(ctx, in, logs)
	i.cancel()
	return ctx.Err()
}

type interruptingBundle struct {
	contract.Bundler
	cancel func()
}

func (b interruptingBundle) Bundle(ctx context.Context, lines []string, update bool, logs contract.Logs) error {
	_ = b.Bundler.Bundle(ctx, lines, update, logs)
	b.cancel()
	return ctx.Err()
}

func TestAMoveFromACopyOnPathKeepsIt(t *testing.T) {
	// invariant: a copy none of the app's methods holds is the owner's: a
	// move installs by the method asked and leaves that copy, said.
	b := benchOf(t, catalog.Mac, "twin", func(r *localtest.Runner, _ *appscalls.ShelfOptions) {
		r.Paths["twin"] = "/opt/yours/twin"
	})
	out, err := b.shelf.Move(context.Background(), "twin", catalog.MethodSkill, nil)
	if err != nil {
		t.Fatal(err)
	}
	o := outcomeOf(t, out, "twin")
	if o.Result != catalog.StateOK || !b.holds(catalog.MethodSkill, "twin") || len(o.Steps) != 1 || o.Steps[0].Item != "copy on PATH" || !strings.HasPrefix(o.Steps[0].Detail, "kept") {
		t.Errorf("moved %+v", o)
	}
}

// value: a ledger that holds an install of each app it names, by shell.
type ledgered []string

func (l ledgered) Recorded(_ context.Context, app string, m catalog.Method) (bool, error) {
	return m == catalog.MethodShell && slices.Contains(l, app), nil
}

func TestAMoveOffALineRunsItsUndoOnlyForACopyTheLedgerHolds(t *testing.T) {
	// invariant: a move carries no word of the owner's, so the old copy of a
	// line (a script, a shell line) is undone only when the ledger says
	// station installed it; one it does not hold is the owner's, kept.
	for _, c := range []struct {
		name   string
		ledger ledgered
		kept   bool
	}{{"yours", nil, true}, {"station's", ledgered{"lined"}, false}} {
		b := benchOf(t, catalog.Mac, "", func(_ *localtest.Runner, o *appscalls.ShelfOptions) { o.HeldByLedger = c.ledger.Recorded })
		b.w.Mac("here").Apps[catalog.MethodShell].Installed["install-lined"] = "1.2.0"
		out, err := b.shelf.Move(context.Background(), "lined", catalog.MethodScript, nil)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		o := outcomeOf(t, out, "lined")
		_, there := b.w.Mac("here").Apps[catalog.MethodShell].Installed["install-lined"]
		if o.Result != catalog.StateOK || len(o.Steps) != 1 || o.Steps[0].Item != "shell copy" || there != c.kept || strings.HasPrefix(o.Steps[0].Detail, "kept") != c.kept {
			t.Errorf("%s: moved %+v, the shell copy there: %v", c.name, o, there)
		}
	}
}

// value: an installer that holds what it adds by app, whose record holds
// the app or not, keeping whom each verb was told it serves.
type holding struct {
	contract.Installer
	holds bool
	told  []contract.For
}

func (h *holding) InstallFor(ctx context.Context, f contract.For, installs []catalog.Install, logs contract.Logs) error {
	h.told = append(h.told, f)
	return h.Install(ctx, installs, logs)
}

func (h *holding) RemoveFor(ctx context.Context, f contract.For, installs []catalog.Install, logs contract.Logs) error {
	h.told = append(h.told, f)
	return h.Remove(ctx, installs, logs)
}

func (h *holding) Holds(context.Context, string) (bool, error) { return h.holds, nil }

func TestAHolderIsToldTheAppAndWhetherTheOwnerGaveTheirWord(t *testing.T) {
	// invariant: a method that holds what it adds by app is told the app
	// each verb serves, and whether the owner asked it by name: a removal
	// without that word of an app it recorded nothing for is refused before
	// anything runs, and a move leaves that copy, the owner's.
	ctx := context.Background()
	spy := &holding{}
	b := benchOf(t, catalog.Mac, "twin", func(_ *localtest.Runner, o *appscalls.ShelfOptions) {
		spy.Installer = o.Installers[catalog.MethodCask]
		o.Installers[catalog.MethodCask] = spy
	})
	b.w.Mac("here").Apps[catalog.MethodCask].Installed["twin"] = "1.0.0"

	var unrecorded *apps.UnrecordedError
	if _, err := b.shelf.Uninstall(ctx, []string{"twin"}, false); !errors.As(err, &unrecorded) || !b.holds(catalog.MethodCask, "twin") || len(spy.told) > 0 {
		t.Errorf("a removal without the owner's word gave %v, told %v; want it refused, the copy kept", err, spy.told)
	}
	out, err := b.shelf.Move(ctx, "twin", catalog.MethodSkill, nil)
	if o := outcomeOf(t, out, "twin"); err != nil || !b.holds(catalog.MethodCask, "twin") || len(o.Steps) != 1 || !strings.HasPrefix(o.Steps[0].Detail, "kept: yours") || !strings.Contains(o.Steps[0].Detail, "station apps remove twin") {
		t.Errorf("moved %+v (%v); want the cask copy kept, the owner's", out, err)
	}
	b.w.Mac("here").Apps[catalog.MethodSkill].Installed = map[string]string{}
	b.shelf.Forget()
	if _, err := b.shelf.Uninstall(ctx, []string{"twin"}, true); err != nil || b.holds(catalog.MethodCask, "twin") || !slices.Equal(spy.told, []contract.For{{App: "twin", Word: true}}) {
		t.Errorf("the owner's removal gave %v, told %v; want twin removed for its own name, on the word", err, spy.told)
	}
}

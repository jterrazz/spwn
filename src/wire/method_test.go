package wire

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/src/apps"
	appsfake "github.com/jterrazz/studio-station/src/apps/adapter/fake"
	appscalls "github.com/jterrazz/studio-station/src/apps/calls"
	"github.com/jterrazz/studio-station/src/apps/contract"
	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/machinefile"
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

type bench struct {
	w     *world.World
	shelf *appscalls.Shelf
	cat   catalog.Catalog
	lib   machinefile.Library
}

// value: a machine of that kind whose every manager is there and holds
// nothing, the shelf over its fakes, and the library of the built-in
// catalog under the twin and a Machinefile `m` that lists entry.
func benchOf(t *testing.T, kind catalog.MachineKind, entry string) bench {
	t.Helper()
	builtin, err := builtinSource().Read(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	user := machinefile.Tree{Layer: machinefile.User, Root: "~/.config/station"}
	files := map[string]string{
		"machinefiles/m.yaml":            "machinefile: station.machinefile/v1\nname: m\nmachine: " + systems()[kind] + "\napps: [" + entry + "]\n",
		"catalog/command-line/twin.yaml": twin,
	}
	for p, body := range files {
		user.Files = append(user.Files, machinefile.File{Path: p, Body: []byte(body)})
	}
	lib, err := machinefile.Load([]machinefile.Tree{user, builtin})
	if err != nil {
		t.Fatal(err)
	}
	system := localtest.Mac(localtest.NewRunner(t), localtest.NewFS(), localtest.Env{HomeDir: "/home/u", OS: "darwin"}, localtest.NewNet(), &localtest.Clock{})
	adapters, _ := installers(machine{sys: system, vars: map[string]string{}}, false)
	w := &world.World{ThisMac: "here", Managers: map[catalog.Method]world.Named{}}
	w.Mac("here").Apps = map[catalog.Method]*world.Manager{}
	registry := contract.Installers{}
	for method, adapter := range adapters {
		w.Managers[method] = world.Named{Name: adapter.Manager().Name, App: adapter.Manager().App}
		w.Mac("here").Apps[method] = &world.Manager{Installed: map[string]string{}}
		_, lined := adapter.(contract.Lined)
		registry[method] = appsfake.InstallerOf(w, "here", method, lined)
	}
	shelf := appscalls.NewShelf(waiting{appsMachine{h: system, run: system.Run}}, lib.Catalog, nil, appscalls.ShelfOptions{ThisMachine: kind, Root: "/", Installers: registry, Bundler: appsfake.BundlerOf(w, "here")})
	return bench{w: w, shelf: shelf, cat: lib.Catalog, lib: lib}
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
	r, err := machinefile.Resolve(b.lib, "m", machinefile.Machine{})
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
		if !slices.Contains(catalog.ManagerKey("").Enum(), key) || !m.Runs(it.Item.Kind) {
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
	// names is the one apps.install runs at the version asked, the one the
	// outcome (which the ledger records) and the read after it (an
	// interrupted install's record) name, the one an update and a removal
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
				out, err := b.shelf.Install(ctx, apps.InstallRequest{Names: []string{a.Name}, Method: catalog.Method(step.Method), Version: step.Version}, nil)
				if err != nil {
					t.Errorf("%s: %v", about, err)
					continue
				}
				o := outcomeOf(t, out, a.Name)
				in, _ := a.InstallBy(kind, catalog.Method(step.Method))
				if string(o.Method) != step.Method || !b.holds(o.Method, in.Key()) {
					t.Errorf("%s: the plan showed %s, the install ran %s", about, step.Method, o.Method)
				}
				if step.Version != "" && o.Version != step.Version {
					t.Errorf("%s: asked at %s, holds %s", about, step.Version, o.Version)
				}
				row := b.row(t, a.Name)
				if string(row.Method) != step.Method {
					t.Errorf("%s: installed by %s, read back by %s", about, step.Method, row.Method)
				}
				seen := map[string]machinefile.Seen{a.Name: {Condition: machinefile.Present, State: catalog.StateOK, Method: string(row.Method)}}
				line := machinefile.Judge(machinefile.Resolved{Name: "m", Steps: []machinefile.Step{step}}, b.cat, seen, nil, false).Lines[0]
				if line.Action != machinefile.Keep || line.Method() != step.Method {
					t.Errorf("%s: then planned %s by %s", about, line.Action, line.Method())
				}
				if can := registryCan(b, o.Method); can.Update {
					up, err := b.shelf.Upgrade(ctx, apps.UpgradeRequest{Names: []string{a.Name}})
					if err != nil || string(outcomeOf(t, up, a.Name).Method) != step.Method {
						t.Errorf("%s: an update reached %+v (%v)", about, up, err)
					}
				}
				if can := registryCan(b, o.Method); can.Remove {
					if _, err := b.shelf.Uninstall(ctx, []string{a.Name}); err != nil || b.holds(o.Method, in.Key()) {
						t.Errorf("%s: the removal left the %s copy (%v)", about, o.Method, err)
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
		if err != nil || outcomeOf(t, out, c.item).Detail != apps.AlreadyInstalled || b.holds(catalog.Method(step.Method), c.key) {
			t.Errorf("%s: installed again: %+v (%v)", c.item, out, err)
		}
		up, err := b.shelf.Upgrade(ctx, apps.UpgradeRequest{Names: []string{c.item}})
		if err != nil || string(outcomeOf(t, up, c.item).Method) != c.held {
			t.Errorf("%s: the update reached %+v (%v)", c.item, up, err)
		}
		gone, err := b.shelf.Uninstall(ctx, []string{c.item})
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

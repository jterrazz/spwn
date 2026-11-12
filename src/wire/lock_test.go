package wire

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/src/apps"
	appsfake "github.com/jterrazz/studio-station/src/apps/adapter/fake"
	appscalls "github.com/jterrazz/studio-station/src/apps/calls"
	"github.com/jterrazz/studio-station/src/apps/contract"
	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/core/call"
	coremachine "github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/machinefile"
	mffake "github.com/jterrazz/studio-station/src/machinefile/adapter/fake"
	machinefilecalls "github.com/jterrazz/studio-station/src/machinefile/calls"
	"github.com/jterrazz/studio-station/src/platform/local/localtest"
	"github.com/jterrazz/studio-station/src/world"
)

// value: a bench whose Machinefile `m` lists entries, with its lock body
// seeded at its file (none when empty).
type lockBench struct{ bench }

func lockBenchOf(t *testing.T, kind catalog.MachineKind, lock string, entries ...string) lockBench {
	t.Helper()
	b := benchOf(t, kind, "")
	b.files["machinefiles/m.yaml"] = machinefileOf(kind, entries...)
	if lock != "" {
		b.w.Mac("here").Files = map[string]string{"machinefiles/m.lock": lock}
	}
	return lockBench{b}
}

func (b lockBench) apply(t *testing.T) machinefile.Applied {
	t.Helper()
	b.shelf.Forget()
	return b.bench.apply(t, false)
}

func (b lockBench) lock(t *testing.T) catalog.LockFile {
	t.Helper()
	l, _, err := mffake.LocksOf(b.w, "here").Read(context.Background(), "m")
	if err != nil {
		t.Fatal(err)
	}
	return l
}

func lockedOf(l catalog.LockFile, item string) (catalog.LockedApp, bool) {
	return machinefile.Lock{LockFile: l}.App(item)
}

// invariant: the lock names what the machine holds: for every app the
// plan lists that reads there by a method, that method and the version it
// reads; a copy no method holds, none; every other app keeps what the
// lock held of it before the run.
func listless(b bench, m catalog.Method) bool {
	_, err := appsfake.InstallerOf(b.w, "here", m, false).Inventory(context.Background())
	return errors.Is(err, contract.ErrUnlisted)
}

func holdsWhatTheLockNames(t *testing.T, about string, b lockBench, plan machinefile.Plan, l catalog.LockFile) {
	t.Helper()
	for _, line := range plan.Lines {
		if line.Step.Kind == catalog.KindSetting {
			continue
		}
		row := b.row(t, line.Step.Item)
		got, ok := lockedOf(l, line.Step.Item)
		st := apps.StateOf(row)
		if row.Managed && row.Method != "" && (st == apps.OK || st == apps.UpdateAvailable) {
			want := catalog.LockedApp{Name: line.Step.Item, Method: string(row.Method), Version: row.Version}
			if listless(b.bench, row.Method) || listless(b.bench, catalog.Method(got.Method)) {
				// invariant: a method that keeps no list is read by its app's own check, which
				// names neither the method that put the app there nor the version its
				// install outcome lists (none), so the lock and the read agree by name.
				want, got = catalog.LockedApp{Name: want.Name}, catalog.LockedApp{Name: got.Name}
			}
			if !ok || got != want {
				t.Errorf("%s: %s holds %+v, the lock names %+v", about, line.Step.Item, want, got)
			}
			continue
		}
		was, had := plan.Lock.App(line.Step.Item)
		if st == apps.NotManaged {
			was, had = catalog.LockedApp{}, false
		}
		if ok != had || got != was {
			t.Errorf("%s: %s reads %s, and the lock moved from %+v to %+v", about, line.Step.Item, st, was, got)
		}
	}
}

func TestWhatApplyInstallsIsWhatTheLockNamesOnEveryPath(t *testing.T) {
	// invariant: on every kind of machine, for every app of the catalog
	// placed there: a fresh apply writes the lock of what it installed; an
	// apply of that lock on a fresh machine of the same kind installs by
	// its method at its version wherever that method pins; and the lock of
	// another kind of machine, whose method this one has not, gives way to
	// this machine's choice, written anew. On each path the lock then
	// names what the machine holds.
	base := benchOf(t, catalog.Mac, "")
	written := map[catalog.MachineKind]map[string]string{}
	pinned := map[catalog.Method]int{}
	for kind := range systems() {
		written[kind] = map[string]string{}
		for _, a := range base.cat.Apps {
			if in, ok := a.InstallOn(kind); !ok || !registryCan(base, in.Method()).Install {
				continue
			}
			about := string(kind) + " " + a.Name
			fresh := lockBenchOf(t, kind, "", a.Name)
			out := fresh.apply(t)
			l := fresh.lock(t)
			holdsWhatTheLockNames(t, about+" fresh", fresh, out.Plan, l)
			body, err := catalog.EncodeLock(l)
			if err != nil {
				t.Fatal(err)
			}
			written[kind][a.Name] = string(body)
			again := lockBenchOf(t, kind, string(body), a.Name)
			out = again.apply(t)
			for _, d := range out.Done {
				locked, ok := lockedOf(l, d.Line.Step.Item)
				if !ok || !d.Line.Acts() {
					continue
				}
				if d.Method != locked.Method {
					t.Errorf("%s locked: %s locked by %s, installed by %s", about, d.Line.Step.Item, locked.Method, d.Method)
				}
				if d.Line.Step.Spec.Pins() {
					pinned[catalog.Method(d.Line.Step.Method)]++
					if !catalog.Exactly(locked.Version).Contains(d.Version) {
						t.Errorf("%s locked: %s locked at %s, installed at %s", about, d.Line.Step.Item, locked.Version, d.Version)
					}
				}
			}
			holdsWhatTheLockNames(t, about+" locked", again, out.Plan, again.lock(t))
		}
	}
	for _, m := range []catalog.Method{catalog.MethodApt, catalog.MethodGitHubRelease} {
		if pinned[m] == 0 {
			t.Errorf("no app of the catalog was installed at its locked version by %s", m)
		}
	}
	for kind := range systems() {
		for other := range systems() {
			if other == kind {
				continue
			}
			for name, body := range written[other] {
				a, _ := base.cat.App(name)
				if in, ok := a.InstallOn(kind); !ok || !registryCan(base, in.Method()).Install {
					continue
				}
				about := string(kind) + " " + name + " from a lock of " + string(other)
				b := lockBenchOf(t, kind, body, name)
				out := b.apply(t)
				holdsWhatTheLockNames(t, about, b, out.Plan, b.lock(t))
			}
		}
	}
}

func TestTheLockChoosesAmongTheMethodsTheMachinefileAllows(t *testing.T) {
	// invariant: a lock's method is kept over the system's order, its
	// version where it meets the one asked; one the entry no longer allows
	// gives way to the entry, and the lock follows what is installed.
	rt := benchOf(t, catalog.Mac, "node").step(t, "node").Method
	for _, c := range []struct {
		about, lock, entry string
		method, version    string
	}{
		{"a lock of the other method", "{ name: twin, method: skill, version: 1.0.0 }", "twin", "skill", ""},
		{"a locked version", "{ name: node, method: " + rt + ", version: 22.11.0 }", "node", rt, "22.11.0"},
		{"a locked version under the one asked", "{ name: node, method: " + rt + ", version: 22.11.0 }", "{ name: node, version: '22' }", rt, "22.11.0"},
		{"a locked version the entry no longer asks", "{ name: node, method: " + rt + ", version: 22.11.0 }", "{ name: node, version: '24' }", rt, "24"},
		{"a locked manager the entry no longer asks", "{ name: twin, method: skill, version: 1.0.0 }", "{ name: twin, manager: cask }", "cask", ""},
	} {
		b := lockBenchOf(t, catalog.Mac, "lock: station.lock/v1\nmachinefile: m\napps:\n    - "+c.lock+"\n", c.entry)
		out := b.apply(t)
		i := slices.IndexFunc(out.Done, func(d machinefile.Done) bool {
			return d.Line.Step.Item != "" && strings.Contains(c.entry, d.Line.Step.Item)
		})
		if i < 0 {
			t.Fatalf("%s: no line in %+v", c.about, out.Done)
		}
		d := out.Done[i]
		if d.Method != c.method || (c.version != "" && d.Version != c.version) {
			t.Errorf("%s: installed %s by %s, want %s by %s", c.about, d.Version, d.Method, c.version, c.method)
		}
		holdsWhatTheLockNames(t, c.about, b, out.Plan, b.lock(t))
	}
}

func TestAnAppAdoptedIsLockedByTheMethodThatHoldsIt(t *testing.T) {
	b := lockBenchOf(t, catalog.Mac, "", "twin")
	b.w.Mac("here").Apps["skill"].Installed["twin"] = "2.0.0"
	out := b.apply(t)
	got, ok := lockedOf(b.lock(t), "twin")
	if !ok || got.Method != "skill" || got.Version != "2.0.0" {
		t.Errorf("locked %+v (%v); want twin 2.0.0 by skill", got, ok)
	}
	holdsWhatTheLockNames(t, "adopted", b, out.Plan, b.lock(t))
}

func TestAnUpgradeLeavesTheLockOfWhatItUpdated(t *testing.T) {
	// invariant: after an upgrade the lock names what the machine holds,
	// a version the Machinefile pins kept as it was, and its diff says
	// each move.
	ctx := context.Background()
	b := lockBenchOf(t, catalog.Mac, "", "jq", "{ name: python, version: 3.12.7 }")
	b.apply(t)
	before := b.lock(t)
	for _, m := range b.w.Mac("here").Apps {
		for key := range m.Installed {
			if m.Behind == nil {
				m.Behind = map[string]string{}
			}
			m.Behind[key] = "9.0.0"
		}
	}
	b.shelf.Forget()
	up, err := call.Invoke(ctx, b.calls, coremachine.Ref{}, machinefilecalls.Upgrade, machinefilecalls.UpgradeIn{Interactive: true}, nil)
	if err != nil {
		t.Fatalf("upgrade: %v", err)
	}
	after := b.lock(t)
	holdsWhatTheLockNames(t, "upgraded", b, up.Plan, after)
	if jq, _ := lockedOf(after, "jq"); jq.Version != "9.0.0" {
		t.Errorf("jq locked at %s; want 9.0.0", jq.Version)
	}
	if was, _ := lockedOf(before, "python"); func() bool { now, _ := lockedOf(after, "python"); return now != was }() {
		t.Errorf("python, pinned by the Machinefile, moved in the lock")
	}
	if !slices.ContainsFunc(up.Lock.Apps, func(c machinefile.LockChange) bool { return c.App == "jq" }) {
		t.Errorf("the diff %+v names no jq", up.Lock.Apps)
	}
	managers := true
	for _, d := range up.Done {
		if d.Line.Step.Kind != catalog.KindManager {
			managers = false
		} else if !managers {
			t.Errorf("the manager %s updated after an app", d.Line.Step.Item)
		}
	}
}

func TestALockedVersionThatCannotInstallHasAWayOut(t *testing.T) {
	// invariant: a locked version that no longer installs is no dead end:
	// apply's failure names `station upgrade`, which installs the app the
	// lock holds but the machine does not, at the newest the Machinefile
	// allows, and the lock follows what it installed.
	ctx := context.Background()
	rt := benchOf(t, catalog.Mac, "node").step(t, "node").Method
	b := lockBenchOf(t, catalog.Mac, "lock: station.lock/v1\nmachinefile: m\napps:\n    - { name: node, method: "+rt+", version: 22.11.0 }\n", "node")
	b.apply(t)
	delete(b.w.Mac("here").Apps[catalog.Method(rt)].Installed, "node")
	b.w.Script = []world.Fault{{Call: "Installer.Install", Times: 1}}
	out := b.apply(t)
	i := slices.IndexFunc(out.Done, func(d machinefile.Done) bool { return d.Line.Step.Item == "node" })
	if i < 0 || out.Done[i].State != catalog.StateBroken || !strings.Contains(out.Done[i].Detail, "station upgrade") {
		t.Fatalf("apply ended %+v; want node broken, naming station upgrade", out.Done)
	}
	if got, _ := lockedOf(b.lock(t), "node"); got.Version != "22.11.0" {
		t.Fatalf("the failed install moved the lock to %+v", got)
	}
	b.shelf.Forget()
	up, err := call.Invoke(ctx, b.calls, coremachine.Ref{}, machinefilecalls.Upgrade, machinefilecalls.UpgradeIn{Interactive: true}, nil)
	if err != nil {
		t.Fatalf("upgrade: %v", err)
	}
	j := slices.IndexFunc(up.Done, func(d machinefile.Done) bool { return d.Line.Step.Item == "node" })
	if j < 0 || up.Done[j].State != catalog.StateOK || !up.Done[j].Installed {
		t.Fatalf("upgrade ended %+v; want node installed", up.Done)
	}
	if got, ok := lockedOf(b.lock(t), "node"); !ok || got.Version == "22.11.0" || got.Version == "" {
		t.Errorf("the lock holds %+v; want node at the version upgrade installed", got)
	}
	if _, ok := b.recorded("node"); !ok {
		t.Error("the app upgrade installed is not in the ledger")
	}
	holdsWhatTheLockNames(t, "upgraded anew", b, up.Plan, b.lock(t))
}

// value: a bundle that lands, then is caught by ctrl-c once armed.
type armedInterrupt struct {
	contract.Bundler
	cancel func()
	armed  *bool
}

func (a armedInterrupt) Bundle(ctx context.Context, lines []string, update bool, logs contract.Logs) error {
	err := a.Bundler.Bundle(ctx, lines, update, logs)
	if *a.armed {
		a.cancel()
		return ctx.Err()
	}
	return err
}

func TestAnInterruptedUpgradeLocksWhatInstalled(t *testing.T) {
	// invariant: an upgrade caught by ctrl-c just after its manager
	// finished reads the app again, as apply does, and the lock names the
	// version the machine holds, not the one before.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	armed := false
	b := lockBench{benchOf(t, catalog.Mac, "jq", func(_ *localtest.Runner, o *appscalls.ShelfOptions) {
		o.Bundler = armedInterrupt{o.Bundler, cancel, &armed}
	})}
	b.apply(t)
	for _, m := range b.w.Mac("here").Apps {
		for key := range m.Installed {
			if m.Behind == nil {
				m.Behind = map[string]string{}
			}
			m.Behind[key] = "9.0.0"
		}
	}
	b.shelf.Forget()
	armed = true
	_, _ = call.Invoke(ctx, b.calls, coremachine.Ref{}, machinefilecalls.Upgrade, machinefilecalls.UpgradeIn{Interactive: true}, nil)
	held := b.row(t, "jq").Version
	if got, _ := lockedOf(b.lock(t), "jq"); got.Version != held || held != "9.0.0" {
		t.Errorf("the lock names jq at %q, the machine holds %q (want 9.0.0)", got.Version, held)
	}
}

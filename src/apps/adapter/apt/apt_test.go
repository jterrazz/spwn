package apt_test

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/src/apps"
	"github.com/jterrazz/studio-station/src/apps/adapter/apt"
	"github.com/jterrazz/studio-station/src/apps/contract"
	"github.com/jterrazz/studio-station/src/apps/contract/contracttest"
	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/platform/local"
	"github.com/jterrazz/studio-station/src/platform/local/localtest"
)

// value: dpkg's database as these tests play it: each installed package
// and its version, what depends on what, and every command run.
type dpkg struct {
	held    map[string]string
	depends map[string][]string
	fails   bool
	// value: an install is cut short once dpkg wrote what it adds, as by
	// ctrl-c.
	cut   bool
	calls []string
}

func (d *dpkg) play(c local.Cmd) (local.Output, error) {
	args := strings.Join(c.Args, " ")
	d.calls = append(d.calls, c.Name+" "+args)
	switch {
	case c.Name == "dpkg-query":
		var out strings.Builder
		for p, v := range d.held {
			out.WriteString(p + "\t" + v + "\tii \n")
		}
		out.WriteString("gone\t1.0\trc \n")
		return local.Output{Stdout: out.String()}, nil
	case c.Name == "apt-get" && strings.HasPrefix(args, "-s remove "):
		var out strings.Builder
		for _, p := range c.Args[slices.Index(c.Args, "--")+1:] {
			out.WriteString("Remv " + p + " [" + d.held[p] + "]\n")
			for _, by := range d.depends[p] {
				out.WriteString("Remv " + by + " [1]\n")
			}
		}
		return local.Output{Stdout: out.String()}, nil
	case c.Name != "sudo":
		return local.Output{ExitCode: 97}, nil
	case d.fails:
		return local.Output{ExitCode: 100, Stderr: "E: Unable to locate package"}, nil
	}
	verb := slices.IndexFunc(c.Args, func(a string) bool { return a == "install" || a == "remove" || a == "update" })
	for _, p := range c.Args[verb+1:] {
		switch {
		case strings.HasPrefix(p, "-"), p == "APT::Get::AutomaticRemove=false":
		case c.Args[verb] == "remove":
			delete(d.held, p)
		case slices.Contains(c.Args, "--only-upgrade"):
			d.held[p] = "2.0"
		default:
			name, version, _ := strings.Cut(p, "=")
			d.held[name] = cmpOr(version, "1.0")
		}
	}
	if d.cut && c.Args[verb] == "install" {
		return local.Output{ExitCode: 130}, nil
	}
	return local.Output{}, nil
}

func cmpOr(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func played(d *dpkg) *apt.Apt {
	return aptOf(local.Played(d.play, "apt-get", "dpkg-query", "apt-cache", "sudo"), false)
}

const aptStore = "/home/u/.local/share/station/apt"

// value: a verb for the app, on the owner's word (a removal asked by
// name), or as an apply runs it, the ledger's alone.
func word(app string) contract.For  { return contract.For{App: app, Word: true} }
func apply(app string) contract.For { return contract.For{App: app} }

// value: a Holder's verb as a plain one, run for one app on the owner's
// word.
func serving(verb func(context.Context, contract.For, []catalog.Install, contract.Logs) error) func(context.Context, []catalog.Install, contract.Logs) error {
	return func(ctx context.Context, in []catalog.Install, logs contract.Logs) error {
		return verb(ctx, word("app"), in, logs)
	}
}

func aptOf(run local.Runner, terminal bool) *apt.Apt {
	return apt.New(run, localtest.NewFS(), aptStore, terminal)
}

func playedOn(d *dpkg, files local.FS) *apt.Apt {
	return apt.New(local.Played(d.play, "apt-get", "dpkg-query", "apt-cache", "sudo"), files, aptStore, false)
}

func rooted(d *dpkg) bool {
	return slices.ContainsFunc(d.calls, func(c string) bool { return strings.HasPrefix(c, "sudo") })
}

func TestInstaller(t *testing.T) {
	contracttest.Installer(t, catalog.AptInstall{Packages: []string{"fd-find"}}, built)
	contracttest.Holder(t, catalog.AptInstall{Packages: []string{"fd-find"}}, built)
}

func built(_ *testing.T, seed contracttest.Seed) contract.Installer {
	d := &dpkg{held: map[string]string{}, fails: seed.Fails}
	for k, v := range seed.Installed {
		d.held[k] = v
	}
	names := []string{"apt-get", "dpkg-query", "apt-cache", "sudo"}
	if seed.Absent {
		names = []string{"sudo"}
	}
	return aptOf(local.Played(d.play, names...), false)
}

func TestAPackageRemovedWithItsConfigurationKeptIsNotHeld(t *testing.T) {
	got, err := played(&dpkg{held: map[string]string{"tree": "2.1.1"}}).Inventory(context.Background())

	if err != nil || got["tree"] != "2.1.1" || len(got) != 1 {
		t.Errorf("inventory %v (%v); want tree alone, `gone` (rc) left out", got, err)
	}
}

func TestEveryRunIsRootAndAsksNothing(t *testing.T) {
	// invariant: every apt run that changes the machine is sudo's, and
	// never stops on a question: off a terminal sudo never prompts (-n),
	// debconf takes its defaults, a changed config file is kept.
	d := &dpkg{held: map[string]string{"bat": "1.0"}}
	a := played(d)
	ctx := context.Background()
	for _, run := range []func() error{
		func() error {
			return a.InstallFor(ctx, word("fd-find"), []catalog.Install{catalog.AptInstall{Packages: []string{"fd-find"}}}, nil)
		},
		func() error {
			return a.Update(ctx, []catalog.Install{catalog.AptInstall{Packages: []string{"bat"}}}, nil)
		},
		func() error {
			return a.RemoveFor(ctx, word("bat"), []catalog.Install{catalog.AptInstall{Packages: []string{"bat"}}}, nil)
		},
	} {
		if err := run(); err != nil {
			t.Fatal(err)
		}
	}
	for _, c := range d.calls {
		if strings.HasPrefix(c, "dpkg-query ") || strings.HasPrefix(c, "apt-get -s ") {
			continue
		}
		if !strings.HasPrefix(c, "sudo -n env DEBIAN_FRONTEND=noninteractive apt-get -y -o Dpkg::Options::=--force-confdef -o Dpkg::Options::=--force-confold ") {
			t.Errorf("ran %q; want every change as root, asking nothing", c)
		}
	}
	if !slices.Contains(d.calls, "sudo -n env DEBIAN_FRONTEND=noninteractive apt-get -y -o Dpkg::Options::=--force-confdef -o Dpkg::Options::=--force-confold remove -o APT::Get::AutomaticRemove=false -- bat") {
		t.Errorf("ran %q; want the removal never to autoremove", d.calls)
	}
}

func TestOnATerminalSudoMayAskItsPassword(t *testing.T) {
	d := &dpkg{held: map[string]string{}}
	a := aptOf(local.Played(d.play, "apt-get", "dpkg-query", "sudo"), true)

	if err := a.InstallFor(context.Background(), word("jq"), []catalog.Install{catalog.AptInstall{Packages: []string{"jq"}}}, nil); err != nil {
		t.Fatal(err)
	}
	for _, c := range d.calls {
		if strings.HasPrefix(c, "sudo -n") {
			t.Errorf("ran %q on a terminal; want sudo free to ask", c)
		}
	}
}

func TestARemovalThatWouldTakeMoreTakesNothing(t *testing.T) {
	// invariant: what apt removes beside what it was asked (what depends
	// on it) is no removal station read: refused before anything runs.
	d := &dpkg{held: map[string]string{"libfoo": "1.0", "foo-tool": "1.0"}, depends: map[string][]string{"libfoo": {"foo-tool"}}}

	err := played(d).RemoveFor(context.Background(), word("libfoo"), []catalog.Install{catalog.AptInstall{Packages: []string{"libfoo"}}}, nil)

	var more *apt.TakesMoreError
	if !errors.As(err, &more) || !slices.Equal(more.Also, []string{"foo-tool"}) {
		t.Fatalf("removal gave %v; want it refused for foo-tool", err)
	}
	if _, ok := d.held["libfoo"]; !ok || slices.ContainsFunc(d.calls, func(c string) bool { return strings.HasPrefix(c, "sudo") }) {
		t.Errorf("ran %q; want nothing run as root", d.calls)
	}
}

func TestAVersionAskedIsTheFirstPackages(t *testing.T) {
	d := &dpkg{held: map[string]string{}}

	in, _ := catalog.Pinned(catalog.AptInstall{Packages: []string{"tree", "tree-extra"}}, "1:2.43.0")
	if err := played(d).InstallFor(context.Background(), word("tree"), []catalog.Install{in}, nil); err != nil {
		t.Fatal(err)
	}
	if d.held["tree"] != "1:2.43.0" || d.held["tree-extra"] != "1.0" {
		t.Errorf("held %v; want tree at the version asked, tree-extra at apt's candidate", d.held)
	}
}

func TestAFailedRunSaysAptsVerbAndItsLastWords(t *testing.T) {
	err := played(&dpkg{held: map[string]string{}, fails: true}).InstallFor(context.Background(), word("nope"), []catalog.Install{catalog.AptInstall{Packages: []string{"nope"}}}, nil)

	if err == nil || err.Error() != (&local.ExitError{Step: "apt-get update", Code: 100, LastStderr: "E: Unable to locate package"}).Error() {
		t.Errorf("failed as %v; want apt-get's verb and last line", err)
	}
}

func TestOutdatedReadsTheCandidateAgainstWhatIsInstalled(t *testing.T) {
	policy := "tree:\n  Installed: 1:2.43.0-1\n  Candidate: 1:2.43.0-2\n  Version table:\n     1:2.43.0-2 500\njq:\n  Installed: 1.7.1\n  Candidate: 1.7.1\nfd-find:\n  Installed: (none)\n  Candidate: 9.0.0\n"
	run := local.Played(func(local.Cmd) (local.Output, error) { return local.Output{Stdout: policy}, nil }, "apt-cache")

	got, err := aptOf(run, false).Outdated(context.Background(), []string{"tree", "jq", "fd-find"})

	want := map[string]apps.Currency{
		"tree":    {Verdict: apps.Behind, Newest: "1:2.43.0-2"},
		"jq":      {Verdict: apps.Current},
		"fd-find": {Verdict: apps.Unknown},
	}
	if err != nil || len(got) != len(want) {
		t.Fatalf("outdated %v (%v)", got, err)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s reads %+v; want %+v", k, got[k], v)
		}
	}
}

func TestAHostileNameNeverReachesApt(t *testing.T) {
	// invariant: a package or a version that is no name by the catalog's
	// rule is refused before anything runs, as root or not.
	type verb func(*apt.Apt) func(context.Context, []catalog.Install, contract.Logs) error
	install := func(a *apt.Apt) func(context.Context, []catalog.Install, contract.Logs) error {
		return serving(a.InstallFor)
	}
	all := []verb{install, func(a *apt.Apt) func(context.Context, []catalog.Install, contract.Logs) error { return a.Update },
		func(a *apt.Apt) func(context.Context, []catalog.Install, contract.Logs) error {
			return serving(a.RemoveFor)
		}}
	type hostile struct {
		in    catalog.AptInstall
		verbs []verb
	}
	cases := make([]hostile, 0, 2*len(contracttest.HostilePackages())+len(contracttest.HostileVersions()))
	for _, p := range contracttest.HostilePackages() {
		cases = append(cases, hostile{catalog.AptInstall{Packages: []string{p}}, all}, hostile{catalog.AptInstall{Packages: []string{"jq", p}}, all})
	}
	for _, v := range contracttest.HostileVersions() {
		cases = append(cases, hostile{catalog.AptInstall{Packages: []string{"jq"}, Version: v}, []verb{install}})
	}
	for _, c := range cases {
		for _, v := range c.verbs {
			d := &dpkg{held: map[string]string{"jq": "1.0"}}
			err := v(played(d))(context.Background(), []catalog.Install{c.in}, nil)
			var name *catalog.NameError
			if !errors.As(err, &name) || len(d.calls) > 0 {
				t.Errorf("%q (%q) gave %v and ran %q; want it refused, nothing run", c.in.Packages, c.in.Version, err, d.calls)
			}
		}
	}
}

func TestEveryArgvEndsItsOptionsBeforeThePackages(t *testing.T) {
	d := &dpkg{held: map[string]string{"bat": "1.0"}}
	a := played(d)
	ctx := context.Background()
	bat := []catalog.Install{catalog.AptInstall{Packages: []string{"bat"}}}
	for _, run := range []func() error{
		func() error { return a.InstallFor(ctx, word("bat"), bat, nil) },
		func() error { return a.Update(ctx, bat, nil) },
		func() error { return a.RemoveFor(ctx, word("bat"), bat, nil) },
	} {
		if err := run(); err != nil {
			t.Fatal(err)
		}
	}
	for _, c := range d.calls {
		if strings.HasSuffix(c, " bat") && !strings.HasSuffix(c, " -- bat") {
			t.Errorf("ran %q; want `--` before the packages", c)
		}
	}
}

func TestAptIsReadInItsOwnWords(t *testing.T) {
	var reads []local.Cmd
	d := &dpkg{held: map[string]string{"bat": "1.0"}}
	run := local.Played(func(c local.Cmd) (local.Output, error) {
		if c.Name != "sudo" {
			reads = append(reads, c)
		}
		return d.play(c)
	}, "apt-get", "dpkg-query", "apt-cache", "sudo")
	a := aptOf(run, false)
	ctx := context.Background()

	_, _ = a.Inventory(ctx)
	_, _ = a.Outdated(ctx, []string{"bat"})
	_ = a.RemoveFor(ctx, word("bat"), []catalog.Install{catalog.AptInstall{Packages: []string{"bat"}}}, nil)

	if len(reads) != 4 {
		t.Fatalf("read %d times; want dpkg-query, apt-cache policy, then dpkg-query and apt-get -s remove", len(reads))
	}
	for _, c := range reads {
		if !slices.Contains(c.Env, "LC_ALL=C") {
			t.Errorf("%s %q read in the machine's language (%q); want LC_ALL=C", c.Name, c.Args, c.Env)
		}
	}
}

func TestARemovalKeepsAPackageTheOwnerHadBefore(t *testing.T) {
	// invariant: of an app's packages beside its own, a removal takes back
	// those station's install added, never one the owner had before it.
	d := &dpkg{held: map[string]string{"libfoo-dev": "1.0"}}
	a := played(d)
	ctx := context.Background()
	foo := []catalog.Install{catalog.AptInstall{Packages: []string{"foo", "libfoo-dev", "foo-data"}}}

	if err := a.InstallFor(ctx, apply("foo"), foo, nil); err != nil {
		t.Fatal(err)
	}
	if err := a.RemoveFor(ctx, apply("foo"), foo, nil); err != nil {
		t.Fatal(err)
	}
	if _, kept := d.held["libfoo-dev"]; !kept || len(d.held) != 1 {
		t.Errorf("held %v after the removal; want libfoo-dev alone, the owner's", d.held)
	}
}

func TestAnAptInstallOfNoPackageIsRefusedNotIndexed(t *testing.T) {
	// invariant: an install of no package (one that crossed the wire, or was
	// built by hand) is refused by every verb before anything runs.
	empty := []catalog.Install{catalog.AptInstall{}}
	for name, run := range map[string]func(*apt.Apt) error{
		"install": func(a *apt.Apt) error { return a.InstallFor(context.Background(), word("empty"), empty, nil) },
		"update":  func(a *apt.Apt) error { return a.Update(context.Background(), empty, nil) },
		"remove":  func(a *apt.Apt) error { return a.RemoveFor(context.Background(), word("empty"), empty, nil) },
	} {
		d := &dpkg{held: map[string]string{}}
		var err error
		func() {
			defer func() {
				if r := recover(); r != nil {
					err = fmt.Errorf("panicked: %v", r)
				}
			}()
			err = run(played(d))
		}()
		if err == nil || len(d.calls) > 0 || strings.Contains(err.Error(), "panicked") {
			t.Errorf("%s of no package gave %v and ran %q; want it refused, nothing run", name, err, d.calls)
		}
	}
}

func TestARemovalKeepsTheOwnersOwnPackageWhereStationAddedOnlyTheRest(t *testing.T) {
	// invariant: an install that found the app's own package held by the
	// owner and added the others is taken back by those alone: the owner's
	// package is never station's to remove.
	d := &dpkg{held: map[string]string{"foo": "1.0"}}
	a := played(d)
	ctx := context.Background()
	foo := []catalog.Install{catalog.AptInstall{Packages: []string{"foo", "foo-extra"}}}

	if err := a.InstallFor(ctx, apply("foo"), foo, nil); err != nil {
		t.Fatal(err)
	}
	if err := a.RemoveFor(ctx, apply("foo"), foo, nil); err != nil {
		t.Fatal(err)
	}
	if _, kept := d.held["foo"]; !kept || len(d.held) != 1 {
		t.Errorf("held %v after the removal; want foo alone, the owner's", d.held)
	}
}

func TestARemovalKeepsAPackageAnotherAppStillHolds(t *testing.T) {
	// invariant: a package two apps' installs share is taken back with the
	// last of them, not the first.
	d := &dpkg{held: map[string]string{}}
	a := played(d)
	ctx := context.Background()
	one := []catalog.Install{catalog.AptInstall{Packages: []string{"aaa", "libshared"}}}
	other := []catalog.Install{catalog.AptInstall{Packages: []string{"bbb", "libshared"}}}
	for app, in := range map[string][]catalog.Install{"aaa": one, "bbb": other} {
		if err := a.InstallFor(ctx, apply(app), in, nil); err != nil {
			t.Fatal(err)
		}
	}

	if err := a.RemoveFor(ctx, apply("aaa"), one, nil); err != nil {
		t.Fatal(err)
	}
	if want := (map[string]string{"bbb": "1.0", "libshared": "1.0"}); !maps.Equal(d.held, want) {
		t.Errorf("held %v after removing the first app; want %v", d.held, want)
	}
	if err := a.RemoveFor(ctx, apply("bbb"), other, nil); err != nil {
		t.Fatal(err)
	}
	if len(d.held) != 0 {
		t.Errorf("held %v after removing both apps; want none", d.held)
	}
}

func TestAnApplyTakesNothingOfAnAppStationAddedNothingOf(t *testing.T) {
	// invariant: a removal without the owner's word for the app (an
	// apply's) takes back only what station recorded adding: an app it
	// added nothing of is the owner's, refused before anything runs.
	d := &dpkg{held: map[string]string{"foo": "1.0"}}
	a := played(d)
	foo := []catalog.Install{catalog.AptInstall{Packages: []string{"foo"}}}

	holds, err := a.Holds(context.Background(), "foo")
	if err != nil || holds {
		t.Fatalf("holds %v (%v); want station holding nothing of foo", holds, err)
	}
	err = a.RemoveFor(context.Background(), apply("foo"), foo, nil)

	var unrecorded *apps.UnrecordedError
	if !errors.As(err, &unrecorded) || d.held["foo"] != "1.0" || rooted(d) {
		t.Errorf("removal gave %v and ran %q; want it refused, foo kept", err, d.calls)
	}
}

func TestARemovalTakesBackWhatTheInstallAddedWhateverTheAppListsNow(t *testing.T) {
	// invariant: a removal reads the app's record, not its list: an app
	// edited since its install takes back what that install added, and the
	// owner's own package stays.
	d := &dpkg{held: map[string]string{"foo": "1.0"}}
	a := played(d)
	ctx := context.Background()

	if err := a.InstallFor(ctx, apply("x"), []catalog.Install{catalog.AptInstall{Packages: []string{"foo", "foo-extra"}}}, nil); err != nil {
		t.Fatal(err)
	}
	if err := a.RemoveFor(ctx, apply("x"), []catalog.Install{catalog.AptInstall{Packages: []string{"foo"}}}, nil); err != nil {
		t.Fatal(err)
	}
	if want := (map[string]string{"foo": "1.0"}); !maps.Equal(d.held, want) {
		t.Errorf("held %v after the removal; want %v, the owner's", d.held, want)
	}
}

func TestTwoAppsThatOpenWithOnePackageAreTwoHolders(t *testing.T) {
	// invariant: a package is held by apps, by their catalogue names, never
	// by a package of theirs: two apps that open with the same package each
	// hold it.
	d := &dpkg{held: map[string]string{}}
	a := played(d)
	ctx := context.Background()
	aaa := []catalog.Install{catalog.AptInstall{Packages: []string{"foo"}}}
	bbb := []catalog.Install{catalog.AptInstall{Packages: []string{"foo", "foo-doc"}}}
	if err := a.InstallFor(ctx, word("aaa"), aaa, nil); err != nil {
		t.Fatal(err)
	}
	if err := a.InstallFor(ctx, word("bbb"), bbb, nil); err != nil {
		t.Fatal(err)
	}

	if err := a.RemoveFor(ctx, word("aaa"), aaa, nil); err != nil {
		t.Fatal(err)
	}
	if want := (map[string]string{"foo": "1.0", "foo-doc": "1.0"}); !maps.Equal(d.held, want) {
		t.Errorf("held %v after removing aaa; want %v, bbb's", d.held, want)
	}
}

func TestAnInstallCutShortLeavesTheRecordOfWhatItMayHaveAdded(t *testing.T) {
	// invariant: what an install may add is recorded before apt runs, so an
	// install cut short once dpkg wrote leaves the record of it, and a
	// later removal takes back that alone; a package it never added is let
	// go once apt has run.
	d := &dpkg{held: map[string]string{"foo": "1.0"}, cut: true}
	a := played(d)
	ctx := context.Background()
	foo := []catalog.Install{catalog.AptInstall{Packages: []string{"foo", "foo-extra"}}}

	if err := a.InstallFor(ctx, apply("x"), foo, nil); err == nil {
		t.Fatal("an install cut short returned no error")
	}
	d.cut = false
	if err := a.RemoveFor(ctx, apply("x"), foo, nil); err != nil {
		t.Fatal(err)
	}
	if want := (map[string]string{"foo": "1.0"}); !maps.Equal(d.held, want) {
		t.Errorf("held %v after the removal; want %v, the owner's", d.held, want)
	}

	failed := &dpkg{held: map[string]string{}, fails: true}
	b := played(failed)
	_ = b.InstallFor(ctx, apply("y"), []catalog.Install{catalog.AptInstall{Packages: []string{"bar"}}}, nil)
	failed.fails = false
	if holds, err := b.Holds(ctx, "y"); err != nil || holds {
		t.Errorf("holds %v (%v) after an install that added nothing; want no record", holds, err)
	}
}

func TestAnUnreadableRecordRefusesTheRemovalAndTheInstall(t *testing.T) {
	// invariant: a record left empty or cut mid-line is unreadable, never a
	// package no app holds: a removal or an install that meets it is
	// refused before anything runs as root.
	for name, body := range map[string]string{"empty": "", "cut": "aaa\nbb", "a blank line": "aaa\n\nbbb\n"} {
		d := &dpkg{held: map[string]string{}}
		files := localtest.NewFS()
		a := playedOn(d, files)
		ctx := context.Background()
		aaa := []catalog.Install{catalog.AptInstall{Packages: []string{"libshared"}}}
		if err := a.InstallFor(ctx, word("aaa"), aaa, nil); err != nil {
			t.Fatal(err)
		}
		if err := files.WriteFile(aptStore+"/libshared", []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		d.calls = nil

		var unreadable *apt.RecordError
		if err := a.RemoveFor(ctx, word("aaa"), aaa, nil); !errors.As(err, &unreadable) || rooted(d) || d.held["libshared"] != "1.0" {
			t.Errorf("%s: removal gave %v and ran %q; want it refused, libshared kept", name, err, d.calls)
		}
		if err := a.InstallFor(ctx, word("bbb"), aaa, nil); !errors.As(err, &unreadable) || slices.ContainsFunc(d.calls, func(c string) bool { return strings.Contains(c, " install ") }) {
			t.Errorf("%s: install gave %v and ran %q; want it refused before apt-get install", name, err, d.calls)
		}
	}
}

// value: a disk that keeps every path written, so a test reads where a
// record was written before its rename.
type writes struct {
	local.FS
	paths []string
}

func (w *writes) WriteFile(p string, data []byte, perm fs.FileMode) error {
	w.paths = append(w.paths, p)
	return w.FS.WriteFile(p, data, perm)
}

func TestARecordIsWrittenBesideThenRenamedOverTheOld(t *testing.T) {
	// invariant: a record is never truncated in place: each is written to a
	// new file beside it, then renamed over it, so a write cut short leaves
	// the old record whole.
	files := &writes{FS: localtest.NewFS()}
	a := playedOn(&dpkg{held: map[string]string{}}, files)
	ctx := context.Background()
	for _, app := range []string{"aaa", "bbb"} {
		if err := a.InstallFor(ctx, word(app), []catalog.Install{catalog.AptInstall{Packages: []string{"libshared"}}}, nil); err != nil {
			t.Fatal(err)
		}
	}

	if slices.Contains(files.paths, aptStore+"/libshared") || len(files.paths) != 2 {
		t.Errorf("wrote %q; want each record written beside its file, never over it", files.paths)
	}
	if got, err := files.ReadFile(aptStore + "/libshared"); err != nil || string(got) != "aaa\nbbb\n" {
		t.Errorf("record reads %q (%v); want both holders", got, err)
	}
}

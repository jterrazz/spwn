package apt_test

import (
	"context"
	"errors"
	"fmt"
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
	calls   []string
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

func aptOf(run local.Runner, terminal bool) *apt.Apt {
	return apt.New(run, localtest.NewFS(), aptStore, terminal)
}

func TestInstaller(t *testing.T) {
	contracttest.Installer(t, catalog.AptInstall{Packages: []string{"fd-find"}}, func(_ *testing.T, seed contracttest.Seed) contract.Installer {
		d := &dpkg{held: map[string]string{}, fails: seed.Fails}
		for k, v := range seed.Installed {
			d.held[k] = v
		}
		names := []string{"apt-get", "dpkg-query", "apt-cache", "sudo"}
		if seed.Absent {
			names = []string{"sudo"}
		}
		return aptOf(local.Played(d.play, names...), false)
	})
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
			return a.Install(ctx, []catalog.Install{catalog.AptInstall{Packages: []string{"fd-find"}}}, nil)
		},
		func() error {
			return a.Update(ctx, []catalog.Install{catalog.AptInstall{Packages: []string{"bat"}}}, nil)
		},
		func() error {
			return a.Remove(ctx, []catalog.Install{catalog.AptInstall{Packages: []string{"bat"}}}, nil)
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

	if err := a.Install(context.Background(), []catalog.Install{catalog.AptInstall{Packages: []string{"jq"}}}, nil); err != nil {
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

	err := played(d).Remove(context.Background(), []catalog.Install{catalog.AptInstall{Packages: []string{"libfoo"}}}, nil)

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
	if err := played(d).Install(context.Background(), []catalog.Install{in}, nil); err != nil {
		t.Fatal(err)
	}
	if d.held["tree"] != "1:2.43.0" || d.held["tree-extra"] != "1.0" {
		t.Errorf("held %v; want tree at the version asked, tree-extra at apt's candidate", d.held)
	}
}

func TestAFailedRunSaysAptsVerbAndItsLastWords(t *testing.T) {
	err := played(&dpkg{held: map[string]string{}, fails: true}).Install(context.Background(), []catalog.Install{catalog.AptInstall{Packages: []string{"nope"}}}, nil)

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
	install := func(a *apt.Apt) func(context.Context, []catalog.Install, contract.Logs) error { return a.Install }
	all := []verb{install, func(a *apt.Apt) func(context.Context, []catalog.Install, contract.Logs) error { return a.Update },
		func(a *apt.Apt) func(context.Context, []catalog.Install, contract.Logs) error { return a.Remove }}
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
		func() error { return a.Install(ctx, bat, nil) },
		func() error { return a.Update(ctx, bat, nil) },
		func() error { return a.Remove(ctx, bat, nil) },
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
	_ = a.Remove(ctx, []catalog.Install{catalog.AptInstall{Packages: []string{"bat"}}}, nil)

	if len(reads) != 3 {
		t.Fatalf("read %d times; want dpkg-query, apt-cache policy, apt-get -s remove", len(reads))
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

	if err := a.Install(ctx, foo, nil); err != nil {
		t.Fatal(err)
	}
	if err := a.Remove(ctx, foo, nil); err != nil {
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
		"install": func(a *apt.Apt) error { return a.Install(context.Background(), empty, nil) },
		"update":  func(a *apt.Apt) error { return a.Update(context.Background(), empty, nil) },
		"remove":  func(a *apt.Apt) error { return a.Remove(context.Background(), empty, nil) },
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

	if err := a.Install(ctx, foo, nil); err != nil {
		t.Fatal(err)
	}
	if err := a.Remove(ctx, foo, nil); err != nil {
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
	for _, in := range [][]catalog.Install{one, other} {
		if err := a.Install(ctx, in, nil); err != nil {
			t.Fatal(err)
		}
	}

	if err := a.Remove(ctx, one, nil); err != nil {
		t.Fatal(err)
	}
	if want := (map[string]string{"bbb": "1.0", "libshared": "1.0"}); !maps.Equal(d.held, want) {
		t.Errorf("held %v after removing the first app; want %v", d.held, want)
	}
	if err := a.Remove(ctx, other, nil); err != nil {
		t.Fatal(err)
	}
	if len(d.held) != 0 {
		t.Errorf("held %v after removing both apps; want none", d.held)
	}
}

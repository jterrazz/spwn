package machinefile_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/machinefile"
)

// value: a catalog of two managers, one ranked on macOS; an app with
// two methods on macOS, a version range on the ranked one, and one on Ubuntu; and two apps that require
// each other.
func items() map[string]string {
	return map[string]string{
		"catalog/package-managers/releases.yaml": `item: station.catalog/v1
kind: manager
name: releases
description: release assets
manages: [skill]
priority: { macos: 1 }
methods:
    - { os: any, script: { run: 'true' } }
`,
		"catalog/package-managers/packages.yaml": `item: station.catalog/v1
kind: manager
name: packages
description: system packages
manages: [cask]
methods:
    - { os: any, script: { run: 'true' } }
`,
		"catalog/command-line/fd.yaml": `item: station.catalog/v1
kind: app
name: fd
description: find
methods:
    - { os: macos, cask: fd }
    - { os: macos, version: '>=26', skill: { repo: owner/fd, name: fd } }
    - { os: ubuntu, script: { run: 'true' } }
`,
		"catalog/command-line/left.yaml": `item: station.catalog/v1
kind: app
name: left
description: one side
requires: [right]
methods:
    - { os: any, script: { run: 'true' } }
`,
		"catalog/command-line/right.yaml": `item: station.catalog/v1
kind: app
name: right
description: the other side
requires: [left]
methods:
    - { os: any, script: { run: 'true' } }
`,
	}
}

func library(t *testing.T, user, builtin map[string]string) machinefile.Library {
	t.Helper()
	tree := func(layer machinefile.Layer, root string, files map[string]string) machinefile.Tree {
		tr := machinefile.Tree{Layer: layer, Root: root}
		for p, body := range files {
			tr.Files = append(tr.Files, machinefile.File{Path: p, Body: []byte(body)})
		}
		return tr
	}
	all := items()
	for p, b := range builtin {
		all[p] = b
	}
	lib, err := machinefile.Load([]machinefile.Tree{tree(machinefile.User, "~/.config/station", user), tree(machinefile.Builtin, "built-in", all)})
	if err != nil {
		t.Fatal(err)
	}
	return lib
}

func machine(name, system, apps string) string {
	return "machinefile: station.machinefile/v1\nname: " + name + "\nmachine: { kind: workspace, system: " + system + " }\napps: " + apps + "\n"
}

func methodOf(t *testing.T, lib machinefile.Library, name, item string) string {
	t.Helper()
	r, err := machinefile.Resolve(lib, name, machinefile.Machine{}, machinefile.Lock{})
	if err != nil {
		t.Fatal(err)
	}
	i := slices.IndexFunc(r.Steps, func(s machinefile.Step) bool { return s.Item == item })
	if i < 0 {
		t.Fatalf("%s holds no step %s: %+v", name, item, r.Steps)
	}
	return r.Steps[i].Method
}

func TestAMethodIsChosenBySelectorThenPriorityThenPrefer(t *testing.T) {
	cases := []struct{ about, file, want string }{
		{"a version range below the system's is no match", machine("a", "{ os: macos, version: '25' }", "[fd]"), "cask"},
		{"a ranked manager before an unranked one, whatever the file's order", machine("a", "{ os: macos, version: '26' }", "[fd]"), "skill"},
		{"prefer before the rank", machine("a", "{ os: macos, version: '26' }", "[fd]") + "prefer: [cask]\n", "cask"},
		{"a manager asked by the app", machine("a", "{ os: macos, version: '26' }", "[{ name: fd, manager: cask }]"), "cask"},
		{"another system, its own method, named as it runs", machine("a", "{ os: ubuntu }", "[fd]"), "shell"},
	}
	for _, c := range cases {
		lib := library(t, map[string]string{"machinefiles/a.yaml": c.file}, nil)
		if got := methodOf(t, lib, "a", "fd"); got != c.want {
			t.Errorf("%s: chose %s, want %s", c.about, got, c.want)
		}
	}
}

func TestAVersionRangeReadsEachComparison(t *testing.T) {
	for _, c := range []struct {
		version, ranges string
		in              bool
	}{
		{"24.04", ">=24.04", true}, {"22.04", ">=24.04", false}, {"26.1", ">=14, <27", true}, {"27", ">=14, <27", false},
		{"15.2", "15", true}, {"150", "15", false}, {"", ">=1", false},
	} {
		if got := catalog.InRange(c.version, c.ranges); got != c.in {
			t.Errorf("%q in %q: %v, want %v", c.version, c.ranges, got, c.in)
		}
	}
}

func TestALaterFileReplacesAndAnOverlayAddsAndRemoves(t *testing.T) {
	base := "machinefile: station.machinefile/v1\nname: base\napps: [packages, { name: fd, version: '1' }]\nvm: { network: nat, resources: { cpus: 2 } }\n"
	top := machine("top", "{ os: macos, version: '26' }", "[{ name: fd, manager: cask }]") + "extends: [base]\nvm: { resources: { cpus: 8 } }\n"
	overlay := "machinefile: station.machinefile/v1\noverrides: top\napps: { add: [releases], remove: [packages] }\n"
	lib := library(t, map[string]string{"machinefiles/top.yaml": top, "machinefiles/top.local.yaml": overlay}, map[string]string{"machinefiles/base.yaml": base})
	r, err := machinefile.Resolve(lib, "top", machinefile.Machine{}, machinefile.Lock{})
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]machinefile.Value{}
	for _, v := range r.Values {
		got[v.Key] = v
	}
	if v := got["vm.resources.cpus"]; v.Text != "8" || v.Origin.Name != "top" {
		t.Errorf("cpus %+v, want top's 8", v)
	}
	if v := got["vm.network"]; v.Text != "nat" || v.Origin.Layer != machinefile.Builtin {
		t.Errorf("network %+v, want base's nat", v)
	}
	if v := got["apps.fd"]; v.Text != "listed" || v.Origin.Name != "top" {
		t.Errorf("fd %+v: top's entry replaces base's whole, its version gone", v)
	}
	if v := got["apps.packages"]; v.Text != "the manager of fd" {
		t.Errorf("packages %+v: the overlay removed it as listed, and fd's manager brings it back", v)
	}
	if v := got["apps.releases"]; v.Origin.Layer != machinefile.Overlay {
		t.Errorf("releases %+v, want the overlay's", v)
	}
}

func TestASameNameFileOfAStrongerLayerReplacesTheWeakerWhole(t *testing.T) {
	lib := library(t, map[string]string{"machinefiles/a.yaml": machine("a", "{ os: macos }", "[fd]")},
		map[string]string{"machinefiles/a.yaml": machine("a", "{ os: ubuntu, version: '24.04' }", "[left]")})
	if got := methodOf(t, lib, "a", "fd"); got != "cask" {
		t.Errorf("the owner's a resolved fd by %s, want its macos cask", got)
	}
}

func TestACycleIsPrintedAroundItsLoop(t *testing.T) {
	ext := func(name, extends string) string {
		return "machinefile: station.machinefile/v1\nname: " + name + "\nmachine: { kind: workspace }\nextends: [" + extends + "]\n"
	}
	cases := []struct {
		about string
		files map[string]string
		want  string
	}{
		{"extends", map[string]string{"machinefiles/a.yaml": ext("a", "b"), "machinefiles/b.yaml": ext("b", "c"), "machinefiles/c.yaml": ext("c", "b")}, "b extends c extends b"},
		{"requires", map[string]string{"machinefiles/a.yaml": machine("a", "{ os: macos }", "[left]")}, "left needs right needs left"},
	}
	for _, c := range cases {
		_, err := machinefile.Resolve(library(t, c.files, nil), "a", machinefile.Machine{}, machinefile.Lock{})
		var cycle *machinefile.CycleError
		if !errors.As(err, &cycle) || cycle.Said() != c.want {
			t.Errorf("%s: %v, want the cycle %s", c.about, err, c.want)
		}
	}
}

func TestAccountsAndPortsAreReplacedWhole(t *testing.T) {
	base := machine("a", "{ os: macos }", "[]") + "accounts: { share: [{ account: claude, for: 7d }, { account: github }], keep: [github] }\nports: { open: 3000-9999, to: private, except: [22] }\n"
	overlay := "machinefile: station.machinefile/v1\noverrides: a\naccounts: { share: [{ account: claude, for: 1d }] }\nports: { open: 8080, to: local }\n"
	lib := library(t, map[string]string{"machinefiles/a.yaml": base, "machinefiles/a.local.yaml": overlay}, nil)
	r, err := machinefile.Resolve(lib, "a", machinefile.Machine{}, machinefile.Lock{})
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, v := range r.Values {
		if strings.HasPrefix(v.Key, "accounts.") || strings.HasPrefix(v.Key, "ports.") {
			got = append(got, v.Key+"="+v.Text+"@"+string(v.Origin.Layer))
		}
	}
	want := []string{"accounts.share.claude=1d@overlay", "ports.open=8080@overlay", "ports.to=local@overlay"}
	if !slices.Equal(got, want) {
		t.Errorf("read %v, want the overlay's accounts and ports alone: %v", got, want)
	}
}

func TestAFileThatIsNoYAMLIsNotRead(t *testing.T) {
	files := map[string]string{
		"machinefiles/a.yaml":       machine("a", "{ os: macos }", "[fd]"),
		"machinefiles/a.lock":       "not yaml at all: [",
		"machinefiles/README.md":    "# my machines",
		"catalog/command-line/NOTE": "a note",
	}
	if got := methodOf(t, library(t, files, nil), "a", "fd"); got != "cask" {
		t.Errorf("chose %s", got)
	}
	for _, p := range []string{"machinefiles/b.yml", "catalog/command-line/jq.yml"} {
		_, err := machinefile.Load([]machinefile.Tree{{Layer: machinefile.User, Root: "~/.config/station", Files: []machinefile.File{{Path: p, Body: []byte("x: 1\n")}}}})
		var refused *machinefile.RefusedFileError
		if !errors.As(err, &refused) {
			t.Errorf("%s read with %v, want it refused: a .yml file was meant to be read", p, err)
		}
	}
}

func TestARefusedEntryIsNamedAtItsLine(t *testing.T) {
	cases := []struct{ about, file, want string }{
		{"an unknown app", machine("a", "{ os: macos }", "\n    - fd\n    - nope"), "nope (~/.config/station/machinefiles/a.yaml:6)"},
		{"a bad value", machine("a", "{ os: macos }", "[{ name: fd, options: { tint: blue } }]"), "fd's tint: no option of fd (~/.config/station/machinefiles/a.yaml:4)"},
		{"no method", machine("a", "{ os: debian, version: '12' }", "[{ name: fd, manager: skill }]"), "(listed, ~/.config/station/machinefiles/a.yaml:4)"},
		{"an item baked that the catalog lacks", machine("a", "{ os: macos }", "[]") + "vm: { image: { bake: [fd, nope] } }\n", "nope (~/.config/station/machinefiles/a.yaml)"},
	}
	for _, c := range cases {
		_, err := machinefile.Resolve(library(t, map[string]string{"machinefiles/a.yaml": c.file}, nil), "a", machinefile.Machine{}, machinefile.Lock{})
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v, want it saying %q", c.about, err, c.want)
		}
	}
}

func TestAnOptionalAppWhoseManagerHasNoMethodSaysWhatItHas(t *testing.T) {
	file := machine("a", "{ os: macos }", "[{ name: fd, manager: skill, optional: true }]")
	r, err := machinefile.Resolve(library(t, map[string]string{"machinefiles/a.yaml": file}, nil), "a", machinefile.Machine{}, machinefile.Lock{})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Skipped) != 1 || r.Skipped[0].Why != "optional, no method by skill for macos, where it has cask" {
		t.Errorf("skipped %+v, want fd skipped naming its cask", r.Skipped)
	}
}

func TestAnItemProvidedByAnotherBringsItInFirst(t *testing.T) {
	// invariant: provided-by is a link like requires: its provider is
	// resolved into the plan and runs before it.
	provided := map[string]string{
		"catalog/ai-agents/core.yaml":    "item: station.catalog/v1\nkind: app\nname: core\ndescription: the engine\nprovides: [engine]\nmethods:\n    - { os: any, provided-by: desktop }\n",
		"catalog/ai-agents/desktop.yaml": "item: station.catalog/v1\nkind: app\nname: desktop\ndescription: the app\nmethods:\n    - { os: any, script: { run: 'true' } }\n",
	}
	lib := library(t, map[string]string{"machinefiles/a.yaml": machine("a", "{ os: macos }", "[core]")}, provided)
	r, err := machinefile.Resolve(lib, "a", machinefile.Machine{}, machinefile.Lock{})
	if err != nil {
		t.Fatal(err)
	}
	order := make([]string, 0, len(r.Steps))
	for _, s := range r.Steps {
		order = append(order, s.Item+" "+s.Why)
	}
	if want := []string{"desktop the provider of core", "core listed"}; !slices.Equal(order, want) {
		t.Errorf("steps %q; want %q", order, want)
	}
}

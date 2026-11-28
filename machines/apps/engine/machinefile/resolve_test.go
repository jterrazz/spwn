package machinefile_test

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/machinefile"
)

// value: a catalog of two managers, one ranked on macOS; an app with
// two methods on macOS, a version range on the ranked one, and one on Ubuntu; and two apps that require
// each other.
func packages() map[string]string {
	return map[string]string{
		"catalog/package-managers/releases.yaml": `package: station.package/v1
kind: manager
name: releases
description: release assets
manages: [skill]
priority: { macos: 1 }
methods:
    - { os: any, script: { run: 'true' } }
`,
		"catalog/package-managers/packages.yaml": `package: station.package/v1
kind: manager
name: packages
description: system packages
manages: [cask]
methods:
    - { os: any, script: { run: 'true' } }
`,
		"catalog/command-line/fd.yaml": `package: station.package/v1
kind: app
name: fd
description: find
methods:
    - { os: macos, cask: fd }
    - { os: macos, version: '>=26', skill: { repo: owner/fd, name: fd } }
    - { os: ubuntu, script: { run: 'true' } }
`,
		"catalog/command-line/left.yaml": `package: station.package/v1
kind: app
name: left
description: one side
requires: [right]
methods:
    - { os: any, script: { run: 'true' } }
`,
		"catalog/command-line/right.yaml": `package: station.package/v1
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
	tree := func(origin machinefile.OriginKind, root string, files map[string]string) machinefile.Tree {
		tr := machinefile.Tree{Kind: origin, Root: root}
		for p, body := range files {
			tr.Files = append(tr.Files, machinefile.File{Path: p, Body: []byte(body)})
		}
		return tr
	}
	all := packages()
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

func methodOf(t *testing.T, lib machinefile.Library, name, pkg string) string {
	t.Helper()
	r, err := machinefile.Resolve(lib, name, machinefile.Machine{}, machinefile.Lock{})
	if err != nil {
		t.Fatal(err)
	}
	i := slices.IndexFunc(r.Steps, func(s machinefile.Step) bool { return s.Package == pkg })
	if i < 0 {
		t.Fatalf("%s holds no step %s: %+v", name, pkg, r.Steps)
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
		lib := library(t, map[string]string{"machinefiles/a.machine.yaml": c.file}, nil)
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
	lib := library(t, map[string]string{"machinefiles/top.machine.yaml": top, "machinefiles/top.local.yaml": overlay}, map[string]string{"machinefiles/base.machine.yaml": base})
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
	if v := got["vm.network"]; v.Text != "nat" || v.Origin.Kind != machinefile.Builtin {
		t.Errorf("network %+v, want base's nat", v)
	}
	if v := got["apps.fd"]; v.Text != "listed" || v.Origin.Name != "top" {
		t.Errorf("fd %+v: top's entry replaces base's whole, its version gone", v)
	}
	if v := got["apps.packages"]; v.Text != "the manager of fd" {
		t.Errorf("packages %+v: the overlay removed it as listed, and fd's manager brings it back", v)
	}
	if v := got["apps.releases"]; v.Origin.Kind != machinefile.Overlay {
		t.Errorf("releases %+v, want the overlay's", v)
	}
}

func TestASameNameFileOfAStrongerLayerReplacesTheWeakerWhole(t *testing.T) {
	lib := library(t, map[string]string{"machinefiles/a.machine.yaml": machine("a", "{ os: macos }", "[fd]")},
		map[string]string{"machinefiles/a.machine.yaml": machine("a", "{ os: ubuntu, version: '24.04' }", "[left]")})
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
		{"extends", map[string]string{"machinefiles/a.machine.yaml": ext("a", "b"), "machinefiles/b.machine.yaml": ext("b", "c"), "machinefiles/c.machine.yaml": ext("c", "b")}, "b extends c extends b"},
		{"requires", map[string]string{"machinefiles/a.machine.yaml": machine("a", "{ os: macos }", "[left]")}, "left needs right needs left"},
	}
	for _, c := range cases {
		_, err := machinefile.Resolve(library(t, c.files, nil), "a", machinefile.Machine{}, machinefile.Lock{})
		var cycle *machinefile.CycleError
		if !errors.As(err, &cycle) || cycle.Said() != c.want {
			t.Errorf("%s: %v, want the cycle %s", c.about, err, c.want)
		}
	}
}

// invariant: a Machinefile's accounts join those it extends and its ports
// replace theirs whole; an overlay's change them key by key, audiences
// port by port, and a merge that cannot be read is refused at the overlay.
func TestAnOverlayChangesAccountsAndPortsKeyByKey(t *testing.T) {
	base := "machinefile: station.machinefile/v1\nname: base\naccounts: { signins: [wrangler] }\nports: { open: [1, 2], to: public }\n"
	a := machine("a", "{ os: macos }", "[]") + "extends: [base]\naccounts: { signins: [gh], grants: [claude] }\nports: { open: 3000-9999, to: private, except: [22], audiences: { 5432: local } }\n"
	overlay := "machinefile: station.machinefile/v1\noverrides: a\naccounts: { grants: [claude] }\nports: { except: [3306], audiences: { 8080: public } }\n"
	lib := library(t, map[string]string{"machinefiles/a.machine.yaml": a, "machinefiles/a.local.yaml": overlay, "machinefiles/base.machine.yaml": base}, nil)
	r, err := machinefile.Resolve(lib, "a", machinefile.Machine{}, machinefile.Lock{})
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, v := range r.Values {
		if strings.HasPrefix(v.Key, "accounts.") || strings.HasPrefix(v.Key, "ports.") {
			got = append(got, v.Key+"="+v.Text+"@"+string(v.Origin.Kind))
		}
	}
	want := []string{"accounts.signins.wrangler=signs in@user", "accounts.signins.gh=signs in@user", "accounts.grants.claude=granted@overlay", "ports.open=3000-9999@user", "ports.to=private@user", "ports.audiences.5432=local@user", "ports.except=3306@overlay", "ports.audiences.8080=public@overlay"}
	if !slices.Equal(got, want) {
		t.Errorf("read %v\nwant a's whole, the overlay's keys over it: %v", got, want)
	}
	given, err := machinefile.GivenOf(lib, "a", "")
	if err != nil || given.Ports.To != catalog.AudiencePrivate || !slices.Equal(given.Ports.Except, []int{3306}) || len(given.Ports.Audiences) != 2 || fmt.Sprint(given.SignIns, given.Grants) != "[wrangler gh] [claude]" {
		t.Errorf("given %+v %+v, %v; want a's ports with the overlay's except and audiences, the chain's sign-ins kept", given, given.Ports, err)
	}
	clash := "machinefile: station.machinefile/v1\noverrides: a\naccounts: { grants: [gh] }\nports: { to: public }\n"
	lib = library(t, map[string]string{"machinefiles/a.machine.yaml": a, "machinefiles/a.local.yaml": clash, "machinefiles/base.machine.yaml": base}, nil)
	var refused *machinefile.RefusedFileError
	if _, err := machinefile.Resolve(lib, "a", machinefile.Machine{}, machinefile.Lock{}); !errors.As(err, &refused) || !strings.Contains(err.Error(), "a.local.yaml: its accounts with those of a's chain: gh is both signed in and granted") {
		t.Errorf("a grant the chain signs in to reads %v; want the overlay refused", err)
	}
	if _, err := machinefile.GivenOf(lib, "a", ""); !errors.As(err, &refused) {
		t.Errorf("given of a refused merge reads %v; want the refusal", err)
	}
	wide := "machinefile: station.machinefile/v1\noverrides: a\nports: { to: public }\n"
	lib = library(t, map[string]string{"machinefiles/a.machine.yaml": a, "machinefiles/a.local.yaml": wide, "machinefiles/base.machine.yaml": base}, nil)
	if _, err := machinefile.Resolve(lib, "a", machinefile.Machine{}, machinefile.Lock{}); !errors.As(err, &refused) || !strings.Contains(err.Error(), "a.local.yaml: its ports with those of a's chain: public opens ports one by one") {
		t.Errorf("an overlay making a range public reads %v; want the overlay refused", err)
	}
}

func TestAMachinesOwnChangesJoinTheChainOfItsMachinefileAlone(t *testing.T) {
	dev := machine("dev", "{ os: macos }", "[]") + "vm: { image: { from: base-ubuntu } }\nports: { open: 3000-3999, to: private }\n"
	own := "machinefile: station.machinefile/v1\noverrides: dev\napps: { add: [fd] }\nports: { except: [3306] }\n"
	lib := library(t, map[string]string{"machinefiles/dev.machine.yaml": dev, "machinefiles/linux-1.local.yaml": own, "machinefiles/web.machine.yaml": machine("web", "{ os: macos }", "[]")}, nil)
	listed := func(vm, file string) []string {
		t.Helper()
		r, err := machinefile.Resolve(lib, file, machinefile.Machine{Name: vm}, machinefile.Lock{})
		if err != nil {
			t.Fatal(err)
		}
		var got []string
		for _, v := range r.Values {
			if v.Origin.Kind == machinefile.Overlay {
				got = append(got, v.Key+"@"+v.Origin.Name)
			}
		}
		return got
	}
	if got, want := listed("linux-1", "dev"), []string{"apps.fd@dev", "ports.except@dev"}; !slices.Equal(got, want) {
		t.Errorf("linux-1 resolves dev to %v, want its own changes: %v", got, want)
	}
	for _, read := range [][2]string{{"agent-7f", "dev"}, {"", "dev"}, {"linux-1", "web"}} {
		if got := listed(read[0], read[1]); len(got) > 0 {
			t.Errorf("%q resolves %s to %v: linux-1's own changes are linux-1's alone", read[0], read[1], got)
		}
	}
	sameName := library(t, map[string]string{"machinefiles/dev.machine.yaml": dev, "machinefiles/linux-1.machine.yaml": machine("linux-1", "{ os: macos }", "[]"), "machinefiles/linux-1.local.yaml": own}, nil)
	if r, err := machinefile.Resolve(sameName, "linux-1", machinefile.Machine{}, machinefile.Lock{}); err != nil {
		t.Fatal(err)
	} else {
		for _, v := range r.Values {
			if v.Origin.Kind == machinefile.Overlay {
				t.Errorf("machinefile linux-1 reads %s from an overlay that overrides dev: an overlay is file-wide only for the file it overrides", v.Key)
			}
		}
	}
	given, err := machinefile.GivenOf(lib, "dev", "linux-1")
	if err != nil || !slices.Equal(given.Ports.Except, []int{3306}) {
		t.Errorf("given to linux-1 %+v, %v; want its own except", given.Ports, err)
	}
	if given, err := machinefile.GivenOf(lib, "dev", "agent-7f"); err != nil || len(given.Ports.Except) != 0 {
		t.Errorf("given to agent-7f %+v, %v; want dev's alone", given.Ports, err)
	}
	if tpl, err := machinefile.TemplateOf(lib, "dev", nil); err != nil || len(tpl.Files) != 1 {
		t.Errorf("dev's template carries %v, %v; want dev alone: a machine's own changes stay on its Mac", tpl.Files, err)
	}
}

func TestAFileThatIsNoYAMLIsNotRead(t *testing.T) {
	files := map[string]string{
		"machinefiles/a.machine.yaml": machine("a", "{ os: macos }", "[fd]"),
		"machinefiles/a.lock":         "not yaml at all: [",
		"machinefiles/README.md":      "# my machines",
		"catalog/command-line/NOTE":   "a note",
	}
	if got := methodOf(t, library(t, files, nil), "a", "fd"); got != "cask" {
		t.Errorf("chose %s", got)
	}
	for _, p := range []string{"machinefiles/b.yml", "catalog/command-line/jq.yml"} {
		_, err := machinefile.Load([]machinefile.Tree{{Kind: machinefile.User, Root: "~/.config/station", Files: []machinefile.File{{Path: p, Body: []byte("x: 1\n")}}}})
		var refused *machinefile.RefusedFileError
		if !errors.As(err, &refused) {
			t.Errorf("%s read with %v, want it refused: a .yml file was meant to be read", p, err)
		}
	}
}

func TestARefusedEntryIsNamedAtItsLine(t *testing.T) {
	cases := []struct{ about, file, want string }{
		{"an unknown app", machine("a", "{ os: macos }", "\n    - fd\n    - nope"), "nope (~/.config/station/machinefiles/a.machine.yaml:6)"},
		{"a bad value", machine("a", "{ os: macos }", "[{ name: fd, options: { tint: blue } }]"), "fd's tint: no option of fd (~/.config/station/machinefiles/a.machine.yaml:4)"},
		{"no method", machine("a", "{ os: debian, version: '12' }", "[{ name: fd, manager: skill }]"), "(listed, ~/.config/station/machinefiles/a.machine.yaml:4)"},
		{"a package baked that the catalog lacks", machine("a", "{ os: macos }", "[]") + "vm: { image: { bake: [fd, nope] } }\n", "nope (~/.config/station/machinefiles/a.machine.yaml)"},
	}
	for _, c := range cases {
		_, err := machinefile.Resolve(library(t, map[string]string{"machinefiles/a.machine.yaml": c.file}, nil), "a", machinefile.Machine{}, machinefile.Lock{})
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v, want it saying %q", c.about, err, c.want)
		}
	}
}

func TestAnOptionalAppWhoseManagerHasNoMethodSaysWhatItHas(t *testing.T) {
	file := machine("a", "{ os: macos }", "[{ name: fd, manager: skill, optional: true }]")
	r, err := machinefile.Resolve(library(t, map[string]string{"machinefiles/a.machine.yaml": file}, nil), "a", machinefile.Machine{}, machinefile.Lock{})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Skipped) != 1 || r.Skipped[0].Why != "optional, no method by skill for macos, where it has cask" {
		t.Errorf("skipped %+v, want fd skipped naming its cask", r.Skipped)
	}
}

func TestAStepSaysItsPackageIsOptional(t *testing.T) {
	for _, c := range []struct {
		apps     string
		optional bool
	}{{"[{ name: fd, optional: true }]", true}, {"[fd]", false}} {
		file := machine("a", "{ os: macos }", c.apps)
		r, err := machinefile.Resolve(library(t, map[string]string{"machinefiles/a.machine.yaml": file}, nil), "a", machinefile.Machine{}, machinefile.Lock{})
		if err != nil {
			t.Fatal(err)
		}
		i := slices.IndexFunc(r.Steps, func(s machinefile.Step) bool { return s.Package == "fd" })
		if i < 0 || r.Steps[i].Optional != c.optional {
			t.Errorf("%s: steps %+v, want fd with Optional %v", c.apps, r.Steps, c.optional)
		}
	}
}

func TestAPackageProvidedByAnotherBringsItInFirst(t *testing.T) {
	// invariant: provided-by is a link like requires: its provider is
	// resolved into the plan and runs before it.
	provided := map[string]string{
		"catalog/ai-agents/core.yaml":    "package: station.package/v1\nkind: app\nname: core\ndescription: the engine\nprovides: [engine]\nmethods:\n    - { os: any, provided-by: desktop }\n",
		"catalog/ai-agents/desktop.yaml": "package: station.package/v1\nkind: app\nname: desktop\ndescription: the app\nmethods:\n    - { os: any, script: { run: 'true' } }\n",
	}
	lib := library(t, map[string]string{"machinefiles/a.machine.yaml": machine("a", "{ os: macos }", "[core]")}, provided)
	r, err := machinefile.Resolve(lib, "a", machinefile.Machine{}, machinefile.Lock{})
	if err != nil {
		t.Fatal(err)
	}
	order := make([]string, 0, len(r.Steps))
	for _, s := range r.Steps {
		order = append(order, s.Package+" "+s.Why)
	}
	if want := []string{"desktop the provider of core", "core listed"}; !slices.Equal(order, want) {
		t.Errorf("steps %q; want %q", order, want)
	}
}

func TestATokenInAnOptionOrARunLineIsWithheldInTheValues(t *testing.T) {
	token := "sk-" + strings.Repeat("station-", 4) + "key"
	served := `package: station.package/v1
kind: app
name: served
description: told where to call
options:
    - { name: endpoint, label: Endpoint, type: string, required: true }
methods:
    - { os: any, script: { run: 'true' } }
`
	file := "machinefile: station.machinefile/v1\nname: top\nmachine: { kind: workspace, system: { os: ubuntu } }\napps: [{ name: served, options: { endpoint: " + token + " } }]\nrun:\n  boot: |\n    export KEY=" + token + "\n    make install\n"
	lib := library(t, map[string]string{"machinefiles/top.machine.yaml": file}, map[string]string{"catalog/command-line/served.yaml": served})
	r, err := machinefile.Resolve(lib, "top", machinefile.Machine{}, machinefile.Lock{})
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]machinefile.Value{}
	for _, v := range r.Values {
		got[v.Key] = v
	}
	if v := got["apps.served.endpoint"]; v.Text != "withheld" || !v.Withheld {
		t.Errorf("an option holding a token reads %+v, want it withheld whole", v)
	}
	if v := got["run.boot"]; v.Text != "export KEY=withheld; make install" || !v.Withheld {
		t.Errorf("a run line holding a token reads %+v, want the token withheld and the rest kept", v)
	}
	for _, v := range r.Values {
		if strings.Contains(v.Text, token) {
			t.Errorf("%s carries the token: %q", v.Key, v.Text)
		}
	}
}

func TestAMixinAnswersItsOwnChainsValuesAndIsNeverResolved(t *testing.T) {
	base := "machinefile: station.machinefile/v1\nname: base\napps: [fd, left]\nvm: { network: nat }\naccounts: { signins: [gh] }\n"
	top := "machinefile: station.machinefile/v1\nname: top\nextends: [base]\napps: [releases]\nvm: { resources: { cpus: 4 } }\n"
	lib := library(t, map[string]string{"machinefiles/top.machine.yaml": top}, map[string]string{"machinefiles/base.machine.yaml": base})
	if _, err := machinefile.Resolve(lib, "top", machinefile.Machine{}, machinefile.Lock{}); !errors.As(err, new(*machinefile.MixinError)) {
		t.Fatalf("resolve of a mixin: %v, want the mixin refusal", err)
	}
	r, err := machinefile.Own(lib, "top")
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]machinefile.Value{}
	for _, v := range r.Values {
		got[v.Key] = v
	}
	if v := got["vm.network"]; v.Text != "nat" || v.Origin.Name != "base" {
		t.Errorf("network %+v, want the extended mixin's nat", v)
	}
	if v := got["vm.resources.cpus"]; v.Text != "4" || v.Origin.Name != "top" {
		t.Errorf("cpus %+v, want its own 4", v)
	}
	if v := got["apps.fd"]; v.Text != "listed" || v.Origin.Name != "base" {
		t.Errorf("fd %+v, want listed by base", v)
	}
	if v := got["apps.releases"]; v.Text != "listed" || v.Origin.Name != "top" {
		t.Errorf("releases %+v, want listed by top", v)
	}
	if v := got["apps.left"]; v.Text != "listed" || v.Origin.Name != "base" {
		t.Errorf("left %+v, want listed by base though no catalog step holds it", v)
	}
	if v := got["accounts.signins.gh"]; v.Text != "signs in" {
		t.Errorf("the sign-in of gh %+v, want signs in", v)
	}
	if len(r.Steps) != 0 {
		t.Errorf("a mixin holds steps %+v, want none judged", r.Steps)
	}
}

func TestAMixinsOwnValuesLeaveWithAnyTokenWithheld(t *testing.T) {
	token := "sk-" + strings.Repeat("station-", 4) + "key"
	top := "machinefile: station.machinefile/v1\nname: top\napps: [{ name: served, options: { endpoint: " + token + " } }]\nsettings: { editor: { api: " + token + " } }\nrun:\n  boot: export KEY=" + token + "\n"
	lib := library(t, map[string]string{"machinefiles/top.machine.yaml": top}, nil)
	r, err := machinefile.Own(lib, "top")
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]machinefile.Value{}
	for _, v := range r.Values {
		got[v.Key] = v
	}
	for _, key := range []string{"apps.served.endpoint", "settings.editor.api", "run.boot"} {
		if v := got[key]; !v.Withheld || strings.Contains(v.Text, token) {
			t.Errorf("%s reads %+v, want the token withheld", key, v)
		}
	}
}

func TestAMachinefileInstallsWhatItListsAndWhatThatRequires(t *testing.T) {
	serve := `package: station.package/v1
kind: app
name: serve
description: a dev server
requires: [fd]
ports:
    - { port: 5173, label: Serve, http: true, env: { SERVE_URL: '{url}' } }
methods:
    - { os: any, script: { run: 'true' } }
`
	lib := library(t, map[string]string{
		"machinefiles/web.machine.yaml":   machine("web", "{ os: ubuntu }", "[serve]"),
		"machinefiles/plain.machine.yaml": machine("plain", "{ os: ubuntu }", "[fd]"),
	}, map[string]string{"catalog/command-line/serve.yaml": serve})
	got, err := machinefile.Installs(lib, "web", "web-1")
	slices.Sort(got)
	if err != nil || !slices.Equal(got, []string{"fd", "serve"}) {
		t.Errorf("web installs %v, %v; want fd and serve", got, err)
	}
	if got, err := machinefile.Installs(lib, "plain", "plain-1"); err != nil || !slices.Equal(got, []string{"fd"}) {
		t.Errorf("plain installs %v, %v; want fd alone", got, err)
	}
	if got, err := machinefile.Installs(lib, "linux-1", "linux-1"); err != nil || got != nil {
		t.Errorf("a VM with no Machinefile installs %v, %v; want none", got, err)
	}
}

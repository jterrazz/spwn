package catalog_test

import (
	"errors"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/jterrazz/studio-station/src/catalog"
)

const gh = `item: station.catalog/v1
kind: app
name: gh
description: GitHub CLI
provides: [gh]
methods:
    - os: macos
      brew: gh
`

func decoded(t *testing.T, body string) error {
	t.Helper()
	_, err := catalog.Decode("items/git/gh.yaml", []byte(body))
	return err
}

func refusedFor(t *testing.T, body, want string) {
	t.Helper()
	err := decoded(t, body)
	var file *catalog.FileError
	if !errors.As(err, &file) || !strings.Contains(err.Error(), want) {
		t.Errorf("decoded with %v, want a refusal saying %q", err, want)
	}
}

func TestAnItemFileDecodesIntoItsItem(t *testing.T) {
	it, err := catalog.Decode("items/git/gh.yaml", []byte(gh))
	if err != nil {
		t.Fatal(err)
	}
	if it.Kind != catalog.KindApp || it.Methods[0].Brew != "gh" {
		t.Errorf("decoded %+v", it)
	}
}

func TestAnUnknownKeyIsRefusedWithItsLine(t *testing.T) {
	refusedFor(t, gh+"tint: blue\n", "line 9: field tint not found")
	refusedFor(t, strings.Replace(gh, "brew: gh", "brew: gh\n      formula: gh", 1), "field formula not found")
}

func TestAnAnchorAnAliasOrAMergeKeyIsRefused(t *testing.T) {
	refusedFor(t, strings.Replace(gh, "provides: [gh]", "provides: &cmds [gh]\nrequires: *cmds", 1), "an anchor (&cmds)")
	refusedFor(t, strings.Replace(gh, "provides: [gh]", "provides: [gh]\n<<: {replaces: git}", 1), "a merge key (<<)")
}

func TestAYAML11ScalarIsRefusedForItsTwoReadings(t *testing.T) {
	setting := "item: station.catalog/v1\nkind: setting\nname: gh\ndescription: d\nreversible: yes\nmethods:\n    - os: any\n      builtin: dns\n"
	refusedFor(t, setting, `"yes" reads differently in YAML 1.1`)
	copied := "item: station.catalog/v1\nkind: setting\nname: gh\ndescription: d\nmethods:\n    - os: any\n      file:\n        copy:\n            - {from: a, to: b, permissions: 0755}\n"
	refusedFor(t, copied, `"0755" reads differently in YAML 1.1`)
	if err := decoded(t, strings.Replace(copied, "0755", "0o755", 1)); err != nil {
		t.Errorf("a YAML 1.2 octal is refused: %v", err)
	}
	if err := decoded(t, strings.Replace(gh, "GitHub CLI", `"yes"`, 1)); err != nil {
		t.Errorf("a quoted yes is refused: %v", err)
	}
}

func TestANewerSchemaIsRefusedWithUpdateStation(t *testing.T) {
	err := decoded(t, strings.Replace(gh, "station.catalog/v1", "station.catalog/v2", 1))
	var newer *catalog.NewerSchemaError
	if !errors.As(err, &newer) || !strings.HasSuffix(err.Error(), ": update station") {
		t.Errorf("a v2 item decoded with %v, want a refusal ending update station", err)
	}
	refusedFor(t, strings.Replace(gh, "station.catalog/v1", "station.machinefile/v1", 1), `names the schema "station.machinefile/v1"`)
	ahead := strings.Replace(strings.Replace(gh, "station.catalog/v1", "station.catalog/v2", 1), "provides: [gh]", "provides: &c [gh]\ntint: blue", 1)
	if err := decoded(t, ahead); !errors.As(err, &newer) {
		t.Errorf("a v2 item with keys v1 refuses decoded with %v, want update station first", err)
	}
	refusedFor(t, strings.Replace(gh, "item: station.catalog/v1\n", "", 1), "names no schema")
}

func TestAnItemIsJudgedAlone(t *testing.T) {
	for body, want := range map[string]string{
		strings.Replace(gh, "name: gh", "name: hub", 1):                                                                `names "hub", its file "gh"`,
		strings.Replace(gh, "kind: app", "kind: tool", 1):                                                              `kind "tool" is none of`,
		strings.Replace(gh, "provides: [gh]", "options: []\nguided: true", 1):                                          "`guided` is not a key of a app",
		strings.Replace(gh, "brew: gh", "brew: gh\n      cask: gh", 1):                                                 "names 2 managers (brew, cask), not one",
		strings.Replace(gh, "os: macos", "family: linux", 1):                                                           "brew does not run on linux",
		strings.Replace(gh, "os: macos", "os: macos\n      family: linux", 1):                                          "names an os and a family",
		strings.Replace(gh, "    - os: macos\n", "    - machine: computer\n", 1):                                       "names no os nor family",
		strings.Replace(gh, "os: macos", "os: macos\n      machine: [computer, desk]", 1):                              `kind "desk" is none of`,
		strings.Replace(gh, "os: macos", "os: macos\n      machine: []", 1):                                            "machine lists no kind of machine",
		strings.Replace(gh, "    - os: macos\n      brew: gh", "    - family: linux\n      apt: {packages: [gh]}", 1):  "apt runs on the debian family",
		strings.Replace(gh, "    - os: macos\n      brew: gh", "    - os: ubuntu\n      apt: {packages: []}", 1):       "apt names its packages",
		strings.Replace(gh, "    - os: macos\n      brew: gh", "    - os: ubuntu\n      apt: {packages: [gh]}", 1):     "apt installs as root: its method says `needs: [sudo]`",
		strings.Replace(gh, "brew: gh", "needs: [root]\n      brew: gh", 1):                                            `need "root" is none of`,
		strings.Replace(gh, "brew: gh", "script: {run: 'gh --host ${option.host}'}", 1):                                "${option.host} names no option of the item",
		strings.Replace(gh, "provides: [gh]", "provides: [gh]\nports: [{port: 0, label: GH}]", 1):                      "port 0: a port from 1 to 65535",
		strings.Replace(gh, "provides: [gh]", "provides: [gh]\nports: [{port: 80, label: GH, env: {A: '{path}'}}]", 1): "env A uses {path}, only {url} and",
		strings.Replace(gh, "provides: [gh]", "provides: [gh]\nports: [{port: 80, label: GH, env: {'A=B': x}}]", 1):    `env name "A=B" is no variable name`,
		strings.Replace(gh, "kind: app", "kind: manager", 1) + "manages: [brew]\npriority: {}\n":                       "a priority ranks at least one system",
		strings.Replace(gh, "provides: [gh]", "provides: [gh]\nmanages: [brew]", 1):                                    "`manages` is not a key of a app",
		strings.Replace(gh, "provides: [gh]", "provides: [gh]\ndeprecated: {use: hub}", 1):                             "deprecated says why",
		strings.Replace(gh, "os: macos", "os: macos\n      arch: x86", 1):                                              `arch "x86" is none of`,
		strings.Replace(gh, "os: macos", "os: any\n      version: '>=14'", 1):                                          "a version range is of one os",
		strings.Replace(gh, "os: macos", "os: macos\n      version: 'fourteen'", 1):                                    `version "fourteen" is no range`,
		strings.Replace(gh, "os: macos", "os: macos\n      version: '>=15,'", 1):                                       `version ">=15," is no range`,
		strings.Replace(gh, "os: macos", "os: macos\n      version: '>=15,,<27'", 1):                                   `version ">=15,,<27" is no range`,
		strings.Replace(gh, "os: macos", "os: macos\n      pin: tag", 1):                                               `pin "tag" is none of`,
		strings.Replace(gh, "brew: gh", "apt: {packages: [gh]}", 1):                                                    "apt runs on the debian family",
		strings.Replace(gh, "brew: gh", "github-release: {repo: gh, asset: gh}", 1):                                    "github-release names its repo",
		strings.Replace(gh, "brew: gh", "script: {run: a, url: b}", 1):                                                 "a run line or a url, one of them",
		strings.Replace(gh, "brew: gh", "script: {check: gh}", 1):                                                      "or says the owner's way in its hint",
		strings.Replace(gh, "brew: gh", "script: {hint: download it, undo: rm gh}", 1):                                 "has a hint, no check, undo or upgrade",
		strings.Replace(gh, "brew: gh", "upgrade: gh up\n      script: {run: a}", 1):                                   "a run line takes no args, env or upgrade",
		strings.Replace(gh, "provides: [gh]", "provides: [gh]\nphase: later", 1):                                       `phase "later" is none of`,
		strings.Replace(gh, "provides: [gh]", "provides: [gh]\nafter-apply: [restart-gh]", 1):                          `"restart-gh" is none of`,
		strings.Replace(gh, "provides: [gh]", "provides: [gh]\naccepts: [play-store]", 1):                              `source "play-store" is none of`,
		strings.Replace(gh, "provides: [gh]", "provides: [gh]\nrenamed-from: [gh]", 1):                                 "not renamed from its own name",
		strings.Replace(gh, "os: macos", "os: windows", 1):                                                             `os "windows" is none of`,
		strings.Replace(gh, "brew: gh", "builtin: dns", 1):                                                             "`builtin` is no method of a app",
		strings.Replace(strings.Replace(gh, "provides: [gh]\n", "", 1), "brew: gh", "provided-by: git", 1):             "an app git provides names its commands in provides",
		strings.Replace(gh, "brew: gh", "bun: gh\n      trusted: true", 1):                                             "`trusted` is for a brew or a cask",
		strings.Replace(gh, "    - os: macos\n      brew: gh\n", "", 1) + "    []\n":                                   "has no method",
	} {
		refusedFor(t, body, want)
	}
	setting := "item: station.catalog/v1\nkind: setting\nname: gh\ndescription: d\nbundle: GH\nmethods:\n    - os: any\n      builtin: dns\n"
	refusedFor(t, setting, "`bundle` is not a key of a setting")
	refusedFor(t, strings.Replace(setting, "bundle: GH", "version-read: {args: [gh], pattern: x}", 1), "`version-read` is not a key of a setting")
	withPort := strings.Replace(gh, "provides: [gh]", "provides: [gh]\nports: [{port: 5173, label: Vite, http: true, env: {A: '{url}', B: 'h={host}'}}]", 1)
	if it, err := catalog.Decode("items/git/gh.yaml", []byte(withPort)); err != nil || !it.Ports[0].HTTP || it.Ports[0].Env["B"] != "h={host}" {
		t.Errorf("a port with http and env decoded to %+v, %v", it.Ports, err)
	}
	if err := decoded(t, strings.Replace(setting, "bundle: GH\n", "", 1)+"    - os: macos\n      builtin: dns\n"); err != nil {
		t.Errorf("a setting with a method per system is refused: %v", err)
	}
	refusedFor(t, strings.Replace(setting, "bundle: GH", "requires: [gh]\nrequires-one-of: [gh, hub]", 1), "requires and requires-one-of, one of them")
	refusedFor(t, strings.Replace(setting, "bundle: GH", "needs: [root]", 1), `need "root" is none of`)
	if err := decoded(t, strings.Replace(strings.Replace(setting, "bundle: GH\n", "", 1), "builtin: dns", "builtin: nope", 1)); err != nil {
		t.Errorf("a builtin a newer station carries is refused at decode: %v", err)
	}
	defaults := strings.Replace(strings.Replace(setting, "bundle: GH\n", "options:\n    - {name: size, label: Size, type: integer, min: 16, max: 128, default: 48}\n", 1),
		"    - os: any\n      builtin: dns\n", "    - os: macos\n      macos-defaults:\n          domain: com.apple.dock\n          keys: [{key: tilesize, type: integer, value: '${option.size}'}]\n", 1)
	if err := decoded(t, defaults); err != nil {
		t.Errorf("a macos-defaults setting reading its option is refused: %v", err)
	}
	refusedFor(t, strings.Replace(defaults, "option.size", "option.width", 1), "${option.width} names no option of the item")
	refusedFor(t, strings.Replace(defaults, "os: macos", "os: any", 1), "macos-defaults is for os: macos")
	refusedFor(t, defaults+"writes: [{path: ~/x, level: settings, os: macos, family: linux}]\n", "~/x: names an os and a family")
}

func TestAnOptionIsJudgedByItsType(t *testing.T) {
	option := func(o string) string {
		return "item: station.catalog/v1\nkind: setting\nname: gh\ndescription: d\noptions:\n    - " + o + "\nmethods:\n    - os: any\n      builtin: dns\n"
	}
	for o, want := range map[string]string{
		"{name: a, label: A, type: text}":                                 `type "text" is none of`,
		"{name: a, label: A, type: string, values: [x]}":                  "an enum lists its values, and only an enum",
		"{name: a, label: A, type: enum}":                                 "an enum lists its values, and only an enum",
		"{name: a, label: A, type: enum, values: [left], default: right}": `"right" is none of its values`,
		"{name: a, label: A, type: boolean, default: 'yes'}":              `"yes" is no boolean`,
		"{name: a, label: A, type: string, min: 1}":                       "min and max bound an integer",
		"{name: a, label: A, type: integer, min: 9, max: 1}":              "min is above max",
		"{name: a, label: A, type: integer, max: 9, examples: [12]}":      "12 is out of its bounds",
		"{name: a, label: A, type: integer, default: many}":               `"many" is no integer`,
		"{name: a, label: A, type: boolean, examples: [true]}":            "a boolean has a default, no examples",
	} {
		refusedFor(t, option(o), "option a: "+want)
	}
	refusedFor(t, option("{name: a, label: A, type: boolean, secret: true}"), "option a: a secret is a string")
	for _, o := range []string{
		"{name: a, label: A, type: boolean, default: true}",
		"{name: a, label: A, type: enum, values: [left, bottom], default: bottom}",
		"{name: a, label: A, type: integer, min: 1, max: 9, default: 4, examples: [2, 8]}",
		"{name: a, label: A, type: string, secret: true, required: true, examples: [hunter2]}",
	} {
		if err := decoded(t, option(o)); err != nil {
			t.Errorf("option %s is refused: %v", o, err)
		}
	}
}

func TestEveryKeyOfTheFormatDecodes(t *testing.T) {
	full := `item: station.catalog/v1
kind: app
name: gh
description: GitHub CLI
provides: [gh]
requires: [git]
after: [tmux]
then: [github-login]
phase: image
renamed-from: [hub]
deprecated: {why: the GitHub app does it, use: git}
ports: [{port: 8080, label: gh webhook}]
version-read: {args: [gh, --version], pattern: "gh version (\\S+)"}
bundle: GitHub
accepts: [app-store]
needs: [sudo, license, app-store-login]
after-apply: [restart-dock, logout]
options:
    - {name: host, label: Host, type: string, default: github.com, examples: [ghe.example.com]}
    - {name: verbose, label: Verbose, type: boolean, default: false}
methods:
    - os: macos
      version: '>=14, <27'
      arch: arm64
      machine: computer
      pin: none
      brew: gh
    - os: ubuntu
      machine: [workspace, server]
      provides: [gh]
      upgrade: gh extension upgrade --all
      pin: version
      needs: [sudo]
      apt: {packages: [gh]}
    - family: linux
      arch: arm64
      github-release: {repo: cli/cli, asset: 'gh_*_linux_{arch}.tar.gz', checksums: gh_checksums.txt}
    - os: any
      script: {check: command -v gh, run: 'curl -fsSL gh.sh | GH_HOST=${option.host} sh', undo: rm ~/.local/bin/gh}
`
	it, err := catalog.Decode("items/git/gh.yaml", []byte(full))
	if err != nil {
		t.Fatal(err)
	}
	if it.Methods[1].Machine[1] != catalog.RoleServer || it.Methods[0].Machine[0] != catalog.RoleComputer || it.Methods[2].GitHubRelease.Repo != "cli/cli" {
		t.Errorf("decoded %+v", it.Methods)
	}
}

func catalogOf(files map[string]string) (catalog.Catalog, error) {
	fsys := fstest.MapFS{}
	for name, body := range files {
		fsys["items/"+name] = &fstest.MapFile{Data: []byte(body)}
	}
	return catalog.Load(fsys, "items")
}

func manager(name, manages, at string) string {
	return "item: station.catalog/v1\nkind: manager\nname: " + name + "\ndescription: d\nmanages: " + manages + "\npriority: {macos: " + at + "}\nmethods:\n    - os: macos\n      script: {url: https://x.sh}\n"
}

func TestTheItemsAreJudgedTogether(t *testing.T) {
	dns := "item: station.catalog/v1\nkind: setting\nname: dns\ndescription: d\nrequires: [gpg]\nmethods:\n    - os: any\n      builtin: dns\n"
	for files, want := range map[*map[string]string]string{
		{"git/gh.yaml": strings.Replace(gh, "provides: [gh]", "requires: [git]", 1)}: `items/git/gh.yaml: gh requires "git", which is no app`,
		{"git/gh.yaml": strings.Replace(gh, "provides: [gh]", "then: [dns]", 1)}:     `gh proposes "dns", which is no setting`,
		{"git/gh.yaml": strings.Replace(gh, "brew: gh", "provided-by: hub", 1)}:      `gh is provided by "hub", which is no app`,
		{"security/dns.yaml": dns}: `items/security/dns.yaml: setting dns requires "gpg", which is no app`,
		{"tools/gh.yaml": gh}:      "folder tools is no category of apps",
		{"git/gh.yaml": strings.Replace(gh, "provides: [gh]", "after: [tmux]", 1)}:                                                                              `items/git/gh.yaml: gh comes after "tmux", which is no item`,
		{"git/gh.yaml": strings.Replace(gh, "provides: [gh]", "renamed-from: [dns]", 1), "security/dns.yaml": strings.Replace(dns, "requires: [gpg]\n", "", 1)}: `gh is renamed from "dns", which names an item still`,
		{"git/gh.yaml": gh, "security/gh.yaml": strings.Replace(dns, "name: dns", "name: gh", 1)}:                                                               `"gh" names both a app and a setting`,
		{"git/gh.yml": gh}: "is no item file",
		{"package-managers/brew.yaml": manager("brew", "[brew, cask]", "1"), "package-managers/tap.yaml": manager("tap", "[cask]", "2")}: "brew and tap both manage cask",
		{"package-managers/brew.yaml": manager("brew", "[brew]", "1"), "package-managers/mas.yaml": manager("mas", "[mas]", "1")}:        "brew and mas both stand at 1 on macos",
		{"git/gh.yaml": strings.Replace(gh, "provides: [gh]", "deprecated: {why: gone, use: hub}", 1)}:                                   `gh says to use "hub", which is no item`,
	} {
		if _, err := catalogOf(*files); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("loaded with %v, want %q", err, want)
		}
	}
}

func TestItemsStandByCategoryThenByName(t *testing.T) {
	item := func(name string) string {
		return strings.Replace(strings.Replace(gh, "name: gh", "name: "+name, 1), "brew: gh", "brew: "+name, 1)
	}
	c, err := catalogOf(map[string]string{
		"git/zz.yaml": item("zz"), "git/aa.yaml": item("aa"),
		"runtimes/mm.yaml": item("mm"), "agent-skills/bb.yaml": item("bb"),
	})
	if err != nil {
		t.Fatal(err)
	}
	got := make([]string, 0, len(c.Apps))
	for _, a := range c.Apps {
		got = append(got, a.Name)
	}
	if strings.Join(got, " ") != "mm aa zz bb" {
		t.Errorf("the apps stand %v, want mm aa zz bb: runtimes, git, then agent skills", got)
	}
}

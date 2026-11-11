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
	copied := "item: station.catalog/v1\nkind: setting\nname: gh\ndescription: d\nmethods:\n    - os: any\n      file:\n        copy:\n            - {from: a, to: b, mode: 0755}\n"
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
	refusedFor(t, strings.Replace(gh, "item: station.catalog/v1\n", "", 1), "names no schema")
}

func TestAnItemIsJudgedAlone(t *testing.T) {
	for body, want := range map[string]string{
		strings.Replace(gh, "name: gh", "name: hub", 1):                                                    `names "hub", its file "gh"`,
		strings.Replace(gh, "kind: app", "kind: tool", 1):                                                  `kind "tool" is none of`,
		strings.Replace(gh, "provides: [gh]", "options: []\nguided: true", 1):                              "`guided` is not a key of a app",
		strings.Replace(gh, "brew: gh", "brew: gh\n      cask: gh", 1):                                     "names 2 managers (brew, cask), not one",
		strings.Replace(gh, "os: macos", "os: linux", 1):                                                   "brew does not run on linux",
		strings.Replace(gh, "os: macos", "os: windows", 1):                                                 `os "windows" is none of`,
		strings.Replace(gh, "brew: gh", "builtin: dns", 1):                                                 "`builtin` is no method of a app",
		strings.Replace(strings.Replace(gh, "provides: [gh]\n", "", 1), "brew: gh", "provided-by: git", 1): "an app git provides names its commands in provides",
		strings.Replace(gh, "brew: gh", "bun: gh\n      trusted: true", 1):                                 "`trusted` is for a brew or a cask",
		strings.Replace(gh, "    - os: macos\n      brew: gh\n", "", 1) + "    []\n":                       "has no method",
	} {
		refusedFor(t, body, want)
	}
	setting := "item: station.catalog/v1\nkind: setting\nname: gh\ndescription: d\nbundle: GH\nmethods:\n    - os: any\n      builtin: dns\n"
	refusedFor(t, setting, "`bundle` is not a key of a setting")
	refusedFor(t, strings.Replace(setting, "bundle: GH\n", "", 1)+"    - os: macos\n      builtin: dns\n", "a setting has one method")
	refusedFor(t, strings.Replace(setting, "bundle: GH", "needs: [root]", 1), `need "root" is none of`)
	refusedFor(t, strings.Replace(strings.Replace(setting, "bundle: GH\n", "", 1), "builtin: dns", "builtin: nope", 1), `builtin "nope" is no procedure`)
}

func catalogOf(files map[string]string) (catalog.Catalog, error) {
	fsys := fstest.MapFS{}
	for name, body := range files {
		fsys["items/"+name] = &fstest.MapFile{Data: []byte(body)}
	}
	return catalog.Load(fsys, "items")
}

func TestTheItemsAreJudgedTogether(t *testing.T) {
	dns := "item: station.catalog/v1\nkind: setting\nname: dns\ndescription: d\nrequires: [gpg]\nmethods:\n    - os: any\n      builtin: dns\n"
	for files, want := range map[*map[string]string]string{
		{"git/gh.yaml": strings.Replace(gh, "provides: [gh]", "requires: [git]", 1)}: `gh requires "git", which is no app`,
		{"git/gh.yaml": strings.Replace(gh, "provides: [gh]", "then: [dns]", 1)}:     `gh proposes "dns", which is no setting`,
		{"git/gh.yaml": strings.Replace(gh, "brew: gh", "provided-by: hub", 1)}:      `gh is provided by "hub", which is no app`,
		{"security/dns.yaml": dns}: `setting dns requires "gpg", which is no app`,
		{"tools/gh.yaml": gh}:      "folder tools is no category of apps",
		{"git/gh.yaml": gh, "security/gh.yaml": strings.Replace(dns, "name: dns", "name: gh", 1)}: `"gh" names both a app and a setting`,
		{"git/gh.yml": gh}: "is no item file",
	} {
		if _, err := catalogOf(*files); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("loaded with %v, want %q", err, want)
		}
	}
}

func TestItemsStandInTheirOrderThenByName(t *testing.T) {
	item := func(name, order string) string {
		return strings.Replace(strings.Replace(gh, "name: gh", "name: "+name+"\n"+order, 1), "brew: gh", "brew: "+name, 1)
	}
	c, err := catalogOf(map[string]string{
		"git/zz.yaml": item("zz", "order: 10"), "git/aa.yaml": item("aa", "order: 20"),
		"git/mm.yaml": item("mm", ""), "git/bb.yaml": item("bb", ""),
	})
	if err != nil {
		t.Fatal(err)
	}
	got := make([]string, 0, len(c.Apps))
	for _, a := range c.Apps {
		got = append(got, a.Name)
	}
	if strings.Join(got, " ") != "zz aa bb mm" {
		t.Errorf("the apps stand %v, want zz aa bb mm", got)
	}
}

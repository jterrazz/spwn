package catalog_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

func TestEveryAppOfTheCatalogueReadsBackAsItWasSent(t *testing.T) {
	c, err := catalog.Builtin()
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range c.Apps {
		raw, err := json.Marshal(tool)
		if err != nil {
			t.Fatalf("%s: %v", tool.Name, err)
		}
		var back catalog.App
		if err := json.Unmarshal(raw, &back); err != nil {
			t.Fatalf("%s: %v", tool.Name, err)
		}
		if !reflect.DeepEqual(back, tool) {
			t.Errorf("%s came back as %+v, want %+v", tool.Name, back.Installs, tool.Installs)
		}
	}
}

func TestAnAptInstallOfNoPackageDoesNotCrossTheWire(t *testing.T) {
	// invariant: a peer's apt install names a package at least, so no reader
	// indexes an empty list; its key is none rather than a panic.
	var p catalog.LivesOn
	for _, body := range []string{
		`{"Kinds":["linux"],"Method":"apt","Install":{"Packages":[]}}`,
		`{"Kinds":["linux"],"Method":"apt","Install":{}}`,
	} {
		if err := json.Unmarshal([]byte(body), &p); err == nil {
			t.Errorf("%s decoded to %+v; want it refused", body, p.Install)
		}
	}
	if got := (catalog.AptInstall{}).Key(); got != "" {
		t.Errorf("the key of no package is %q; want none", got)
	}
}

func TestAnInstallWordsItsSystemAndWhatItFetches(t *testing.T) {
	for _, c := range []struct {
		on              catalog.LivesOn
		system, fetches string
	}{
		{catalog.LivesOn{Kinds: []catalog.MachineKind{catalog.Mac}, Install: catalog.BrewInstall{Formula: "jq"}}, "macOS", "jq"},
		{catalog.LivesOn{Kinds: []catalog.MachineKind{catalog.Mac, catalog.MacOS}, Install: catalog.MiseInstall{App: "node", Version: "lts"}}, "macOS", "node lts"},
		{catalog.LivesOn{Kinds: []catalog.MachineKind{catalog.Linux}, Install: catalog.AptInstall{Packages: []string{"fd-find", "fzf"}}}, "Ubuntu", "fd-find, fzf"},
		{catalog.LivesOn{Kinds: []catalog.MachineKind{catalog.Mac, catalog.Linux}, Install: catalog.SkillInstall{Repo: "acme/skills", Skill: "layout"}}, "macOS, Ubuntu", "acme/skills layout"},
	} {
		raw, err := json.Marshal(c.on)
		if err != nil {
			t.Fatal(err)
		}
		var got struct{ System, Fetches string }
		if err := json.Unmarshal(raw, &got); err != nil {
			t.Fatal(err)
		}
		if got.System != c.system || got.Fetches != c.fetches {
			t.Errorf("%s: system %q, fetches %q; want %q, %q", raw, got.System, got.Fetches, c.system, c.fetches)
		}
	}
}

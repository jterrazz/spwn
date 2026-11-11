package catalog_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/jterrazz/studio-station/src/catalog"
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

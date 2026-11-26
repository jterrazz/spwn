package agents_test

import (
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/agents"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

func TestANameIsLowercaseAndNoVerb(t *testing.T) {
	for name, want := range map[string]bool{
		"neo": true, "web-2": true, "a": true, strings.Repeat("a", agents.MaxName): true,
		"": false, "Neo": false, "2web": false, "-neo": false, "neo_1": false, "neo.yaml": false,
		strings.Repeat("a", agents.MaxName+1): false, "create": false, "talk": false, "help": false,
	} {
		if got := agents.ValidName(name); got != want {
			t.Errorf("ValidName(%q) = %v; want %v", name, got, want)
		}
	}
}

func TestAnEditNamesAKnownHarnessOrClearsIt(t *testing.T) {
	known, empty, other := catalog.Harnesses()[1], "", "gemini"
	for _, c := range []struct {
		edit agents.Edit
		ok   bool
	}{{agents.Edit{}, true}, {agents.Edit{Harness: &known}, true}, {agents.Edit{Harness: &empty}, true}, {agents.Edit{Harness: &other}, false}} {
		if err := c.edit.Check(); (err == nil) != c.ok {
			t.Errorf("Check(%+v) = %v; want ok %v", c.edit, err, c.ok)
		}
	}
}

func TestAnEditChangesTheFieldsItNamesAlone(t *testing.T) {
	team, none := "core", ""
	a := agents.Agent{Name: "neo", Description: "writes", Team: "web", Harness: catalog.Harnesses()[1]}
	got := agents.Edit{Team: &team, Harness: &none}.Apply(a)
	if want := (agents.Agent{Name: "neo", Description: "writes", Team: "core"}); got != want {
		t.Errorf("Apply = %+v; want %+v", got, want)
	}
	if got := (agents.Edit{}).Apply(a); got != a {
		t.Errorf("an empty edit = %+v; want %+v", got, a)
	}
}

func TestAnArchiveNamesItsAgent(t *testing.T) {
	if got := agents.ArchiveName("/tmp/out/neo.tar.gz"); got != "neo" {
		t.Errorf("ArchiveName = %q; want neo", got)
	}
}

func TestANewAgentRunsInTheDefaultHarness(t *testing.T) {
	if got := agents.Blank("neo"); got.Name != "neo" || got.Harness != catalog.Harnesses()[0] || !strings.Contains(got.Description, "neo") {
		t.Errorf("Blank = %+v; want neo, the default harness, a line naming it", got)
	}
}

package calls_test

import (
	"encoding/json"
	"testing"

	appscalls "github.com/jterrazz/studio-station/src/apps/calls"
	"github.com/jterrazz/studio-station/src/catalog"
)

func TestAnOlderPeerIsAskedAnUpdateInItsManagerWords(t *testing.T) {
	// invariant: v3.0 names skill skills and cask by Homebrew's other
	// key; a manager it has none of is the word it lacks, the In
	// untouched, and another call's In is never written.
	asked := func(keys ...catalog.Method) json.RawMessage {
		words := make([]string, 0, len(keys))
		for _, k := range keys {
			words = append(words, string(k))
		}
		raw, _ := json.Marshal(appscalls.UpdateIn{Method: words})
		return raw
	}
	got, lacks := appscalls.OlderIn(appscalls.Update.Name, asked(catalog.MethodSkill, catalog.MethodCask))
	var read struct{ Method []string }
	_ = json.Unmarshal(got, &read)
	homebrew, _ := catalog.OlderManager(string(catalog.MethodCask))
	if want := []string{"skills", homebrew}; lacks != "" || len(read.Method) != 2 || read.Method[0] != want[0] || read.Method[1] != want[1] || homebrew == string(catalog.MethodCask) {
		t.Errorf("written %s (lacks %q), want the managers %v", got, lacks, want)
	}
	in := asked(catalog.MethodSkill, catalog.MethodShell)
	if got, lacks := appscalls.OlderIn(appscalls.Update.Name, in); string(got) != string(in) || lacks != string(catalog.MethodShell) {
		t.Errorf("written %s, lacks %q; want it as it came, lacking %s", got, lacks, catalog.MethodShell)
	}
	other := asked(catalog.MethodSkill)
	if got, lacks := appscalls.OlderIn(appscalls.Install.Name, other); string(got) != string(other) || lacks != "" {
		t.Errorf("another call's In written %s (lacks %q)", got, lacks)
	}
}

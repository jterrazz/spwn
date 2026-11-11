package apps_test

import (
	"maps"
	"testing"

	"github.com/jterrazz/studio-station/src/apps"
	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/core/machine"
)

func TestAnUpdateWaitingIsFixedByItsManagerInBothWords(t *testing.T) {
	// invariant: a v3.0 menu bar reads the fix's manager in `method`, so
	// the fix carries it there in v3.0's word beside today's `manager`,
	// and a manager v3.0 has none of by its own key, which v3.0 matches to
	// no app: without it, its Update all updates every app.
	waiting := func(m catalog.Method, key string) apps.Report {
		return apps.Report{App: catalog.App{Name: string(m)}, Method: m, Manager: key, State: catalog.StateDegraded, Why: apps.WhyOutdated, Managed: true}
	}
	skill, cask, apt := string(catalog.MethodSkill), string(catalog.MethodCask), string(catalog.MethodApt)
	homebrew, _ := catalog.OlderManager(cask)
	found := apps.Findings([]apps.Report{waiting(catalog.MethodSkill, skill), waiting(catalog.MethodCask, cask), waiting(catalog.MethodApt, apt)}, machine.Ref{})
	want := []map[string]string{{"manager": skill, "method": "skills"}, {"manager": cask, "method": homebrew}, {"manager": apt, "method": apt}}
	if len(found) != len(want) {
		t.Fatalf("found %+v", found)
	}
	for i, f := range found {
		if !maps.Equal(f.Fix.Args, want[i]) {
			t.Errorf("fix %d args %v, want %v", i, f.Fix.Args, want[i])
		}
	}
}

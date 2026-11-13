package apps_test

import (
	"maps"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/apps"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
)

func TestAnUpdateWaitingIsFixedByItsManagerKey(t *testing.T) {
	// invariant: the fix of a waiting update names its manager by today's
	// key alone, `manager`, whatever the method (cask's is `cask`).
	waiting := func(m catalog.Method, key string) apps.Report {
		return apps.Report{App: catalog.App{Name: string(m)}, Method: m, Manager: key, State: catalog.StateDegraded, Why: apps.WhyOutdated, Managed: true}
	}
	skill, cask, apt := string(catalog.MethodSkill), string(catalog.MethodCask), string(catalog.MethodApt)
	found := apps.Findings([]apps.Report{waiting(catalog.MethodSkill, skill), waiting(catalog.MethodCask, cask), waiting(catalog.MethodApt, apt)}, machine.Ref{})
	want := []map[string]string{{"manager": skill}, {"manager": cask}, {"manager": apt}}
	if len(found) != len(want) {
		t.Fatalf("found %+v", found)
	}
	for i, f := range found {
		if !maps.Equal(f.Fix.Args, want[i]) {
			t.Errorf("fix %d args %v, want %v", i, f.Fix.Args, want[i])
		}
	}
}

package fake_test

import (
	"encoding/json"
	"errors"
	"slices"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/automations"
	"github.com/jterrazz/studio-station/apps/engine/automations/adapter/fake"
	"github.com/jterrazz/studio-station/apps/engine/automations/contract"
	"github.com/jterrazz/studio-station/apps/engine/automations/contract/contracttest"
	"github.com/jterrazz/studio-station/apps/engine/world"
)

func worldWith(t *testing.T, stores map[string]any) *world.World {
	t.Helper()
	raw, err := json.Marshal(map[string]any{"macs": map[string]any{"this-mac": map[string]any{"stores": stores}}})
	if err != nil {
		t.Fatal(err)
	}
	w := &world.World{}
	if err := json.Unmarshal(raw, w); err != nil {
		t.Fatal(err)
	}
	return w
}

func TestAutomations(t *testing.T) {
	contracttest.Automations(t, func(t *testing.T, declared []contracttest.Declared, ran []automations.Run) (contract.Automations, string) {
		project := t.TempDir()
		kept := make([]automations.Automation, 0, len(declared))
		for _, d := range declared {
			kept = append(kept, automations.Automation{Name: d.Name, Agent: d.Agent, Trigger: d.Trigger})
		}
		w := worldWith(t, map[string]any{"automations": map[string]any{project: kept}, "automation_runs": map[string]any{project: ran}})
		return fake.Of(w, "this-mac"), project
	})
}

func TestAFolderNoProjectHoldsDeclaresNone(t *testing.T) {
	contracttest.NoProject(t, func(t *testing.T) (contract.Automations, string) {
		return fake.Of(worldWith(t, map[string]any{}), "this-mac"), ""
	})
}

func TestANameTwoGroupsDeclareIsNotUniqueUntilQualified(t *testing.T) {
	project := t.TempDir()
	declared := []automations.Automation{{Group: "b", Name: "same", Agent: "neo"}, {Group: "a", Name: "same", Agent: "neo"}}
	ran := []automations.Run{{Group: "a", Name: "same", OK: true}, {Group: "b", Name: "same", OK: true}, {Group: "b", Name: "same"}}
	w := worldWith(t, map[string]any{"automations": map[string]any{project: declared}, "automation_runs": map[string]any{project: ran}})
	fakeAutomations := fake.Of(w, "this-mac")
	all, err := fakeAutomations.List(t.Context(), project)
	if err != nil || len(all) != 2 || all[0].Group != "a" || !all[0].Shared || all[1].Fires != 2 || all[1].Failed != 1 {
		t.Fatalf("List = %+v, %v; want both, group a first, each with its own runs", all, err)
	}
	var many *automations.NotUniqueError
	if _, shown := fakeAutomations.Automation(t.Context(), project, "same"); !errors.As(shown, &many) || !slices.Equal(many.Groups, []string{"a", "b"}) {
		t.Fatalf("Automation(same) = %v; want not unique in a, b", shown)
	}
	if _, logged := fakeAutomations.Log(t.Context(), project, "same", 0); !errors.As(logged, &many) {
		t.Fatalf("Log(same) = %v; want not unique", logged)
	}
	logged, err := fakeAutomations.Log(t.Context(), project, "b/same", 0)
	runs := logged.Runs
	if err != nil || len(runs) != 2 || runs[0].Group != "b" || runs[1].Group != "b" {
		t.Fatalf("Log(b/same) = %+v, %v; want the two runs of b", runs, err)
	}
}

func TestAManifestThatDoesNotReadIsOneInvalidProjectError(t *testing.T) {
	project := t.TempDir()
	w := worldWith(t, map[string]any{
		"automation_manifest_errors": map[string]any{project: "worlds: cannot read a list"},
		"automation_runs":            map[string]any{project: []automations.Run{{Name: "morning", OK: true}}},
	})
	fakeAutomations := fake.Of(w, "this-mac")
	var bad *automations.InvalidProjectError
	if _, err := fakeAutomations.List(t.Context(), project); !errors.As(err, &bad) || bad.Reason != "worlds: cannot read a list" {
		t.Fatalf("List = %v; want an InvalidProjectError with the manifest's error", err)
	}
	if _, err := fakeAutomations.Log(t.Context(), project, "morning", 0); !errors.As(err, &bad) {
		t.Fatalf("Log(morning) = %v; want an InvalidProjectError", err)
	}
	logged, err := fakeAutomations.Log(t.Context(), project, "", 0)
	runs := logged.Runs
	if err != nil || len(runs) != 1 {
		t.Fatalf("Log() = %+v, %v; want the runs, the manifest unread", runs, err)
	}
}

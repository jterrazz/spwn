package fake_test

import (
	"encoding/json"
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

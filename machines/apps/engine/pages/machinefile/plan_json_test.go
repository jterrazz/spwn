package machinefile_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/machinefile"
	pagemachinefile "github.com/jterrazz/studio-station/apps/engine/pages/machinefile"
)

func TestARunsDocumentSaysWhyTheSkillWasNotWrittenAndNothingWhenItWas(t *testing.T) {
	const why = "open ~/.agents/skills/spwn/SKILL.md: permission denied"
	for _, c := range []struct {
		name   string
		failed string
		doc    any
	}{
		{"apply", why, pagemachinefile.AppliedContract("station.machinefile.apply/v2", machinefile.Applied{SkillFailed: why}, nil, nil)},
		{"update", why, pagemachinefile.UpgradedContract("station.machinefile.upgrade/v2", machinefile.Upgraded{SkillFailed: why}, nil)},
		{"apply that wrote it", "", pagemachinefile.AppliedContract("station.machinefile.apply/v2", machinefile.Applied{}, nil, nil)},
		{"update that wrote it", "", pagemachinefile.UpgradedContract("station.machinefile.upgrade/v2", machinefile.Upgraded{}, nil)},
	} {
		body, err := json.Marshal(c.doc)
		if err != nil {
			t.Fatal(err)
		}
		var keys map[string]any
		if err := json.Unmarshal(body, &keys); err != nil {
			t.Fatal(err)
		}
		got, said := keys["skill_failed"]
		switch {
		case c.failed == "" && said:
			t.Errorf("%s: skill_failed is %v; want it absent: %s", c.name, got, body)
		case c.failed != "" && got != c.failed:
			t.Errorf("%s: skill_failed is %v; want %q: %s", c.name, got, c.failed, body)
		}
	}
	plan := pagemachinefile.PlanContract("station.machinefile.plan/v2", machinefile.Plan{})
	if body, _ := json.Marshal(plan); strings.Contains(string(body), "skill_failed") {
		t.Errorf("a plan says skill_failed: %s", body)
	}
}

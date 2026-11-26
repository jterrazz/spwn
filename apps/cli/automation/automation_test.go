package automation

import (
	"testing"

	"spwn.sh/packages/project"
)

func TestCountAutomations(t *testing.T) {
	p := &project.Project{
		Manifest: &project.Manifest{
			Worlds: map[string]project.World{
				"a": {Automations: map[string]project.Automation{"x": {}, "y": {}}},
				"b": {Automations: map[string]project.Automation{"z": {}}},
			},
		},
	}
	if got := countAutomations(p); got != 3 {
		t.Errorf("got %d, want 3", got)
	}
	if got := countAutomations(nil); got != 0 {
		t.Errorf("nil project = %d, want 0", got)
	}
}

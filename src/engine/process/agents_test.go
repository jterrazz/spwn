package process

import (
	"slices"
	"testing"

	"github.com/jterrazz/studio-station/src/agents"
	"github.com/jterrazz/studio-station/src/catalog"
)

// invariant: SPEC A4 Home 3: the agent a VM is said by carries its current
// step to the panel, in its parts, as the hooks recorded it.
func TestAVMsAgentSaysItsCurrentStep(t *testing.T) {
	v := agentView([]agents.Agent{{State: agents.Working, Task: "fix the flaky test", Steps: []agents.Step{
		{Verb: catalog.AgentEdit, Object: "src/auth/refresh.ts"},
		{Verb: catalog.AgentTest, Tests: &agents.Tests{Pass: 41, Total: 58}},
	}}})
	if v == nil || !slices.Equal(v.Step, []string{"running tests", "41/58 pass"}) || v.Task != "fix the flaky test" {
		t.Fatalf("view %+v", v)
	}
	if v := agentView([]agents.Agent{{State: agents.Working}}); v == nil || v.Step != nil {
		t.Fatalf("an agent with no step yet says one: %+v", v)
	}
}

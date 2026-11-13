package calls

import (
	"slices"

	"github.com/jterrazz/studio-station/apps/engine/agents"
	model "github.com/jterrazz/studio-station/apps/engine/machines"
)

type claim int

const (
	unclaimed claim = iota
	claimMac
	claimVM
)

type heldAgent struct{ Name, VM string }

// value: every name a new VM may not take: each Mac's and VM's of the fleet,
// with the Mac a VM is on, and each agent's of a workspace that answers.
type claims struct {
	held   map[string]claim
	on     map[string]string
	agents []heldAgent
}

// invariant: a name is held by a VM record on any host of the fleet or by a
// host, and by any agent the VM would give its own agents unasked (SPEC
// B5.22): its name or its ·<n>, so no agent run by hand on it shares a name;
// no tailnet node holds a VM's name since VMs left the tailnet.
func (c claims) agentOf(name string) (heldAgent, bool) {
	i := slices.IndexFunc(c.agents, func(a heldAgent) bool { return agents.Defaults(name, a.Name) })
	if i < 0 {
		return heldAgent{}, false
	}
	return c.agents[i], true
}

func (c claims) free(name string) bool {
	_, agent := c.agentOf(name)
	return c.held[name] == unclaimed && !agent
}

func (c claims) taken(name string) *model.NameTakenError {
	if a, ok := c.agentOf(name); ok && c.held[name] == unclaimed {
		return &model.NameTakenError{Name: name, Agent: a.Name, On: a.VM}
	}
	return &model.NameTakenError{Name: name, VM: c.held[name] == claimVM, Mac: c.held[name] == claimMac, On: c.on[name]}
}

func allocateName(image string, c claims) string { return model.FirstFree(image, c.free) }

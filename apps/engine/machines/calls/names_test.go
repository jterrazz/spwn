package calls

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/agents"
)

func TestAllocateNameSkipsEveryHeldName(t *testing.T) {
	for _, c := range []struct {
		name   string
		held   []string
		agents []string
		want   string
	}{
		{name: "the smallest free integer fills a gap", held: []string{"linux-1", "linux-3"}, want: "linux-2"},
		{name: "a host named like a VM holds its name", held: []string{"x", "linux-1"}, want: "linux-2"},
		{name: "an agent holds the name it bears", held: []string{"linux-1"}, agents: []string{"linux-2"}, want: "linux-3"},
		{name: "an agent holds the VM its nth name is of", held: []string{"linux-1"}, agents: []string{"linux-2" + agents.Nth + "2"}, want: "linux-3"},
	} {
		t.Run(c.name, func(t *testing.T) {
			held := claims{held: map[string]claim{}}
			for _, r := range c.held {
				held.held[r] = claimVM
			}
			for _, a := range c.agents {
				held.agents = append(held.agents, heldAgent{Name: a, VM: "linux-1"})
			}
			if got := allocateName("linux", held); got != c.want {
				t.Errorf("allocateName = %q, want %q", got, c.want)
			}
		})
	}
}

package agents_test

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/agents"
)

func TestANameIsTakenByAnyAgentOrByTheWorkspaceWhoseOwnItIs(t *testing.T) {
	held := agents.Held{Agents: []string{"docs-bot"}, Workspaces: []string{"linux-1", "ci-1"}}
	for _, c := range []struct {
		name, vm, by string
		taken        bool
	}{
		{"docs-bot", "linux-1", "", true},
		{"ci-1", "linux-1", "ci-1", true},
		{"ci-1" + agents.Nth + "3", "linux-1", "ci-1", true},
		{"ci-1-3", "linux-1", "ci-1", true},
		{"ci-1", "ci-1", "", false},
		{"linux-1" + agents.Nth + "2", "linux-1", "", false},
		{"ci-1-bot", "linux-1", "", false},
		{"review-bot", "linux-1", "", false},
	} {
		by, taken := held.Takes(c.name, c.vm)
		if by != c.by || taken != c.taken {
			t.Errorf("%s on %s: taken %v by %q, want %v by %q", c.name, c.vm, taken, by, c.taken, c.by)
		}
	}
}

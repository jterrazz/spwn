//go:build e2e

package e2e

import (
	"os"
	"path/filepath"
	"testing"

	"spwn.sh/packages/agent"
	"spwn.sh/packages/world/tests/e2e/setup"
)

func TestDestroy_RemovesContainer(t *testing.T) {
	// Given - a world with no agent
	// When - the world is destroyed
	chain := setup.NewSpawnBuilder(t).
		NoAgent().
		Execute()

	chain.Destroy().
		// Then - the state should have no worlds
		ExpectState(func(s *setup.StateAssertion) {
			s.WorldCount(0)
		}).
		// AND the container should no longer exist
		ExpectContainer(func(c *setup.ContainerAssertion) {
			c.NotExists()
		})
}

func TestDestroy_AgentSurvives(t *testing.T) {
	// Given - a world with an agent
	ctx := setup.NewTestContext(t)
	ctx.InitAgent("survivor-agent")

	u := ctx.Spawn().
		WithAgent("survivor-agent").
		Execute()

	// When - the world is destroyed
	u.Destroy().
		ExpectState(func(s *setup.StateAssertion) {
			s.WorldCount(0)
		})

	// Then - the agent Mind should still exist on the host
	info, err := agent.InspectAgent("survivor-agent")
	if err != nil {
		t.Fatalf("Agent should survive after destroy: %v", err)
	}
	soulPath := filepath.Join(info.Path, "SOUL.md")
	if _, err := os.Stat(soulPath); err != nil {
		t.Fatalf("Agent SOUL.md should still exist: %v", err)
	}
}

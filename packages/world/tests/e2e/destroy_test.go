//go:build e2e

package e2e

import (
	"testing"

	agentDomain "spwn.sh/packages/mind"
	"spwn.sh/packages/world/tests/e2e/setup"
)

func TestDestroy_RemovesContainer(t *testing.T) {
	// GIVEN a world with no agent
	// WHEN the world is destroyed
	chain := setup.NewSpawnBuilder(t).
		NoAgent().
		Execute()

	chain.Destroy().
		// THEN the state should have no worlds
		ExpectState(func(s *setup.StateAssertion) {
			s.WorldCount(0)
		}).
		// AND the container should no longer exist
		ExpectContainer(func(c *setup.ContainerAssertion) {
			c.NotExists()
		})
}

func TestDestroy_AgentSurvives(t *testing.T) {
	// GIVEN a world with an agent
	ctx := setup.NewTestContext(t)
	ctx.InitAgent("survivor-agent")

	u := ctx.Spawn().
		WithAgent("survivor-agent").
		Execute()

	// WHEN the world is destroyed
	u.Destroy().
		ExpectState(func(s *setup.StateAssertion) {
			s.WorldCount(0)
		})

	// THEN the agent Mind should still exist on the host
	info, err := agentDomain.InspectAgent("survivor-agent")
	if err != nil {
		t.Fatalf("Agent should survive after destroy: %v", err)
	}
	if _, ok := info.Layers["core"]; !ok {
		t.Fatal("Agent Mind should still have core layer")
	}
}

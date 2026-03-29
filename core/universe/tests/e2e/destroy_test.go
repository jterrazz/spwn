//go:build e2e

package e2e

import (
	"testing"

	agentDomain "github.com/jterrazz/spwn/core/agent"
	"github.com/jterrazz/spwn/core/universe/tests/e2e/setup"
)

func TestDestroy_RemovesContainer(t *testing.T) {
	// GIVEN a universe with no agent
	// WHEN the universe is destroyed
	chain := setup.NewSpawnBuilder(t).
		NoAgent().
		Execute()

	chain.Destroy().
		// THEN the state should have no universes
		ExpectState(func(s *setup.StateAssertion) {
			s.UniverseCount(0)
		}).
		// AND the container should no longer exist
		ExpectContainer(func(c *setup.ContainerAssertion) {
			c.NotExists()
		})
}

func TestDestroy_AgentSurvives(t *testing.T) {
	// GIVEN a universe with an agent
	ctx := setup.NewTestContext(t)
	ctx.InitAgent("survivor-agent")

	u := ctx.Spawn().
		WithAgent("survivor-agent").
		Execute()

	// WHEN the universe is destroyed
	u.Destroy().
		ExpectState(func(s *setup.StateAssertion) {
			s.UniverseCount(0)
		})

	// THEN the agent Mind should still exist on the host
	info, err := agentDomain.InspectAgent("survivor-agent")
	if err != nil {
		t.Fatalf("Agent should survive after destroy: %v", err)
	}
	if _, ok := info.Layers["personas"]; !ok {
		t.Fatal("Agent Mind should still have personas layer")
	}
}

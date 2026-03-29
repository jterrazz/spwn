//go:build e2e

package e2e

import (
	"testing"

	agentDomain "spwn.sh/core/agent"
	"spwn.sh/core/universe"
	"spwn.sh/core/universe/tests/e2e/setup"
)

func TestFullLifecycle_SpawnInspectDestroy(t *testing.T) {
	// GIVEN a universe spawned with an agent
	ctx := setup.NewTestContext(t)
	ctx.InitAgent("e2e-agent")

	u := ctx.Spawn().
		WithAgent("e2e-agent").
		Execute()

	// THEN the state should show one idle universe
	u.ExpectState(func(s *setup.StateAssertion) {
		s.UniverseCount(1)
		s.UniverseStatus(universe.StatusIdle)
	})

	// AND the container should be running with physics, faculties, and mind
	u.ExpectContainer(func(c *setup.ContainerAssertion) {
		c.IsRunning()
		c.HasFile("/universe/physics.md")
		c.HasFile("/universe/faculties.md")
		c.HasMount("/mind")
	})

	// WHEN inspecting the universe
	// THEN it should report the default config
	u.Inspect().ExpectConfig("default")

	// AND listing should show one universe
	u.List().ExpectCount(1)

	// WHEN the universe is destroyed
	u.Destroy().
		// THEN the state should be empty
		ExpectState(func(s *setup.StateAssertion) {
			s.UniverseCount(0)
		}).
		// AND the container should no longer exist
		ExpectContainer(func(c *setup.ContainerAssertion) {
			c.NotExists()
		})

	// AND the agent should persist on the host
	info, err := agentDomain.InspectAgent("e2e-agent")
	if err != nil {
		t.Fatalf("Agent should survive after destroy: %v", err)
	}
	if _, ok := info.Layers["personas"]; !ok {
		t.Fatal("Agent Mind should still have personas layer")
	}
}

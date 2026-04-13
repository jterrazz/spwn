//go:build e2e

package e2e

import (
	"testing"

	agentDomain "spwn.sh/packages/agent"
	"spwn.sh/packages/world"
	"spwn.sh/packages/world/tests/e2e/setup"
)

func TestFullLifecycle_SpawnInspectDestroy(t *testing.T) {
	// GIVEN a world spawned with an agent
	ctx := setup.NewTestContext(t)
	ctx.InitAgent("e2e-agent")

	u := ctx.Spawn().
		WithAgent("e2e-agent").
		Execute()

	// THEN the state should show one idle world
	u.ExpectState(func(s *setup.StateAssertion) {
		s.WorldCount(1)
		s.WorldStatus(world.StatusRunning)
	})

	// AND the container should be running with physics, faculties, and mind
	u.ExpectContainer(func(c *setup.ContainerAssertion) {
		c.IsRunning()
		c.HasFile("/world/physics.md")
		c.HasFile("/world/faculties.md")
		c.HasMount("/agents")
	})

	// WHEN inspecting the world
	// THEN it should report the default config
	u.Inspect().ExpectConfig("default")

	// AND listing should show one world
	u.List().ExpectCount(1)

	// WHEN the world is destroyed
	u.Destroy().
		// THEN the state should be empty
		ExpectState(func(s *setup.StateAssertion) {
			s.WorldCount(0)
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
	if _, ok := info.Layers["core"]; !ok {
		t.Fatal("Agent Mind should still have core layer")
	}
}

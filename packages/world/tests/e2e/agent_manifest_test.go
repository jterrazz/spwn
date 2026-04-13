//go:build e2e

package e2e

import (
	"testing"

	"spwn.sh/packages/world/tests/e2e/setup"
)

func TestAgentManifest_Optional(t *testing.T) {
	// GIVEN an agent without an agent.yaml manifest
	// WHEN a world is spawned with that agent
	chain := setup.NewSpawnBuilder(t).
		WithAgent("test-agent").
		Execute()

	// THEN the spawn should succeed and the container should be running
	chain.ExpectState(func(s *setup.StateAssertion) {
		s.WorldCount(1)
	})
	chain.ExpectContainer(func(c *setup.ContainerAssertion) {
		c.IsRunning()
	})
}

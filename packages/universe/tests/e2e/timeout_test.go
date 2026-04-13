//go:build e2e

package e2e

import (
	"testing"

	"spwn.sh/packages/universe/tests/e2e/setup"
)

func TestTimeout_ShortTimeoutStopsContainer(t *testing.T) {
	// BLOCKED: Container-level timeout enforcement is not yet implemented in the architect.
	// The timeout value in physics.constants.timeout is currently only informational
	// (written to physics.md and AGENT.md) and not enforced as a container stop-timeout.
	//
	// When implemented, this test should:
	// 1. Spawn a world with a very short timeout (e.g., 10s)
	// 2. Start a long-running agent (mock-claude sleeps forever)
	// 3. Wait for the timeout to expire
	// 4. Verify the container is no longer running
	// 5. Verify the agent's journal records the timeout
	//
	// Tracking: requires Architect to pass --stop-timeout to Docker or use
	// context.WithTimeout when spawning agents.
	t.Skip("Container timeout enforcement not yet implemented — timeout is informational only")

	// Placeholder for future implementation:
	// chain := setup.NewSpawnBuilder(t).
	//     WithConfigYAML(`
	// physics:
	//   constants:
	//     cpu: 1
	//     memory: 256m
	//     timeout: 10s
	//   tools:
	//     - "@spwn/unix"
	// `).
	//     WithAgent("test-agent").
	//     Detached().
	//     Execute()
	//
	// // Wait for timeout to expire
	// setup.WaitFor(t, 30*time.Second, 1*time.Second, "container to stop after timeout", func() bool {
	//     running, _ := tc.Backend.IsRunning(context.Background(), chain.Universe().ContainerID)
	//     return !running
	// })
	//
	// chain.ExpectJournal(func(j *setup.JournalAssertion) {
	//     j.HasEntries(1)
	//     j.LatestOutcome("timeout")
	// })
}

func TestTimeout_PhysicsMDReflectsTimeout(t *testing.T) {
	// GIVEN a world config with a specific timeout value
	// WHEN the world is spawned
	chain := setup.NewSpawnBuilder(t).
		WithConfigYAML(`
physics:
  constants:
    cpu: 1
    memory: 256m
    timeout: 45m
tools:
  - "@spwn/unix"
`).
		NoAgent().
		Execute()

	// THEN physics.md should contain the timeout value
	chain.ExpectContainer(func(c *setup.ContainerAssertion) {
		c.IsRunning()
		c.FileContains("/world/physics.md", "45m")
	})
}

func TestTimeout_DefaultTimeoutApplied(t *testing.T) {
	// GIVEN a world config without an explicit timeout
	// WHEN the world is spawned
	chain := setup.NewSpawnBuilder(t).
		WithConfigYAML(`
physics:
  constants:
    cpu: 1
    memory: 256m
tools:
  - "@spwn/unix"
`).
		NoAgent().
		Execute()

	// THEN physics.md should contain the default timeout (30m)
	chain.ExpectContainer(func(c *setup.ContainerAssertion) {
		c.IsRunning()
		c.FileContains("/world/physics.md", "30m")
	})
}

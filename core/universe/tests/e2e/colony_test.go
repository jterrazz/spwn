//go:build e2e

package e2e

import (
	"testing"

	"spwn.sh/core/universe/tests/e2e/setup"
)

func TestColony_ChiefAndWorkerSpawn(t *testing.T) {
	// BLOCKED: Multi-agent (colony) spawning via the Architect public API
	// is not yet fully wired. The Architect currently supports single-agent
	// spawn only. When multi-agent support is added (Architect.SpawnColony or
	// similar), this test should:
	// 1. Spawn a world with a chief + worker agent spec
	// 2. Verify both agents' Mind directories are mounted
	// 3. Verify AGENT.md for each role is correct
	// 4. Verify messaging between chief and worker works
	//
	// Tracking: requires Architect.SpawnColony() or multi-agent SpawnOpts.
	t.Skip("Colony multi-agent spawn not yet exposed via Architect public API")

	// Placeholder for future implementation:
	// tc := setup.NewTestContext(t)
	// tc.InitAgent("gov-agent")
	// tc.InitAgent("worker-agent")
	//
	// Write profile.yaml for chief:
	// govDir := filepath.Join(tc.BaseDir, "agents", "gov-agent")
	// os.WriteFile(filepath.Join(govDir, "profile.yaml"), []byte("role: chief\n"), 0644)
	//
	// chain := tc.Spawn().
	//     WithAgent("gov-agent").
	//     Execute()
	//
	// Spawn worker into same world:
	// tc.Arc.SpawnAgent(ctx, chain.Universe().ID, "worker-agent")
	//
	// chain.ExpectContainer(func(c *setup.ContainerAssertion) {
	//     c.HasMount("/mind")
	// })
}

func TestColony_SingleAgentDefaultsToWorker(t *testing.T) {
	// GIVEN an agent without a profile.yaml (no role specified)
	// WHEN spawned into a world
	chain := setup.NewSpawnBuilder(t).
		WithAgent("test-agent").
		Execute()

	// THEN the state should track it (role defaults to worker internally)
	chain.ExpectState(func(s *setup.StateAssertion) {
		s.WorldCount(1)
		s.HasAgent("test-agent")
	})

	// AND the container should be running with Mind mounted
	chain.ExpectContainer(func(c *setup.ContainerAssertion) {
		c.IsRunning()
		c.HasMount("/mind")
	})
}

func TestColony_AgentMindLayersPresent(t *testing.T) {
	// GIVEN an agent with standard Mind structure
	// WHEN spawned
	chain := setup.NewSpawnBuilder(t).
		WithAgent("test-agent").
		Execute()

	// THEN the Mind should have all standard layers
	chain.ExpectMind(func(m *setup.MindAssertion) {
		m.HasLayer("identity")
		m.HasLayer("skills")
		m.HasLayer("memory/knowledge")
		m.HasLayer("memory/playbooks")
		m.HasLayer("memory/journal")
		m.HasLayer("sessions")
	})
}

func TestColony_RoleFromProfileYAML(t *testing.T) {
	// BLOCKED: Role assignment from profile.yaml requires multi-agent colony support.
	// The profile.yaml role field is parsed but not yet used in spawn because
	// the Architect only supports single-agent worlds. When colony spawn is
	// added, verify that the role from profile.yaml is reflected in AgentRecord.
	//
	// Tracking: depends on TestColony_ChiefAndWorkerSpawn being unblocked.
	t.Skip("Colony role assignment from profile.yaml not yet testable via E2E")
}

func TestColony_MessagingBetweenAgents(t *testing.T) {
	// BLOCKED: Colony messaging requires multi-agent support in the Architect.
	// The messaging infrastructure (inbox directories, JSON messages) is implemented
	// in the gate and messenger packages, but cannot be E2E-tested until multi-agent
	// spawn is available.
	//
	// When implemented:
	// 1. Spawn world with chief + worker
	// 2. Chief sends a message to worker via messenger
	// 3. Worker checks inbox and sees the message
	// 4. Worker replies
	// 5. Chief reads the reply
	//
	// Tracking: depends on Architect.SpawnColony() or multi-agent SpawnOpts.
	t.Skip("Colony messaging requires multi-agent spawn — not yet available")
}

//go:build e2e

package e2e

import (
	"testing"

	"spwn.sh/core/universe/tests/e2e/setup"
)

func TestColony_GovernorAndCitizenSpawn(t *testing.T) {
	// TODO: Multi-agent (colony) spawning via the Architect public API
	// is not yet fully wired. When implemented, this test should:
	// 1. Spawn a world with a governor + citizen agent spec
	// 2. Verify both agents' Mind directories are mounted
	// 3. Verify AGENT.md for each tier is correct
	// 4. Verify messaging between governor and citizen works
	t.Skip("Colony multi-agent spawn not yet exposed via Architect public API")

	// Placeholder for future implementation:
	// tc := setup.NewTestContext(t)
	// tc.InitAgent("gov-agent")
	// tc.InitAgent("citizen-agent")
	//
	// Write profile.yaml for governor:
	// govDir := filepath.Join(tc.BaseDir, "agents", "gov-agent")
	// os.WriteFile(filepath.Join(govDir, "profile.yaml"), []byte("tier: governor\n"), 0644)
	//
	// chain := tc.Spawn().
	//     WithAgent("gov-agent").
	//     Execute()
	//
	// Spawn citizen into same world:
	// tc.Arc.SpawnAgent(ctx, chain.Universe().ID, "citizen-agent")
	//
	// chain.ExpectContainer(func(c *setup.ContainerAssertion) {
	//     c.HasMount("/mind")
	// })
}

func TestColony_SingleAgentDefaultsToCitizen(t *testing.T) {
	// GIVEN an agent without a profile.yaml (no tier specified)
	// WHEN spawned into a world
	chain := setup.NewSpawnBuilder(t).
		WithAgent("test-agent").
		Execute()

	// THEN the state should track it (tier defaults to citizen internally)
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

func TestColony_TierFromProfileYAML(t *testing.T) {
	// TODO: Once multi-agent colony is exposed, verify that tier is read
	// from profile.yaml and reflected in the state's AgentRecord.
	t.Skip("Colony tier assignment from profile.yaml not yet testable via E2E")
}

func TestColony_MessagingBetweenAgents(t *testing.T) {
	// TODO: Colony messaging requires multi-agent support in the Architect.
	// When implemented:
	// 1. Spawn world with governor + citizen
	// 2. Governor sends a message to citizen via messenger
	// 3. Citizen checks inbox and sees the message
	// 4. Citizen replies
	// 5. Governor reads the reply
	t.Skip("Colony messaging requires multi-agent spawn — not yet available")
}

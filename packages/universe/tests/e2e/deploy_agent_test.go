//go:build e2e

package e2e

import (
	"strings"
	"testing"

	"spwn.sh/packages/universe/tests/e2e/setup"
)

// TestDeployAgent_ToRunningWorld verifies that an agent can be hot-deployed
// to an already-running world without destroying/recreating it.
func TestDeployAgent_ToRunningWorld(t *testing.T) {
	tc := setup.NewTestContext(t)

	// Create two agents
	tc.InitAgent("agent-a")
	tc.InitAgent("agent-b")

	// Spawn a world with only agent-a
	chain := tc.Spawn().
		WithAgent("agent-a").
		Detached().
		Execute()

	// World should have 1 agent
	chain.ExpectState(func(s *setup.StateAssertion) {
		s.WorldCount(1)
	})

	// Deploy agent-b to the running world
	worldID := chain.Universe().ID
	err := tc.Arc.DeployAgent(t.Context(), worldID, "agent-b", "worker")
	if err != nil {
		t.Fatalf("DeployAgent failed: %v", err)
	}

	// World should now have 2 agents in state
	world, err := tc.Arc.Inspect(t.Context(), worldID)
	if err != nil {
		t.Fatalf("Inspect: %v", err)
	}
	if len(world.Agents) != 2 {
		t.Errorf("expected 2 agents, got %d: %+v", len(world.Agents), world.Agents)
	}

	// Verify agent-b is registered
	found := false
	for _, a := range world.Agents {
		if a.Name == "agent-b" {
			found = true
			if a.Role != "worker" {
				t.Errorf("agent-b role = %q, want worker", a.Role)
			}
		}
	}
	if !found {
		t.Error("agent-b not found in world agents")
	}
}

// TestDeployAgent_AlreadyDeployed should reject duplicate deployment.
func TestDeployAgent_AlreadyDeployed(t *testing.T) {
	tc := setup.NewTestContext(t)
	tc.InitAgent("agent-x")

	chain := tc.Spawn().
		WithAgent("agent-x").
		Detached().
		Execute()

	worldID := chain.Universe().ID
	err := tc.Arc.DeployAgent(t.Context(), worldID, "agent-x", "worker")
	if err == nil || !strings.Contains(err.Error(), "already deployed") {
		t.Errorf("expected 'already deployed' error, got: %v", err)
	}
}

// TestDeployAgent_TalkAfterDeploy verifies that after hot-deploying an
// agent, the talk/exec infrastructure can find and address that agent.
// This is the E2E regression for the "world not found or has no agent"
// bug: handleTalk was only checking u.Agent (empty for hot-deploys) and
// ignoring u.Agents.
func TestDeployAgent_TalkAfterDeploy(t *testing.T) {
	tc := setup.NewTestContext(t)
	tc.InitAgent("talker")

	// Spawn a world with NO agents
	chain := tc.Spawn().
		NoAgent().
		Execute()

	worldID := chain.Universe().ID

	// Deploy the agent
	if err := tc.Arc.DeployAgent(t.Context(), worldID, "talker", "worker"); err != nil {
		t.Fatalf("DeployAgent: %v", err)
	}

	// Verify the agent is in the Agents slice (not the legacy Agent field)
	world, _ := tc.Arc.Inspect(t.Context(), worldID)
	if world.Agent != "" {
		t.Errorf("legacy Agent field should be empty for hot-deploy, got %q", world.Agent)
	}
	if len(world.Agents) != 1 || world.Agents[0].Name != "talker" {
		t.Errorf("expected 1 agent 'talker' in Agents slice, got %+v", world.Agents)
	}
}

// TestDeployAgent_AgentNotFound should reject unknown agents.
func TestDeployAgent_AgentNotFound(t *testing.T) {
	tc := setup.NewTestContext(t)
	tc.InitAgent("real-agent")

	chain := tc.Spawn().
		WithAgent("real-agent").
		NoAgent().
		Execute()

	worldID := chain.Universe().ID
	err := tc.Arc.DeployAgent(t.Context(), worldID, "ghost", "worker")
	if err == nil {
		t.Error("expected error for nonexistent agent")
	}
}

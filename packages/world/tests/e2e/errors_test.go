//go:build e2e

package e2e

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	agentDomain "spwn.sh/packages/mind"
	"spwn.sh/packages/world"
	"spwn.sh/packages/world/tests/e2e/setup"
)

func TestError_SpawnWithInvalidConfigName(t *testing.T) {
	// GIVEN a test context with no configs on disk
	tc := setup.NewTestContext(t)

	// WHEN loading a non-existent manifest directly
	configPath := filepath.Join(tc.BaseDir, "worlds", "nonexistent-config-xyz.yaml")
	_, err := world.LoadManifestPath(configPath)

	// THEN it should fail with a config-related error
	if err == nil {
		t.Fatal("Expected load to fail for non-existent config, got nil")
	}
}

func TestError_SpawnWithNonExistentAgent(t *testing.T) {
	// GIVEN a test context with no such agent
	_ = setup.NewTestContext(t)

	// WHEN validating a non-existent agent
	err := agentDomain.ValidateMind("ghost-agent-does-not-exist")

	// THEN it should return an error
	if err == nil {
		t.Fatal("Expected validate to fail for non-existent agent, got nil")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "agent") && !strings.Contains(err.Error(), "not found") {
		t.Fatalf("Expected agent-related error, got: %v", err)
	}
}

func TestError_DestroyNonExistentWorld(t *testing.T) {
	// GIVEN a test context with no worlds
	tc := setup.NewTestContext(t)

	// WHEN destroying a non-existent world
	_, err := tc.Arc.Destroy(context.Background(), "u-nonexistent-99999")

	// THEN it should return an error
	if err == nil {
		t.Fatal("Expected error when destroying non-existent world, got nil")
	}
}

func TestError_InspectNonExistentWorld(t *testing.T) {
	// GIVEN a test context with no worlds
	tc := setup.NewTestContext(t)

	// WHEN inspecting a non-existent world
	_, err := tc.Arc.Inspect(context.Background(), "u-nonexistent-99999")

	// THEN it should return an error
	if err == nil {
		t.Fatal("Expected error when inspecting non-existent world, got nil")
	}
}


func TestError_DoubleDestroySameWorld(t *testing.T) {
	// GIVEN a world that has been destroyed once
	tc := setup.NewTestContext(t)

	chain := tc.Spawn().
		NoAgent().
		Execute()

	worldID := chain.World().ID

	chain.Destroy().
		ExpectState(func(s *setup.StateAssertion) {
			s.WorldCount(0)
		})

	// WHEN destroying the same world again
	_, err := tc.Arc.Destroy(context.Background(), worldID)

	// THEN it should return an error
	if err == nil {
		t.Fatal("Expected error on second destroy of same world, got nil")
	}
}

func TestError_SpawnAgentDetachedOnNonExistentWorld(t *testing.T) {
	// GIVEN an agent and a non-existent world ID
	tc := setup.NewTestContext(t)
	tc.InitAgent("orphan-agent")

	// WHEN spawning the agent in the non-existent world
	err := tc.Arc.SpawnAgentDetached(context.Background(), "u-nonexistent-99999", "orphan-agent")

	// THEN it should return an error
	if err == nil {
		t.Fatal("Expected error when spawning agent in non-existent world, got nil")
	}
}

func TestError_SpawnAgentOnNonExistentWorld(t *testing.T) {
	// GIVEN an agent and a non-existent world ID
	tc := setup.NewTestContext(t)
	tc.InitAgent("orphan-agent-2")

	// WHEN spawning the agent (blocking) in the non-existent world
	err := tc.Arc.SpawnAgent(context.Background(), "u-nonexistent-99999", "orphan-agent-2")

	// THEN it should return an error
	if err == nil {
		t.Fatal("Expected error when spawning agent in non-existent world, got nil")
	}
}

func TestError_SpawnWithNegativeCPU(t *testing.T) {
	// GIVEN a config with negative CPU count
	// WHEN a world is spawned
	// THEN it should fail with a CPU validation error
	setup.NewSpawnBuilder(t).
		WithConfigYAML(`
physics:
  constants:
    cpu: -1
`).
		NoAgent().
		ExecuteExpectError("CPU")
}

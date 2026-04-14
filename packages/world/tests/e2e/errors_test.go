//go:build e2e

package e2e

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"spwn.sh/packages/agent"
	"spwn.sh/packages/world"
	"spwn.sh/packages/world/tests/e2e/setup"
)

func TestError_SpawnWithInvalidConfigName(t *testing.T) {
	// Given - a test context with no configs on disk
	tc := setup.NewTestContext(t)

	// When - loading a non-existent manifest directly
	configPath := filepath.Join(tc.BaseDir, "worlds", "nonexistent-config-xyz.yaml")
	_, err := world.LoadManifestPath(configPath)

	// Then - it should fail with a config-related error
	if err == nil {
		t.Fatal("Expected load to fail for non-existent config, got nil")
	}
}

func TestError_SpawnWithNonExistentAgent(t *testing.T) {
	// Given - a test context with no such agent
	_ = setup.NewTestContext(t)

	// When - validating a non-existent agent
	err := agent.ValidateMind("ghost-agent-does-not-exist")

	// Then - it should return an error
	if err == nil {
		t.Fatal("Expected validate to fail for non-existent agent, got nil")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "agent") && !strings.Contains(err.Error(), "not found") {
		t.Fatalf("Expected agent-related error, got: %v", err)
	}
}

func TestError_DestroyNonExistentWorld(t *testing.T) {
	// Given - a test context with no worlds
	tc := setup.NewTestContext(t)

	// When - destroying a non-existent world
	_, err := tc.Arc.Destroy(context.Background(), "u-nonexistent-99999")

	// Then - it should return an error
	if err == nil {
		t.Fatal("Expected error when destroying non-existent world, got nil")
	}
}

func TestError_InspectNonExistentWorld(t *testing.T) {
	// Given - a test context with no worlds
	tc := setup.NewTestContext(t)

	// When - inspecting a non-existent world
	_, err := tc.Arc.Inspect(context.Background(), "u-nonexistent-99999")

	// Then - it should return an error
	if err == nil {
		t.Fatal("Expected error when inspecting non-existent world, got nil")
	}
}

func TestError_DoubleDestroySameWorld(t *testing.T) {
	// Given - a world that has been destroyed once
	tc := setup.NewTestContext(t)

	chain := tc.Spawn().
		NoAgent().
		Execute()

	worldID := chain.World().ID

	chain.Destroy().
		ExpectState(func(s *setup.StateAssertion) {
			s.WorldCount(0)
		})

	// When - destroying the same world again
	_, err := tc.Arc.Destroy(context.Background(), worldID)

	// Then - it should return an error
	if err == nil {
		t.Fatal("Expected error on second destroy of same world, got nil")
	}
}

func TestError_SpawnAgentDetachedOnNonExistentWorld(t *testing.T) {
	// Given - an agent and a non-existent world ID
	tc := setup.NewTestContext(t)
	tc.InitAgent("orphan-agent")

	// When - spawning the agent in the non-existent world
	err := tc.Arc.SpawnAgentDetached(context.Background(), "u-nonexistent-99999", "orphan-agent")

	// Then - it should return an error
	if err == nil {
		t.Fatal("Expected error when spawning agent in non-existent world, got nil")
	}
}

func TestError_SpawnAgentOnNonExistentWorld(t *testing.T) {
	// Given - an agent and a non-existent world ID
	tc := setup.NewTestContext(t)
	tc.InitAgent("orphan-agent-2")

	// When - spawning the agent (blocking) in the non-existent world
	err := tc.Arc.SpawnAgent(context.Background(), "u-nonexistent-99999", "orphan-agent-2")

	// Then - it should return an error
	if err == nil {
		t.Fatal("Expected error when spawning agent in non-existent world, got nil")
	}
}

func TestError_SpawnWithNegativeCPU(t *testing.T) {
	// Given - a config with negative CPU count
	// When - a world is spawned
	// Then - it should fail with a CPU validation error
	setup.NewSpawnBuilder(t).
		WithConfigYAML(`
physics:
  constants:
    cpu: -1
`).
		NoAgent().
		ExecuteExpectError("CPU")
}

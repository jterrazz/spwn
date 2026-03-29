//go:build e2e

package e2e

import (
	"context"
	"testing"

	"github.com/jterrazz/spwn/core/universe/tests/e2e/setup"
)

func TestError_SpawnWithInvalidConfigName(t *testing.T) {
	// GIVEN an invalid config name
	// WHEN a universe is spawned
	// THEN it should fail with a config-related error
	setup.NewSpawnBuilder(t).
		WithConfig("nonexistent-config-xyz").
		NoAgent().
		ExecuteExpectError("config")
}

func TestError_SpawnWithNonExistentAgent(t *testing.T) {
	// GIVEN an agent that does not exist
	tc := setup.NewTestContext(t)

	// WHEN a universe is spawned with that agent
	// THEN it should fail with an agent-related error
	tc.Spawn().
		WithConfig("default").
		WithAgent("ghost-agent-does-not-exist").
		ExecuteExpectError("agent")
}

func TestError_DestroyNonExistentUniverse(t *testing.T) {
	// GIVEN a test context with no universes
	tc := setup.NewTestContext(t)

	// WHEN destroying a non-existent universe
	_, err := tc.Arc.Destroy(context.Background(), "u-nonexistent-99999")

	// THEN it should return an error
	if err == nil {
		t.Fatal("Expected error when destroying non-existent universe, got nil")
	}
}

func TestError_InspectNonExistentUniverse(t *testing.T) {
	// GIVEN a test context with no universes
	tc := setup.NewTestContext(t)

	// WHEN inspecting a non-existent universe
	_, err := tc.Arc.Inspect(context.Background(), "u-nonexistent-99999")

	// THEN it should return an error
	if err == nil {
		t.Fatal("Expected error when inspecting non-existent universe, got nil")
	}
}

func TestError_SpawnWithInvalidNetworkLaw(t *testing.T) {
	// GIVEN a config with an invalid network law
	// WHEN a universe is spawned
	// THEN it should fail with a validation error
	setup.NewSpawnBuilder(t).
		WithConfigYAML(`
physics:
  laws:
    network: "bad-network-mode"
`).
		NoAgent().
		ExecuteExpectError("invalid network law")
}

func TestError_SpawnWithInvalidElement(t *testing.T) {
	// GIVEN a config with a non-existent element binary
	// WHEN a universe is spawned
	// THEN it should fail because the image does not provide it
	setup.NewSpawnBuilder(t).
		WithConfigYAML(`
physics:
  elements:
    - "completely-nonexistent-binary-xyz"
`).
		NoAgent().
		ExecuteExpectError("does not provide it")
}

func TestError_DoubleDestroySameUniverse(t *testing.T) {
	// GIVEN a universe that has been destroyed once
	tc := setup.NewTestContext(t)

	chain := tc.Spawn().
		NoAgent().
		Execute()

	universeID := chain.Universe().ID

	chain.Destroy().
		ExpectState(func(s *setup.StateAssertion) {
			s.UniverseCount(0)
		})

	// WHEN destroying the same universe again
	_, err := tc.Arc.Destroy(context.Background(), universeID)

	// THEN it should return an error
	if err == nil {
		t.Fatal("Expected error on second destroy of same universe, got nil")
	}
}

func TestError_LogsOnNonExistentUniverse(t *testing.T) {
	// GIVEN a test context with no universes
	tc := setup.NewTestContext(t)

	// WHEN requesting logs for a non-existent universe
	_, err := tc.Arc.Logs(context.Background(), "u-nonexistent-99999", false, "10")

	// THEN it should return an error
	if err == nil {
		t.Fatal("Expected error when getting logs for non-existent universe, got nil")
	}
}

func TestError_SpawnAgentDetachedOnNonExistentUniverse(t *testing.T) {
	// GIVEN an agent and a non-existent universe ID
	tc := setup.NewTestContext(t)
	tc.InitAgent("orphan-agent")

	// WHEN spawning the agent in the non-existent universe
	err := tc.Arc.SpawnAgentDetached(context.Background(), "u-nonexistent-99999", "orphan-agent")

	// THEN it should return an error
	if err == nil {
		t.Fatal("Expected error when spawning agent in non-existent universe, got nil")
	}
}

func TestError_SpawnAgentOnNonExistentUniverse(t *testing.T) {
	// GIVEN an agent and a non-existent universe ID
	tc := setup.NewTestContext(t)
	tc.InitAgent("orphan-agent-2")

	// WHEN spawning the agent (blocking) in the non-existent universe
	err := tc.Arc.SpawnAgent(context.Background(), "u-nonexistent-99999", "orphan-agent-2")

	// THEN it should return an error
	if err == nil {
		t.Fatal("Expected error when spawning agent in non-existent universe, got nil")
	}
}

func TestError_SpawnWithNegativeCPU(t *testing.T) {
	// GIVEN a config with negative CPU count
	// WHEN a universe is spawned
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

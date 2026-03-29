//go:build e2e

package e2e

import (
	"context"
	"testing"

	"github.com/jterrazz/spwn/core/universe/tests/e2e/setup"
)

func TestError_SpawnWithInvalidConfigName(t *testing.T) {
	setup.NewSpawnBuilder(t).
		WithConfig("nonexistent-config-xyz").
		NoAgent().
		ExecuteExpectError("config")
}

func TestError_SpawnWithNonExistentAgent(t *testing.T) {
	tc := setup.NewTestContext(t)

	// Do NOT init agent — it should not exist
	tc.Spawn().
		WithConfig("default").
		WithAgent("ghost-agent-does-not-exist").
		ExecuteExpectError("agent")
}

func TestError_DestroyNonExistentUniverse(t *testing.T) {
	tc := setup.NewTestContext(t)

	_, err := tc.Arc.Destroy(context.Background(), "u-nonexistent-99999")
	if err == nil {
		t.Fatal("Expected error when destroying non-existent universe, got nil")
	}
}

func TestError_InspectNonExistentUniverse(t *testing.T) {
	tc := setup.NewTestContext(t)

	_, err := tc.Arc.Inspect(context.Background(), "u-nonexistent-99999")
	if err == nil {
		t.Fatal("Expected error when inspecting non-existent universe, got nil")
	}
}

func TestError_SpawnWithInvalidNetworkLaw(t *testing.T) {
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
	tc := setup.NewTestContext(t)

	chain := tc.Spawn().
		NoAgent().
		Execute()

	universeID := chain.Universe().ID

	// First destroy should succeed
	chain.Destroy().
		ExpectState(func(s *setup.StateAssertion) {
			s.UniverseCount(0)
		})

	// Second destroy should fail — universe no longer exists in state
	_, err := tc.Arc.Destroy(context.Background(), universeID)
	if err == nil {
		t.Fatal("Expected error on second destroy of same universe, got nil")
	}
}

func TestError_LogsOnNonExistentUniverse(t *testing.T) {
	tc := setup.NewTestContext(t)

	_, err := tc.Arc.Logs(context.Background(), "u-nonexistent-99999", false, "10")
	if err == nil {
		t.Fatal("Expected error when getting logs for non-existent universe, got nil")
	}
}

func TestError_SpawnAgentDetachedOnNonExistentUniverse(t *testing.T) {
	tc := setup.NewTestContext(t)
	tc.InitAgent("orphan-agent")

	err := tc.Arc.SpawnAgentDetached(context.Background(), "u-nonexistent-99999", "orphan-agent")
	if err == nil {
		t.Fatal("Expected error when spawning agent in non-existent universe, got nil")
	}
}

func TestError_SpawnAgentOnNonExistentUniverse(t *testing.T) {
	tc := setup.NewTestContext(t)
	tc.InitAgent("orphan-agent-2")

	err := tc.Arc.SpawnAgent(context.Background(), "u-nonexistent-99999", "orphan-agent-2")
	if err == nil {
		t.Fatal("Expected error when spawning agent in non-existent universe, got nil")
	}
}

func TestError_SpawnWithNegativeCPU(t *testing.T) {
	setup.NewSpawnBuilder(t).
		WithConfigYAML(`
physics:
  constants:
    cpu: -1
`).
		NoAgent().
		ExecuteExpectError("CPU")
}

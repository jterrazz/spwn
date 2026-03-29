//go:build e2e

package e2e

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jterrazz/spwn/core/universe"
	"github.com/jterrazz/spwn/core/universe/tests/e2e/setup"
)

func TestLife_ManifestOptional(t *testing.T) {
	// GIVEN an agent without a life.yaml manifest
	// WHEN a universe is spawned with that agent
	chain := setup.NewSpawnBuilder(t).
		WithAgent("test-agent").
		Execute()

	// THEN the spawn should succeed and the container should be running
	chain.ExpectState(func(s *setup.StateAssertion) {
		s.UniverseCount(1)
	})
	chain.ExpectContainer(func(c *setup.ContainerAssertion) {
		c.IsRunning()
	})
}

func TestLife_BodyRequiresValidation(t *testing.T) {
	// GIVEN an agent with a life.yaml requiring @node
	tc := setup.NewTestContext(t)
	tc.InitAgent("life-agent")

	agentDir := filepath.Join(tc.BaseDir, "agents", "life-agent")
	lifeYAML := `body:
  requires:
    - "@node"
`
	if err := os.WriteFile(filepath.Join(agentDir, "life.yaml"), []byte(lifeYAML), 0644); err != nil {
		t.Fatalf("Failed to write life.yaml: %v", err)
	}

	// WHEN spawning with the default config (which does not include @node)
	// THEN it should fail because the requirement is not satisfied
	tc.Spawn().
		WithAgent("life-agent").
		ExecuteExpectError("requires element")
}

func TestLife_BodyRequiresSatisfied(t *testing.T) {
	// GIVEN an agent with a life.yaml requiring @unix
	tc := setup.NewTestContext(t)
	tc.InitAgent("life-agent")

	agentDir := filepath.Join(tc.BaseDir, "agents", "life-agent")
	lifeYAML := `body:
  requires:
    - "@unix"
`
	if err := os.WriteFile(filepath.Join(agentDir, "life.yaml"), []byte(lifeYAML), 0644); err != nil {
		t.Fatalf("Failed to write life.yaml: %v", err)
	}

	// WHEN spawning with the default config (which includes @unix)
	chain := tc.Spawn().
		WithAgent("life-agent").
		Execute()

	// THEN the spawn should succeed
	chain.ExpectState(func(s *setup.StateAssertion) {
		s.UniverseCount(1)
		s.UniverseStatus(universe.StatusIdle)
	})
}

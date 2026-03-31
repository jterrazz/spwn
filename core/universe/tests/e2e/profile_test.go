//go:build e2e

package e2e

import (
	"os"
	"path/filepath"
	"testing"

	"spwn.sh/core/universe"
	"spwn.sh/core/universe/tests/e2e/setup"
)

func TestProfile_ManifestOptional(t *testing.T) {
	// GIVEN an agent without a profile.yaml manifest
	// WHEN a universe is spawned with that agent
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

func TestProfile_RequiresValidation(t *testing.T) {
	// GIVEN an agent with a profile.yaml requiring @node
	tc := setup.NewTestContext(t)
	tc.InitAgent("profile-agent")

	agentDir := filepath.Join(tc.BaseDir, "agents", "profile-agent")
	profileYAML := `requires:
  - "@node"
`
	if err := os.WriteFile(filepath.Join(agentDir, "profile.yaml"), []byte(profileYAML), 0644); err != nil {
		t.Fatalf("Failed to write profile.yaml: %v", err)
	}

	// WHEN spawning with the default config (which does not include @node)
	// THEN it should fail because the requirement is not satisfied
	tc.Spawn().
		WithAgent("profile-agent").
		ExecuteExpectError("requires element")
}

func TestProfile_RequiresSatisfied(t *testing.T) {
	// GIVEN an agent with a profile.yaml requiring @unix
	tc := setup.NewTestContext(t)
	tc.InitAgent("profile-agent")

	agentDir := filepath.Join(tc.BaseDir, "agents", "profile-agent")
	profileYAML := `requires:
  - "@unix"
`
	if err := os.WriteFile(filepath.Join(agentDir, "profile.yaml"), []byte(profileYAML), 0644); err != nil {
		t.Fatalf("Failed to write profile.yaml: %v", err)
	}

	// WHEN spawning with the default config (which includes @unix)
	chain := tc.Spawn().
		WithAgent("profile-agent").
		Execute()

	// THEN the spawn should succeed
	chain.ExpectState(func(s *setup.StateAssertion) {
		s.WorldCount(1)
		s.WorldStatus(universe.StatusIdle)
	})
}

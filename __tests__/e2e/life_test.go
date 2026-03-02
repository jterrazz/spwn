//go:build e2e

package e2e

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jterrazz/universe/__tests__/e2e/setup"
	"github.com/jterrazz/universe/internal/config"
)

func TestLife_ManifestOptional(t *testing.T) {
	// No life.yaml → spawn succeeds, agent dir used as-is
	setup.NewSpawnBuilder(t).
		WithAgent("test-agent").
		Execute().
		ExpectState(func(s *setup.StateAssertion) {
			s.UniverseCount(1)
		}).
		ExpectContainer(func(c *setup.ContainerAssertion) {
			c.IsRunning()
		})
}

func TestLife_BodyRequiresValidation(t *testing.T) {
	// Create a test context and init agent with life.yaml requiring @node
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

	// Spawn with default config (no @node) — should fail
	tc.Spawn().
		WithAgent("life-agent").
		ExecuteExpectError("requires element")
}

func TestLife_BodyRequiresSatisfied(t *testing.T) {
	// Create a test context and init agent with life.yaml requiring @unix
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

	// Default config has @unix — should succeed
	tc.Spawn().
		WithAgent("life-agent").
		Execute().
		ExpectState(func(s *setup.StateAssertion) {
			s.UniverseCount(1)
			s.UniverseStatus(config.StatusIdle)
		})
}

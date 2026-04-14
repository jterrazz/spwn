//go:build e2e

package e2e

import (
	"testing"

	"spwn.sh/packages/world/tests/e2e/setup"
)

func TestPhysics_ContainsLaws(t *testing.T) {
	// Given - the default laws configuration
	// When - a world is spawned
	chain := setup.NewSpawnBuilder(t).
		NoAgent().
		Execute()

	// Then - the physics file should document the default network law
	chain.ExpectContainer(func(c *setup.ContainerAssertion) {
		c.HasFile("/world/physics.md")
		c.FileContains("/world/physics.md", "Network: bridge")
	})
}

func TestFaculties_ContainsTools(t *testing.T) {
	// Given - a config with @spwn/unix and @spwn/git tools
	// When - a world is spawned
	chain := setup.NewSpawnBuilder(t).
		WithConfigYAML(`
tools:
  - "@spwn/unix"
  - "@spwn/git"
`).
		NoAgent().
		Execute()

	// Then - the faculties file should list the available tools
	chain.ExpectContainer(func(c *setup.ContainerAssertion) {
		c.HasFile("/world/faculties.md")
		c.FileContains("/world/faculties.md", "Tools")
		c.FileContains("/world/faculties.md", "bash")
		c.FileContains("/world/faculties.md", "git")
	})
}

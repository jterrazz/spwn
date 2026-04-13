//go:build e2e

package e2e

import (
	"testing"

	"spwn.sh/packages/universe/tests/e2e/setup"
)

func TestPhysics_ContainsConstants(t *testing.T) {
	// GIVEN the default physics configuration
	// WHEN a universe is spawned
	chain := setup.NewSpawnBuilder(t).
		NoAgent().
		Execute()

	// THEN the physics file should contain the default constants
	chain.ExpectContainer(func(c *setup.ContainerAssertion) {
		c.HasFile("/world/physics.md")
		c.FileContains("/world/physics.md", "1 core(s)")
		c.FileContains("/world/physics.md", "512m")
		c.FileContains("/world/physics.md", "2g")
		c.FileContains("/world/physics.md", "30m")
	})
}

func TestPhysics_ContainsLaws(t *testing.T) {
	// GIVEN the default laws configuration
	// WHEN a universe is spawned
	chain := setup.NewSpawnBuilder(t).
		NoAgent().
		Execute()

	// THEN the physics file should document the default network law
	chain.ExpectContainer(func(c *setup.ContainerAssertion) {
		c.FileContains("/world/physics.md", "Network: bridge")
	})
}

func TestFaculties_ContainsTools(t *testing.T) {
	// GIVEN a config with @spwn/unix and @spwn/git tools
	// WHEN a universe is spawned
	chain := setup.NewSpawnBuilder(t).
		WithConfigYAML(`
physics:
tools:
  - "@spwn/unix"
  - "@spwn/git"
`).
		NoAgent().
		Execute()

	// THEN the faculties file should list the available tools
	chain.ExpectContainer(func(c *setup.ContainerAssertion) {
		c.HasFile("/world/faculties.md")
		c.FileContains("/world/faculties.md", "Tools")
		c.FileContains("/world/faculties.md", "bash")
		c.FileContains("/world/faculties.md", "git")
	})
}

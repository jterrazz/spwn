//go:build e2e

package e2e

import (
	"testing"

	"github.com/jterrazz/spwn/core/universe/tests/e2e/setup"
)

func TestPhysics_ContainsConstants(t *testing.T) {
	// GIVEN the default physics configuration
	// WHEN a universe is spawned
	chain := setup.NewSpawnBuilder(t).
		NoAgent().
		Execute()

	// THEN the physics file should contain the default constants
	chain.ExpectContainer(func(c *setup.ContainerAssertion) {
		c.HasFile("/universe/physics.md")
		c.FileContains("/universe/physics.md", "1 core(s)")
		c.FileContains("/universe/physics.md", "512m")
		c.FileContains("/universe/physics.md", "2g")
		c.FileContains("/universe/physics.md", "30m")
	})
}

func TestPhysics_ContainsLaws(t *testing.T) {
	// GIVEN the default laws configuration
	// WHEN a universe is spawned
	chain := setup.NewSpawnBuilder(t).
		NoAgent().
		Execute()

	// THEN the physics file should contain the default laws
	chain.ExpectContainer(func(c *setup.ContainerAssertion) {
		c.FileContains("/universe/physics.md", "No outbound network")
		c.FileContains("/universe/physics.md", "128")
	})
}

func TestFaculties_ContainsElements(t *testing.T) {
	// GIVEN a config with @unix and @git elements
	// WHEN a universe is spawned
	chain := setup.NewSpawnBuilder(t).
		WithConfigYAML(`
physics:
  elements:
    - "@unix"
    - "@git"
`).
		NoAgent().
		Execute()

	// THEN the faculties file should list the available elements
	chain.ExpectContainer(func(c *setup.ContainerAssertion) {
		c.HasFile("/universe/faculties.md")
		c.FileContains("/universe/faculties.md", "Elements")
		c.FileContains("/universe/faculties.md", "bash")
		c.FileContains("/universe/faculties.md", "git")
	})
}

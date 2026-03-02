//go:build e2e

package e2e

import (
	"testing"

	"github.com/jterrazz/universe/__tests__/e2e/setup"
)

func TestPhysics_ContainsConstants(t *testing.T) {
	setup.NewSpawnBuilder(t).
		NoAgent().
		Execute().
		ExpectContainer(func(c *setup.ContainerAssertion) {
			c.HasFile("/universe/physics.md")
			c.FileContains("/universe/physics.md", "1 core(s)")
			c.FileContains("/universe/physics.md", "512m")
			c.FileContains("/universe/physics.md", "2g")
			c.FileContains("/universe/physics.md", "30m")
		})
}

func TestPhysics_ContainsLaws(t *testing.T) {
	setup.NewSpawnBuilder(t).
		NoAgent().
		Execute().
		ExpectContainer(func(c *setup.ContainerAssertion) {
			c.FileContains("/universe/physics.md", "No outbound network")
			c.FileContains("/universe/physics.md", "128")
		})
}

func TestFaculties_ContainsTechnologies(t *testing.T) {
	setup.NewSpawnBuilder(t).
		WithConfigYAML(`
physics: {}
technologies:
  - "@unix"
  - "@git"
`).
		NoAgent().
		Execute().
		ExpectContainer(func(c *setup.ContainerAssertion) {
			c.HasFile("/universe/faculties.md")
			c.FileContains("/universe/faculties.md", "Technologies")
			c.FileContains("/universe/faculties.md", "bash")
			c.FileContains("/universe/faculties.md", "git")
		})
}

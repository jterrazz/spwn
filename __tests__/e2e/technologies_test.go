//go:build e2e

package e2e

import (
	"testing"

	"github.com/jterrazz/universe/__tests__/e2e/setup"
)

func TestSpawn_TechnologiesVerified(t *testing.T) {
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
			c.FileContains("/universe/faculties.md", "bash")
			c.FileContains("/universe/faculties.md", "git")
		})
}

func TestSpawn_MissingTechnologyFails(t *testing.T) {
	setup.NewSpawnBuilder(t).
		WithConfigYAML(`
physics: {}
technologies:
  - totally-fake-binary
`).
		NoAgent().
		ExecuteExpectError("does not provide it")
}

func TestSpawn_PackExpansion(t *testing.T) {
	setup.NewSpawnBuilder(t).
		WithConfigYAML(`
physics: {}
technologies:
  - "@unix"
`).
		NoAgent().
		Execute().
		ExpectContainer(func(c *setup.ContainerAssertion) {
			c.FileContains("/universe/faculties.md", "bash")
			c.FileContains("/universe/faculties.md", "grep")
			c.FileContains("/universe/faculties.md", "curl")
		})
}

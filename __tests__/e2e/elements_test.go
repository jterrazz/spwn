//go:build e2e

package e2e

import (
	"testing"

	"github.com/jterrazz/universe/__tests__/e2e/setup"
)

func TestSpawn_ElementsVerified(t *testing.T) {
	setup.NewSpawnBuilder(t).
		WithConfigYAML(`
physics:
  elements:
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

func TestSpawn_MissingElementFails(t *testing.T) {
	setup.NewSpawnBuilder(t).
		WithConfigYAML(`
physics:
  elements:
    - totally-fake-binary
`).
		NoAgent().
		ExecuteExpectError("does not provide it")
}

func TestSpawn_PackExpansion(t *testing.T) {
	setup.NewSpawnBuilder(t).
		WithConfigYAML(`
physics:
  elements:
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

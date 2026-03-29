//go:build e2e

package e2e

import (
	"testing"

	"spwn.sh/core/universe/tests/e2e/setup"
)

func TestSpawn_ElementsVerified(t *testing.T) {
	// GIVEN a config requesting @unix and @git elements
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

	// THEN the faculties file should list bash and git capabilities
	chain.ExpectContainer(func(c *setup.ContainerAssertion) {
		c.FileContains("/universe/faculties.md", "bash")
		c.FileContains("/universe/faculties.md", "git")
	})
}

func TestSpawn_MissingElementFails(t *testing.T) {
	// GIVEN a config requesting a non-existent element
	// WHEN a universe is spawned
	// THEN it should fail with an error about the missing element
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
	// GIVEN a config requesting the @unix pack
	// WHEN a universe is spawned
	chain := setup.NewSpawnBuilder(t).
		WithConfigYAML(`
physics:
  elements:
    - "@unix"
`).
		NoAgent().
		Execute()

	// THEN the faculties should include all @unix pack members
	chain.ExpectContainer(func(c *setup.ContainerAssertion) {
		c.FileContains("/universe/faculties.md", "bash")
		c.FileContains("/universe/faculties.md", "grep")
		c.FileContains("/universe/faculties.md", "curl")
	})
}

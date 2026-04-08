//go:build e2e

package e2e

import (
	"testing"

	"spwn.sh/core/universe/tests/e2e/setup"
)

func TestSpawn_ToolsVerified(t *testing.T) {
	// GIVEN a config requesting @spwn/unix and @spwn/git tools
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

	// THEN the faculties file should list bash and git capabilities
	chain.ExpectContainer(func(c *setup.ContainerAssertion) {
		c.FileContains("/universe/faculties.md", "bash")
		c.FileContains("/universe/faculties.md", "git")
	})
}

func TestSpawn_MissingToolFails(t *testing.T) {
	// GIVEN a config requesting a non-existent tool
	// WHEN a universe is spawned
	// THEN it should fail with an error about the missing tool
	setup.NewSpawnBuilder(t).
		WithConfigYAML(`
physics:
  tools:
    - totally-fake-binary
`).
		NoAgent().
		ExecuteExpectError("does not provide it")
}

func TestSpawn_PackExpansion(t *testing.T) {
	// GIVEN a config requesting the @spwn/unix pack
	// WHEN a universe is spawned
	chain := setup.NewSpawnBuilder(t).
		WithConfigYAML(`
physics:
  tools:
    - "@spwn/unix"
`).
		NoAgent().
		Execute()

	// THEN the faculties should include all @spwn/unix pack members
	chain.ExpectContainer(func(c *setup.ContainerAssertion) {
		c.FileContains("/universe/faculties.md", "bash")
		c.FileContains("/universe/faculties.md", "grep")
		c.FileContains("/universe/faculties.md", "curl")
	})
}

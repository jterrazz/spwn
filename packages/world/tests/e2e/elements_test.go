//go:build e2e

package e2e

import (
	"testing"

	"spwn.sh/packages/world/tests/e2e/setup"
)

func TestSpawn_ToolsVerified(t *testing.T) {
	// Given - a config requesting @spwn/unix and @spwn/git tools
	// When - a world is spawned
	chain := setup.NewSpawnBuilder(t).
		WithConfigYAML(`
physics:
  constants:
    cpu: 1
    memory: 512m
tools:
  - "@spwn/unix"
  - "@spwn/git"
`).
		NoAgent().
		Execute()

	// Then - the faculties file should list bash and git capabilities
	chain.ExpectContainer(func(c *setup.ContainerAssertion) {
		c.FileContains("/world/faculties.md", "bash")
		c.FileContains("/world/faculties.md", "git")
	})
}

func TestSpawn_MissingToolFails(t *testing.T) {
	// Given - a config requesting a non-existent tool
	// When - a world is spawned
	// Then - it should fail with an error about the missing tool
	setup.NewSpawnBuilder(t).
		WithConfigYAML(`
physics:
  constants:
    cpu: 1
    memory: 512m
tools:
  - totally-fake-binary
`).
		NoAgent().
		ExecuteExpectError("does not provide it")
}

func TestSpawn_PackExpansion(t *testing.T) {
	// Given - a config requesting the @spwn/unix pack
	// When - a world is spawned
	chain := setup.NewSpawnBuilder(t).
		WithConfigYAML(`
physics:
  constants:
    cpu: 1
    memory: 512m
tools:
  - "@spwn/unix"
`).
		NoAgent().
		Execute()

	// Then - the faculties should include all @spwn/unix pack members
	chain.ExpectContainer(func(c *setup.ContainerAssertion) {
		c.FileContains("/world/faculties.md", "bash")
		c.FileContains("/world/faculties.md", "grep")
		c.FileContains("/world/faculties.md", "curl")
	})
}

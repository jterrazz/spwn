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
dependencies:
  - "@spwn/unix"
  - "@spwn/git"
`).
		NoAgent().
		Execute()

	// Then - the faculties file lists the verified tools by ref.
	// Binaries live under each tool's Verify() spec, not in faculties.md.
	chain.ExpectContainer(func(c *setup.ContainerAssertion) {
		c.FileContains("/world/faculties.md", "@spwn/unix")
		c.FileContains("/world/faculties.md", "@spwn/git")
	})
}

func TestSpawn_MissingToolFails(t *testing.T) {
	// Given - a config requesting a non-existent tool
	// When - a world is spawned
	// Then - it should fail with "tool not found" (image.Registry.Resolve)
	setup.NewSpawnBuilder(t).
		WithConfigYAML(`
dependencies:
  - totally-fake-binary
`).
		NoAgent().
		ExecuteExpectError("tool not found")
}

func TestSpawn_PackExpansion(t *testing.T) {
	// Given - a config requesting the @spwn/unix dependency
	// When - a world is spawned
	chain := setup.NewSpawnBuilder(t).
		WithConfigYAML(`
dependencies:
  - "@spwn/unix"
`).
		NoAgent().
		Execute()

	// Then - the faculties file lists @spwn/unix as a verified tool.
	chain.ExpectContainer(func(c *setup.ContainerAssertion) {
		c.FileContains("/world/faculties.md", "@spwn/unix")
	})
}

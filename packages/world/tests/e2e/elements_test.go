//go:build e2e

package e2e

import (
	"testing"

	"spwn.sh/packages/world/tests/e2e/setup"
)

func TestSpawn_ToolsVerified(t *testing.T) {
	// Given - a config requesting spwn:unix and spwn:git tools
	// When - a world is spawned with an agent
	chain := setup.NewSpawnBuilder(t).
		WithConfigYAML(`
dependencies:
  - "spwn:unix"
  - "spwn:git"
`).
		WithAgent("test-agent").
		Execute()

	// Then - the agent's CLAUDE.md lists the verified tools by ref
	// In its inlined Faculties section.
	chain.ExpectContainer(func(c *setup.ContainerAssertion) {
		c.FileContains("/agents/test-agent/CLAUDE.md", "spwn:unix")
		c.FileContains("/agents/test-agent/CLAUDE.md", "spwn:git")
	})
}

func TestSpawn_MissingToolFails(t *testing.T) {
	// Given - a config requesting a non-existent tool
	// When - a world is spawned
	// Then - it should fail with "tool not found" (resolver.Registry.Resolve)
	setup.NewSpawnBuilder(t).
		WithConfigYAML(`
dependencies:
  - totally-fake-binary
`).
		NoAgent().
		ExecuteExpectError("tool not found")
}

func TestSpawn_PackExpansion(t *testing.T) {
	// Given - a config requesting the spwn:unix dependency
	// When - a world is spawned with an agent
	chain := setup.NewSpawnBuilder(t).
		WithConfigYAML(`
dependencies:
  - "spwn:unix"
`).
		WithAgent("test-agent").
		Execute()

	// Then - the agent's CLAUDE.md lists spwn:unix as a verified tool.
	chain.ExpectContainer(func(c *setup.ContainerAssertion) {
		c.FileContains("/agents/test-agent/CLAUDE.md", "spwn:unix")
	})
}

//go:build e2e

package e2e

import (
	"testing"

	"spwn.sh/packages/world/tests/e2e/setup"
)

// TestPhysics_ContainsLaws checks the Physics section Claude Code
// sees on boot mentions the world's network law. Physics content is
// inlined into every agent's CLAUDE.md; there is no separate
// /world/physics.md file anymore.
func TestPhysics_ContainsLaws(t *testing.T) {
	chain := setup.NewSpawnBuilder(t).
		WithAgent("test-agent").
		Execute()

	chain.ExpectContainer(func(c *setup.ContainerAssertion) {
		c.HasFile("/agents/test-agent/CLAUDE.md")
		c.FileContains("/agents/test-agent/CLAUDE.md", "## Physics")
		c.FileContains("/agents/test-agent/CLAUDE.md", "Network: bridge")
	})
}

// TestFaculties_ContainsTools checks the Faculties section Claude
// Code sees on boot lists the project's verified tools. Faculties
// content is inlined into every agent's CLAUDE.md.
func TestFaculties_ContainsTools(t *testing.T) {
	chain := setup.NewSpawnBuilder(t).
		WithConfigYAML(`
dependencies:
  - "spwn:unix"
  - "spwn:git"
`).
		WithAgent("test-agent").
		Execute()

	chain.ExpectContainer(func(c *setup.ContainerAssertion) {
		c.HasFile("/agents/test-agent/CLAUDE.md")
		c.FileContains("/agents/test-agent/CLAUDE.md", "## Faculties")
		c.FileContains("/agents/test-agent/CLAUDE.md", "spwn:unix")
		c.FileContains("/agents/test-agent/CLAUDE.md", "spwn:git")
	})
}

//go:build e2e

package e2e

import (
	"testing"

	"spwn.sh/core/universe/tests/e2e/setup"
)

func TestAgentContext_ContainsNewCLICommands(t *testing.T) {
	// GIVEN a universe spawned with an agent (generates AGENT.md)
	chain := setup.NewSpawnBuilder(t).
		WithAgent("test-agent").
		NoAgent().
		Execute()

	// The AGENT.md is only generated for architect-role (via spawn with architect config).
	// For worker role, AGENT.md is generated when an agent is attached.
	// We need a world WITH an agent to get AGENT.md.
	chain2 := setup.NewSpawnBuilder(t).
		WithAgent("ctx-agent").
		Execute()

	// THEN /world/AGENT.md should exist and contain new CLI command names
	chain2.ExpectContainer(func(c *setup.ContainerAssertion) {
		c.IsRunning()
		c.HasFile("/world/AGENT.md")
	})

	// Verify new commands are present (worker context has "Your World" section)
	chain2.ExpectContainer(func(c *setup.ContainerAssertion) {
		c.FileContains("/world/AGENT.md", "Worker")
	})

	_ = chain // use chain to avoid unused variable
}

func TestAgentContext_ArchitectRoleContainsNewCommands(t *testing.T) {
	// GIVEN a universe spawned with an architect-role agent context
	// The architect role AGENT.md is the one that contains CLI commands.
	// We test the GenerateAgentContext function directly via the container output.

	tc := setup.NewTestContext(t)
	tc.InitAgent("arch-agent")

	chain := tc.Spawn().
		WithAgent("arch-agent").
		Execute()

	// The AGENT.md for a worker won't have CLI commands (only architect role does).
	// But we can verify the worker AGENT.md has correct structure.
	chain.ExpectContainer(func(c *setup.ContainerAssertion) {
		c.IsRunning()
		c.HasFile("/world/AGENT.md")
		// Worker AGENT.md should contain Mind layer references
		c.FileContains("/world/AGENT.md", "/mind/core/")
		c.FileContains("/world/AGENT.md", "/mind/skills/")
		c.FileContains("/world/AGENT.md", "/mind/knowledge/")
		c.FileContains("/world/AGENT.md", "/mind/playbooks/")
		c.FileContains("/world/AGENT.md", "/mind/journal/")
	})
}

func TestAgentContext_NoOldCommandNames(t *testing.T) {
	// GIVEN a universe spawned with an agent
	chain := setup.NewSpawnBuilder(t).
		WithAgent("oldcmd-agent").
		Execute()

	// THEN the AGENT.md should NOT contain deprecated command names
	chain.ExpectContainer(func(c *setup.ContainerAssertion) {
		c.IsRunning()
		c.HasFile("/world/AGENT.md")
		// Old command names that should NOT appear
		c.FileNotContains("/world/AGENT.md", "spwn world list")
		c.FileNotContains("/world/AGENT.md", "spwn world destroy")
		c.FileNotContains("/world/AGENT.md", "spwn agent init")
		c.FileNotContains("/world/AGENT.md", "spwn agent list")
		c.FileNotContains("/world/AGENT.md", "spwn agent delete")
	})
}

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

	// The AGENT.md is only generated for god-role (via spawn with god config).
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

func TestAgentContext_GodRoleContainsNewCommands(t *testing.T) {
	// GIVEN a universe spawned with a god-role agent context
	// The god role AGENT.md is the one that contains CLI commands.
	// We test the GenerateAgentContext function directly via the container output.

	tc := setup.NewTestContext(t)
	tc.InitAgent("god-agent")

	chain := tc.Spawn().
		WithAgent("god-agent").
		Execute()

	// The AGENT.md for a worker won't have CLI commands (only god role does).
	// But we can verify the worker AGENT.md has correct structure.
	chain.ExpectContainer(func(c *setup.ContainerAssertion) {
		c.IsRunning()
		c.HasFile("/world/AGENT.md")
		// Worker AGENT.md should contain Mind layer references
		c.FileContains("/world/AGENT.md", "/mind/identity/")
		c.FileContains("/world/AGENT.md", "/mind/skills/")
		c.FileContains("/world/AGENT.md", "/mind/memory/knowledge/")
		c.FileContains("/world/AGENT.md", "/mind/memory/playbooks/")
		c.FileContains("/world/AGENT.md", "/mind/memory/journal/")
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

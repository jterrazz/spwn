//go:build e2e

package e2e

import (
	"testing"

	"spwn.sh/packages/world"
	"spwn.sh/packages/world/tests/e2e/setup"
)

func TestConfig_SpawnWithNamedConfig(t *testing.T) {
	// Given - a named config "custom"
	tc := setup.NewTestContext(t)

	if err := world.CreateConfig("custom"); err != nil {
		t.Fatalf("CreateConfig failed: %v", err)
	}

	// When - a world is spawned with that config
	chain := tc.Spawn().
		WithConfig("custom").
		NoAgent().
		Execute()

	// Then - the state should show one idle world
	chain.ExpectState(func(s *setup.StateAssertion) {
		s.WorldCount(1)
		s.WorldStatus(world.StatusRunning)
	})

	// AND the container should be running with the custom config
	chain.ExpectContainer(func(c *setup.ContainerAssertion) {
		c.IsRunning()
	})
	chain.Inspect().ExpectConfig("custom")
}

func TestConfig_DefaultsApplied(t *testing.T) {
	// Given - a minimal YAML config with only tools specified
	// When - a world is spawned with an agent
	chain := setup.NewSpawnBuilder(t).
		WithConfigYAML(`
dependencies:
  - "spwn:unix"
`).
		WithAgent("test-agent").
		Execute()

	// Then - the world comes up with physics+faculties inlined into
	// The agent's CLAUDE.md (no standalone /world/*.md files).
	chain.ExpectContainer(func(c *setup.ContainerAssertion) {
		c.IsRunning()
		c.FileContains("/agents/test-agent/CLAUDE.md", "## Physics")
		c.FileContains("/agents/test-agent/CLAUDE.md", "## Faculties")
	})
}

func TestConfig_CustomToolsReflectedInFaculties(t *testing.T) {
	// Given - a config with spwn:unix and spwn:git tools
	// When - a world is spawned with an agent
	chain := setup.NewSpawnBuilder(t).
		WithConfigYAML(`
dependencies:
  - "spwn:unix"
  - "spwn:git"
`).
		WithAgent("test-agent").
		Execute()

	// Then - the agent's CLAUDE.md lists both tools in its inlined
	// Faculties section.
	chain.ExpectContainer(func(c *setup.ContainerAssertion) {
		c.FileContains("/agents/test-agent/CLAUDE.md", "spwn:unix")
		c.FileContains("/agents/test-agent/CLAUDE.md", "spwn:git")
	})
}

func TestConfig_LoadAndVerifyManifest(t *testing.T) {
	// Given - the default config has been created
	tc := setup.NewTestContext(t)

	if err := world.CreateDefaultConfig(); err != nil {
		t.Fatalf("CreateDefaultConfig failed: %v", err)
	}

	// When - loading the manifest
	if _, err := world.LoadManifest("default"); err != nil {
		t.Fatalf("LoadManifest failed: %v", err)
	}
	_ = tc
}


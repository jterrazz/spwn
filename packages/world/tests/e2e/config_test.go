//go:build e2e

package e2e

import (
	"testing"

	"spwn.sh/packages/world"
	"spwn.sh/packages/world/tests/e2e/setup"
)

func TestConfig_SpawnWithNamedConfig(t *testing.T) {
	// GIVEN a named config "custom"
	tc := setup.NewTestContext(t)

	if err := world.CreateConfig("custom"); err != nil {
		t.Fatalf("CreateConfig failed: %v", err)
	}

	// WHEN a world is spawned with that config
	chain := tc.Spawn().
		WithConfig("custom").
		NoAgent().
		Execute()

	// THEN the state should show one idle world
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
	// GIVEN a minimal YAML config with only tools specified
	// WHEN a world is spawned
	chain := setup.NewSpawnBuilder(t).
		WithConfigYAML(`
tools:
  - "@spwn/unix"
`).
		NoAgent().
		Execute()

	// THEN the world should come up with physics/faculties files present
	chain.ExpectContainer(func(c *setup.ContainerAssertion) {
		c.IsRunning()
		c.HasFile("/world/physics.md")
		c.HasFile("/world/faculties.md")
	})
}

func TestConfig_CustomToolsReflectedInFaculties(t *testing.T) {
	// GIVEN a config with @spwn/unix and @spwn/git tools
	// WHEN a world is spawned
	chain := setup.NewSpawnBuilder(t).
		WithConfigYAML(`
tools:
  - "@spwn/unix"
  - "@spwn/git"
`).
		NoAgent().
		Execute()

	// THEN the faculties file should list bash and git
	chain.ExpectContainer(func(c *setup.ContainerAssertion) {
		c.HasFile("/world/faculties.md")
		c.FileContains("/world/faculties.md", "bash")
		c.FileContains("/world/faculties.md", "git")
	})
}

func TestConfig_LoadAndVerifyManifest(t *testing.T) {
	// GIVEN the default config has been created
	tc := setup.NewTestContext(t)

	if err := world.CreateDefaultConfig(); err != nil {
		t.Fatalf("CreateDefaultConfig failed: %v", err)
	}

	// WHEN loading the manifest
	if _, err := world.LoadManifest("default"); err != nil {
		t.Fatalf("LoadManifest failed: %v", err)
	}
	_ = tc
}


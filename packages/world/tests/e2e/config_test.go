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
physics:
tools:
  - "@spwn/unix"
`).
		NoAgent().
		Execute()

	// THEN default constants should be applied
	chain.ExpectContainer(func(c *setup.ContainerAssertion) {
		c.IsRunning()
		c.HasFile("/world/physics.md")
		c.HasFile("/world/faculties.md")
		c.FileContains("/world/physics.md", "core(s)")
		c.FileContains("/world/physics.md", "m")
	})
}

func TestConfig_CustomCPUReflectedInPhysics(t *testing.T) {
	// GIVEN a config with custom CPU and memory
	// WHEN a world is spawned
	chain := setup.NewSpawnBuilder(t).
		WithConfigYAML(`
physics:
  constants:
    cpu: 4
    memory: 2g
`).
		NoAgent().
		Execute()

	// THEN the physics file should reflect the custom values
	chain.ExpectContainer(func(c *setup.ContainerAssertion) {
		c.FileContains("/world/physics.md", "4 core(s)")
		c.FileContains("/world/physics.md", "2g")
	})
}

func TestConfig_CustomToolsReflectedInFaculties(t *testing.T) {
	// GIVEN a config with @spwn/unix and @spwn/git tools
	// WHEN a world is spawned
	chain := setup.NewSpawnBuilder(t).
		WithConfigYAML(`
physics:
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
	m, err := world.LoadManifest("default")
	if err != nil {
		t.Fatalf("LoadManifest failed: %v", err)
	}

	// THEN the defaults should have reasonable values
	if m.Physics.Constants.CPU == 0 {
		t.Fatal("Expected non-zero CPU in default config")
	}
	if m.Physics.Constants.Memory == "" {
		t.Fatal("Expected non-empty memory in default config")
	}
	_ = tc
}


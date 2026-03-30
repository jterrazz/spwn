//go:build e2e

package e2e

import (
	"testing"

	"spwn.sh/core/gate"
	"spwn.sh/core/universe"
	"spwn.sh/core/universe/tests/e2e/setup"
)

func TestConfig_SpawnWithNamedConfig(t *testing.T) {
	// GIVEN a named config "custom"
	tc := setup.NewTestContext(t)

	if err := universe.CreateConfig("custom"); err != nil {
		t.Fatalf("CreateConfig failed: %v", err)
	}

	// WHEN a universe is spawned with that config
	chain := tc.Spawn().
		WithConfig("custom").
		NoAgent().
		Execute()

	// THEN the state should show one idle world
	chain.ExpectState(func(s *setup.StateAssertion) {
		s.WorldCount(1)
		s.WorldStatus(universe.StatusIdle)
	})

	// AND the container should be running with the custom config
	chain.ExpectContainer(func(c *setup.ContainerAssertion) {
		c.IsRunning()
	})
	chain.Inspect().ExpectConfig("custom")
}

func TestConfig_DefaultsApplied(t *testing.T) {
	// GIVEN a minimal YAML config with only elements specified
	// WHEN a universe is spawned
	chain := setup.NewSpawnBuilder(t).
		WithConfigYAML(`
physics:
  elements:
    - "@unix"
`).
		NoAgent().
		Execute()

	// THEN default constants should be applied
	chain.ExpectContainer(func(c *setup.ContainerAssertion) {
		c.IsRunning()
		c.HasFile("/universe/physics.md")
		c.HasFile("/universe/faculties.md")
		c.FileContains("/universe/physics.md", "core(s)")
		c.FileContains("/universe/physics.md", "m")
	})
}

func TestConfig_CustomCPUReflectedInPhysics(t *testing.T) {
	// GIVEN a config with custom CPU, memory, disk, and timeout
	// WHEN a universe is spawned
	chain := setup.NewSpawnBuilder(t).
		WithConfigYAML(`
physics:
  constants:
    cpu: 4
    memory: 2g
    disk: 8g
    timeout: 120m
`).
		NoAgent().
		Execute()

	// THEN the physics file should reflect the custom values
	chain.ExpectContainer(func(c *setup.ContainerAssertion) {
		c.FileContains("/universe/physics.md", "4 core(s)")
		c.FileContains("/universe/physics.md", "2g")
		c.FileContains("/universe/physics.md", "8g")
		c.FileContains("/universe/physics.md", "120m")
	})
}

func TestConfig_CustomElementsReflectedInFaculties(t *testing.T) {
	// GIVEN a config with @unix and @git elements
	// WHEN a universe is spawned
	chain := setup.NewSpawnBuilder(t).
		WithConfigYAML(`
physics:
  elements:
    - "@unix"
    - "@git"
`).
		NoAgent().
		Execute()

	// THEN the faculties file should list bash and git
	chain.ExpectContainer(func(c *setup.ContainerAssertion) {
		c.HasFile("/universe/faculties.md")
		c.FileContains("/universe/faculties.md", "bash")
		c.FileContains("/universe/faculties.md", "git")
	})
}

func TestConfig_CustomMaxProcesses(t *testing.T) {
	// GIVEN a config with a custom max-processes law
	// WHEN a universe is spawned
	chain := setup.NewSpawnBuilder(t).
		WithConfigYAML(`
physics:
  laws:
    max-processes: 256
`).
		NoAgent().
		Execute()

	// THEN the physics file should contain the custom process limit
	chain.ExpectContainer(func(c *setup.ContainerAssertion) {
		c.FileContains("/universe/physics.md", "256")
	})
}

func TestConfig_GateBridgesInConfig(t *testing.T) {
	// GIVEN a config with a gate bridge
	bridge := gate.Bridge{
		Source:       "mcp/config-tool",
		As:           "config-bridge",
		Capabilities: []string{"read", "write"},
	}

	// WHEN a universe is spawned with the bridge
	chain := setup.NewSpawnBuilder(t).
		NoAgent().
		WithGate(bridge).
		Execute()

	// THEN the bridge should be installed and executable
	chain.ExpectGate(func(g *setup.GateAssertion) {
		g.HasBridge("config-bridge")
		g.BridgeIsExecutable("config-bridge")
	})

	// AND the faculties should reference the bridge
	chain.ExpectContainer(func(c *setup.ContainerAssertion) {
		c.FileContains("/universe/faculties.md", "config-bridge")
	})
}

func TestConfig_LoadAndVerifyManifest(t *testing.T) {
	// GIVEN the default config has been created
	tc := setup.NewTestContext(t)

	if err := universe.CreateDefaultConfig(); err != nil {
		t.Fatalf("CreateDefaultConfig failed: %v", err)
	}

	// WHEN loading the manifest
	m, err := universe.LoadManifest("default")
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


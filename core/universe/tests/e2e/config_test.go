//go:build e2e

package e2e

import (
	"testing"

	"github.com/jterrazz/spwn/core/gate"
	"github.com/jterrazz/spwn/core/universe"
	"github.com/jterrazz/spwn/core/universe/tests/e2e/setup"
)

func TestConfig_SpawnWithNamedConfig(t *testing.T) {
	tc := setup.NewTestContext(t)

	// Create a named config
	if err := universe.CreateConfig("custom"); err != nil {
		t.Fatalf("CreateConfig failed: %v", err)
	}

	tc.Spawn().
		WithConfig("custom").
		NoAgent().
		Execute().
		ExpectState(func(s *setup.StateAssertion) {
			s.UniverseCount(1)
			s.UniverseStatus(universe.StatusIdle)
		}).
		ExpectContainer(func(c *setup.ContainerAssertion) {
			c.IsRunning()
		}).
		Inspect().
		ExpectConfig("custom")
}

func TestConfig_DefaultsApplied(t *testing.T) {
	// Spawn with minimal YAML — defaults should fill in the rest
	setup.NewSpawnBuilder(t).
		WithConfigYAML(`
physics:
  elements:
    - "@unix"
`).
		NoAgent().
		Execute().
		ExpectContainer(func(c *setup.ContainerAssertion) {
			c.IsRunning()
			c.HasFile("/universe/physics.md")
			c.HasFile("/universe/faculties.md")
			// Defaults should be applied for constants
			c.FileContains("/universe/physics.md", "core(s)")
			c.FileContains("/universe/physics.md", "m")
		})
}

func TestConfig_CustomCPUReflectedInPhysics(t *testing.T) {
	setup.NewSpawnBuilder(t).
		WithConfigYAML(`
physics:
  constants:
    cpu: 4
    memory: 2g
    disk: 8g
    timeout: 120m
`).
		NoAgent().
		Execute().
		ExpectContainer(func(c *setup.ContainerAssertion) {
			c.FileContains("/universe/physics.md", "4 core(s)")
			c.FileContains("/universe/physics.md", "2g")
			c.FileContains("/universe/physics.md", "8g")
			c.FileContains("/universe/physics.md", "120m")
		})
}

func TestConfig_CustomElementsReflectedInFaculties(t *testing.T) {
	setup.NewSpawnBuilder(t).
		WithConfigYAML(`
physics:
  elements:
    - "@unix"
    - "@git"
`).
		NoAgent().
		Execute().
		ExpectContainer(func(c *setup.ContainerAssertion) {
			c.HasFile("/universe/faculties.md")
			c.FileContains("/universe/faculties.md", "bash")
			c.FileContains("/universe/faculties.md", "git")
		})
}

func TestConfig_CustomNetworkLawReflectedInPhysics(t *testing.T) {
	setup.NewSpawnBuilder(t).
		WithConfigYAML(`
physics:
  laws:
    network: bridge
`).
		NoAgent().
		Execute().
		ExpectContainer(func(c *setup.ContainerAssertion) {
			c.HasFile("/universe/physics.md")
			// Bridge mode should not say "No outbound network"
			c.FileContains("/universe/physics.md", "bridge")
		})
}

func TestConfig_CustomMaxProcesses(t *testing.T) {
	setup.NewSpawnBuilder(t).
		WithConfigYAML(`
physics:
  laws:
    network: none
    max-processes: 256
`).
		NoAgent().
		Execute().
		ExpectContainer(func(c *setup.ContainerAssertion) {
			c.FileContains("/universe/physics.md", "256")
		})
}

func TestConfig_GateBridgesInConfig(t *testing.T) {
	bridge := gate.Bridge{
		Source:       "mcp/config-tool",
		As:           "config-bridge",
		Capabilities: []string{"read", "write"},
	}

	setup.NewSpawnBuilder(t).
		NoAgent().
		WithGate(bridge).
		Execute().
		ExpectGate(func(g *setup.GateAssertion) {
			g.HasBridge("config-bridge")
			g.BridgeIsExecutable("config-bridge")
		}).
		ExpectContainer(func(c *setup.ContainerAssertion) {
			c.FileContains("/universe/faculties.md", "config-bridge")
		})
}

func TestConfig_LoadAndVerifyManifest(t *testing.T) {
	tc := setup.NewTestContext(t)

	if err := universe.CreateDefaultConfig(); err != nil {
		t.Fatalf("CreateDefaultConfig failed: %v", err)
	}

	m, err := universe.LoadManifest("default")
	if err != nil {
		t.Fatalf("LoadManifest failed: %v", err)
	}

	// Verify defaults are reasonable
	if m.Physics.Constants.CPU == 0 {
		t.Fatal("Expected non-zero CPU in default config")
	}
	if m.Physics.Constants.Memory == "" {
		t.Fatal("Expected non-empty memory in default config")
	}
	if m.Physics.Laws.Network == "" {
		t.Fatal("Expected non-empty network law in default config")
	}

	_ = tc
}

func TestConfig_ValidateRejectsInvalidManifest(t *testing.T) {
	m := universe.Manifest{
		Physics: universe.PhysicsManifest{
			Laws: universe.LawsManifest{
				Network: "invalid-mode",
			},
		},
	}

	err := universe.ValidateManifest(m)
	if err == nil {
		t.Fatal("Expected validation error for invalid network mode, got nil")
	}
}

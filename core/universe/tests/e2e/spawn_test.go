//go:build e2e

package e2e

import (
	"path/filepath"
	"testing"

	"spwn.sh/core/universe"
	"spwn.sh/core/universe/tests/e2e/setup"
)

func TestSpawn_DefaultConfig(t *testing.T) {
	// GIVEN the default universe configuration
	// WHEN a universe is spawned with an agent
	chain := setup.NewSpawnBuilder(t).
		WithAgent("test-agent").
		Execute()

	// THEN the state should contain one idle world
	chain.ExpectState(func(s *setup.StateAssertion) {
		s.WorldCount(1)
		s.WorldStatus(universe.StatusIdle)
	})

	// AND the container should have physics and faculties files
	chain.ExpectContainer(func(c *setup.ContainerAssertion) {
		c.IsRunning()
		c.HasFile("/universe/physics.md")
		c.HasFile("/universe/faculties.md")
	})
}

func TestSpawn_NoAgent(t *testing.T) {
	// GIVEN the default configuration
	// WHEN a universe is spawned without an agent
	chain := setup.NewSpawnBuilder(t).
		NoAgent().
		Execute()

	// THEN the state should show one universe with no agent
	chain.ExpectState(func(s *setup.StateAssertion) {
		s.WorldCount(1)
		s.HasNoAgent()
	})
}

func TestSpawn_WithWorkspace(t *testing.T) {
	// GIVEN a test workspace directory
	// WHEN a universe is spawned with that workspace mounted
	chain := setup.NewSpawnBuilder(t).
		WithAgent("test-agent").
		WithWorkspace(filepath.Join(setup.TestdataDir(), "project")).
		Execute()

	// THEN the workspace should be mounted and contain the test project files
	chain.ExpectContainer(func(c *setup.ContainerAssertion) {
		c.HasMount("/workspace")
		c.FileContains("/workspace/README.md", "test project")
	})
}

func TestSpawn_WithAgent(t *testing.T) {
	// GIVEN an agent with a standard Mind structure
	// WHEN a universe is spawned with that agent
	chain := setup.NewSpawnBuilder(t).
		WithAgent("test-agent").
		Execute()

	// THEN the mind should be mounted in the container
	chain.ExpectContainer(func(c *setup.ContainerAssertion) {
		c.HasMount("/mind")
	})

	// AND the Mind should have all standard layers
	chain.ExpectMind(func(m *setup.MindAssertion) {
		m.HasLayer("personas")
		m.HasLayer("skills")
		m.HasLayer("knowledge")
		m.HasLayer("playbooks")
		m.HasLayer("journal")
		m.HasLayer("sessions")
		m.HasFile("personas/default.md")
	})
}

func TestSpawn_CustomPhysics(t *testing.T) {
	// GIVEN a custom physics configuration with specific constants and laws
	// WHEN a universe is spawned with that configuration
	chain := setup.NewSpawnBuilder(t).
		WithConfigYAML(`
physics:
  constants:
    cpu: 2
    memory: 1g
    disk: 4g
    timeout: 60m
  laws:
    max-processes: 64
  elements:
    - "@unix"
`).
		NoAgent().
		Execute()

	// THEN the physics.md should reflect the custom values
	chain.ExpectContainer(func(c *setup.ContainerAssertion) {
		c.FileContains("/universe/physics.md", "2 core(s)")
		c.FileContains("/universe/physics.md", "1g")
		c.FileContains("/universe/physics.md", "60m")
		c.FileContains("/universe/physics.md", "64")
	})
}

func TestSpawn_MockSeesEverything(t *testing.T) {
	// GIVEN a universe with an agent, workspace, and detached execution
	// WHEN the mock agent runs
	chain := setup.NewSpawnBuilder(t).
		WithAgent("test-agent").
		WithWorkspace(filepath.Join(setup.TestdataDir(), "project")).
		Detached().
		Execute()

	// THEN the mock should observe all mounted resources
	chain.ExpectMock(func(m *setup.MockAssertion) {
		m.WasCalled()
		m.SawMind()
		m.SawPhysics()
		m.SawFaculties()
		m.SawWorkspace()
	})
}

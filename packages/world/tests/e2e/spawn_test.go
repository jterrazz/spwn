//go:build e2e

package e2e

import (
	"path/filepath"
	"testing"

	"spwn.sh/packages/world"
	"spwn.sh/packages/world/tests/e2e/setup"
)

func TestSpawn_DefaultConfig(t *testing.T) {
	// Given - the default world configuration
	// When - a world is spawned with an agent
	chain := setup.NewSpawnBuilder(t).
		WithAgent("test-agent").
		Execute()

	// Then - the state should contain one idle world
	chain.ExpectState(func(s *setup.StateAssertion) {
		s.WorldCount(1)
		s.WorldStatus(world.StatusRunning)
	})

	// AND the container should have physics and faculties files
	chain.ExpectContainer(func(c *setup.ContainerAssertion) {
		c.IsRunning()
		c.HasFile("/world/physics.md")
		c.HasFile("/world/faculties.md")
	})
}

func TestSpawn_NoAgent(t *testing.T) {
	// Given - the default configuration
	// When - a world is spawned without an agent
	chain := setup.NewSpawnBuilder(t).
		NoAgent().
		Execute()

	// Then - the state should show one world with no agent
	chain.ExpectState(func(s *setup.StateAssertion) {
		s.WorldCount(1)
		s.HasNoAgent()
	})
}

func TestSpawn_WithWorkspace(t *testing.T) {
	// Given - a test workspace directory
	// When - a world is spawned with that workspace mounted
	chain := setup.NewSpawnBuilder(t).
		WithAgent("test-agent").
		WithWorkspace(filepath.Join(setup.TestdataDir(), "project")).
		Execute()

	// Then - the workspace should be mounted and contain the test project files
	chain.ExpectContainer(func(c *setup.ContainerAssertion) {
		c.HasMount("/work/default")
		c.FileContains("/work/default/README.md", "test project")
	})
}

func TestSpawn_WithAgent(t *testing.T) {
	// Given - an agent with a standard Mind structure
	// When - a world is spawned with that agent
	chain := setup.NewSpawnBuilder(t).
		WithAgent("test-agent").
		Execute()

	// Then - the mind should be mounted in the container
	chain.ExpectContainer(func(c *setup.ContainerAssertion) {
		c.HasMount("/agents")
	})

	// AND the Mind should have all standard layers
	chain.ExpectMind(func(m *setup.MindAssertion) {
		m.HasLayer("core")
		m.HasLayer("skills")
		m.HasLayer("knowledge")
		m.HasLayer("playbooks")
		m.HasLayer("journal")
		m.HasFile("core/profile.md")
	})
}

func TestSpawn_MockSeesEverything(t *testing.T) {
	// Given - a world with an agent, workspace, and detached execution
	// When - the mock agent runs
	chain := setup.NewSpawnBuilder(t).
		WithAgent("test-agent").
		WithWorkspace(filepath.Join(setup.TestdataDir(), "project")).
		Detached().
		Execute()

	// Then - the mock should observe all mounted resources
	chain.ExpectMock(func(m *setup.MockAssertion) {
		m.WasCalled()
		m.SawMind()
		m.SawPhysics()
		m.SawFaculties()
		m.SawWorkspace()
	})
}

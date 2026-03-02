//go:build e2e

package e2e

import (
	"path/filepath"
	"testing"

	"github.com/jterrazz/universe/__tests__/e2e/setup"
	"github.com/jterrazz/universe/internal/config"
)

func TestSpawn_DefaultConfig(t *testing.T) {
	setup.NewSpawnBuilder(t).
		WithAgent("test-agent").
		Execute().
		ExpectState(func(s *setup.StateAssertion) {
			s.UniverseCount(1)
			s.UniverseStatus(config.StatusIdle)
		}).
		ExpectContainer(func(c *setup.ContainerAssertion) {
			c.IsRunning()
			c.HasFile("/universe/physics.md")
			c.HasFile("/universe/faculties.md")
		})
}

func TestSpawn_NoAgent(t *testing.T) {
	setup.NewSpawnBuilder(t).
		NoAgent().
		Execute().
		ExpectState(func(s *setup.StateAssertion) {
			s.UniverseCount(1)
			s.HasNoAgent()
		})
}

func TestSpawn_WithWorkspace(t *testing.T) {
	setup.NewSpawnBuilder(t).
		WithAgent("test-agent").
		WithWorkspace(filepath.Join(setup.TestdataDir(), "project")).
		Execute().
		ExpectContainer(func(c *setup.ContainerAssertion) {
			c.HasMount("/workspace")
			c.FileContains("/workspace/README.md", "test project")
		})
}

func TestSpawn_WithAgent(t *testing.T) {
	setup.NewSpawnBuilder(t).
		WithAgent("test-agent").
		Execute().
		ExpectContainer(func(c *setup.ContainerAssertion) {
			c.HasMount("/mind")
		}).
		ExpectMind(func(m *setup.MindAssertion) {
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
	setup.NewSpawnBuilder(t).
		WithConfigYAML(`
physics:
  constants:
    cpu: 2
    memory: 1g
    disk: 4g
    timeout: 60m
  laws:
    network: none
    max-processes: 64
  elements:
    - "@unix"
`).
		NoAgent().
		Execute().
		ExpectContainer(func(c *setup.ContainerAssertion) {
			c.FileContains("/universe/physics.md", "2 core(s)")
			c.FileContains("/universe/physics.md", "1g")
			c.FileContains("/universe/physics.md", "60m")
			c.FileContains("/universe/physics.md", "64")
		})
}

func TestSpawn_MockSeesEverything(t *testing.T) {
	setup.NewSpawnBuilder(t).
		WithAgent("test-agent").
		WithWorkspace(filepath.Join(setup.TestdataDir(), "project")).
		Detached().
		Execute().
		ExpectMock(func(m *setup.MockAssertion) {
			m.WasCalled()
			m.SawMind()
			m.SawPhysics()
			m.SawFaculties()
			m.SawWorkspace()
		})
}

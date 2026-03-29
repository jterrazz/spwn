//go:build e2e

package e2e

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jterrazz/spwn/domains/universe/tests/e2e/setup"
	"github.com/jterrazz/spwn/domains/universe"
	agentDomain "github.com/jterrazz/spwn/domains/agent"
	"github.com/jterrazz/spwn/shared/config"
)

func TestDefaults_SpawnWorksWithoutInit(t *testing.T) {
	// Fresh test context — no init, no agent init, nothing.
	// Simulate what ensureDefaults() does in PersistentPreRunE.
	universe.CreateDefaultConfig()
	agentDomain.InitMind("default")

	setup.NewSpawnBuilder(t).
		NoAgent().
		Execute().
		ExpectState(func(s *setup.StateAssertion) {
			s.UniverseCount(1)
			s.UniverseStatus(universe.StatusIdle)
		}).
		ExpectContainer(func(c *setup.ContainerAssertion) {
			c.IsRunning()
			c.HasFile("/universe/physics.md")
			c.HasFile("/universe/faculties.md")
		})
}

func TestDefaults_SpawnWithDefaultAgent(t *testing.T) {
	universe.CreateDefaultConfig()
	agentDomain.InitMind("default")

	setup.NewSpawnBuilder(t).
		WithAgent("default").
		Execute().
		ExpectState(func(s *setup.StateAssertion) {
			s.UniverseCount(1)
			s.HasAgent("default")
		}).
		ExpectContainer(func(c *setup.ContainerAssertion) {
			c.IsRunning()
			c.HasMount("/mind")
		}).
		ExpectMind(func(m *setup.MindAssertion) {
			m.HasLayer("personas")
			m.HasFile("personas/default.md")
		})
}

func TestDefaults_DefaultConfigIsLoadable(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("SPWN_HOME", tmpDir)

	if err := universe.CreateDefaultConfig(); err != nil {
		t.Fatalf("CreateDefault failed: %v", err)
	}

	m, err := universe.LoadManifest("default")
	if err != nil {
		t.Fatalf("Load default failed: %v", err)
	}

	if m.Physics.Constants.CPU != config.DefaultCPU {
		t.Errorf("Expected CPU %d, got %d", config.DefaultCPU, m.Physics.Constants.CPU)
	}
	if m.Physics.Constants.Memory != config.DefaultMemory {
		t.Errorf("Expected memory %q, got %q", config.DefaultMemory, m.Physics.Constants.Memory)
	}
	if m.Physics.Laws.Network != config.DefaultNetwork {
		t.Errorf("Expected network %q, got %q", config.DefaultNetwork, m.Physics.Laws.Network)
	}
}

func TestDefaults_DefaultAgentIsValid(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("SPWN_HOME", tmpDir)

	os.MkdirAll(filepath.Join(tmpDir, "agents"), 0755)

	_, err := agentDomain.InitMind("default")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	if err := agentDomain.ValidateMind("default"); err != nil {
		t.Fatalf("Validate after Init failed: %v", err)
	}
}

func TestDefaults_IdempotentRerun(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("SPWN_HOME", tmpDir)

	os.MkdirAll(filepath.Join(tmpDir, "agents"), 0755)

	// First run
	if err := universe.CreateDefaultConfig(); err != nil {
		t.Fatalf("First CreateDefault failed: %v", err)
	}
	if _, err := agentDomain.InitMind("default"); err != nil {
		t.Fatalf("First Init failed: %v", err)
	}

	// Second run — should not corrupt files (errors are expected, just ignore them)
	universe.CreateDefaultConfig()
	agentDomain.InitMind("default")

	// Everything should still be valid
	if _, err := universe.LoadManifest("default"); err != nil {
		t.Fatalf("Load after idempotent create failed: %v", err)
	}
	if err := agentDomain.ValidateMind("default"); err != nil {
		t.Fatalf("Validate after idempotent init failed: %v", err)
	}
}

//go:build e2e

package e2e

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jterrazz/universe/__tests__/e2e/setup"
	"github.com/jterrazz/universe/internal/config"
	"github.com/jterrazz/universe/internal/manifest"
	"github.com/jterrazz/universe/internal/mind"
)

func TestDefaults_SpawnWorksWithoutInit(t *testing.T) {
	// Fresh test context — no init, no agent init, nothing.
	// Simulate what ensureDefaults() does in PersistentPreRunE.
	manifest.CreateDefault()
	mind.Init("default")

	setup.NewSpawnBuilder(t).
		NoAgent().
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

func TestDefaults_SpawnWithDefaultAgent(t *testing.T) {
	manifest.CreateDefault()
	mind.Init("default")

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
	t.Setenv("UNIVERSE_HOME", tmpDir)

	if err := manifest.CreateDefault(); err != nil {
		t.Fatalf("CreateDefault failed: %v", err)
	}

	m, err := manifest.Load("default")
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
	t.Setenv("UNIVERSE_HOME", tmpDir)

	os.MkdirAll(filepath.Join(tmpDir, "agents"), 0755)

	_, err := mind.Init("default")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	if err := mind.Validate("default"); err != nil {
		t.Fatalf("Validate after Init failed: %v", err)
	}
}

func TestDefaults_IdempotentRerun(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("UNIVERSE_HOME", tmpDir)

	os.MkdirAll(filepath.Join(tmpDir, "agents"), 0755)

	// First run
	if err := manifest.CreateDefault(); err != nil {
		t.Fatalf("First CreateDefault failed: %v", err)
	}
	if _, err := mind.Init("default"); err != nil {
		t.Fatalf("First Init failed: %v", err)
	}

	// Second run — should not corrupt files (errors are expected, just ignore them)
	manifest.CreateDefault()
	mind.Init("default")

	// Everything should still be valid
	if _, err := manifest.Load("default"); err != nil {
		t.Fatalf("Load after idempotent create failed: %v", err)
	}
	if err := mind.Validate("default"); err != nil {
		t.Fatalf("Validate after idempotent init failed: %v", err)
	}
}

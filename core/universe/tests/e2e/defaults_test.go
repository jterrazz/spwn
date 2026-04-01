//go:build e2e

package e2e

import (
	"os"
	"path/filepath"
	"testing"

	agentDomain "spwn.sh/core/agent"
	"spwn.sh/core/foundation"
	"spwn.sh/core/universe"
	"spwn.sh/core/universe/tests/e2e/setup"
)

func TestDefaults_SpawnWorksWithoutInit(t *testing.T) {
	// GIVEN default config and default agent are created (simulating ensureDefaults)
	universe.CreateDefaultConfig()
	agentDomain.InitMind("default")

	// WHEN a universe is spawned with no agent
	chain := setup.NewSpawnBuilder(t).
		NoAgent().
		Execute()

	// THEN the state should show one idle world with physics and faculties
	chain.ExpectState(func(s *setup.StateAssertion) {
		s.WorldCount(1)
		s.WorldStatus(universe.StatusIdle)
	})
	chain.ExpectContainer(func(c *setup.ContainerAssertion) {
		c.IsRunning()
		c.HasFile("/universe/physics.md")
		c.HasFile("/universe/faculties.md")
	})
}

func TestDefaults_SpawnWithDefaultAgent(t *testing.T) {
	// GIVEN default config and default agent
	universe.CreateDefaultConfig()
	agentDomain.InitMind("default")

	// WHEN a universe is spawned with the default agent
	chain := setup.NewSpawnBuilder(t).
		WithAgent("default").
		Execute()

	// THEN the state should track the agent
	chain.ExpectState(func(s *setup.StateAssertion) {
		s.WorldCount(1)
		s.HasAgent("default")
	})

	// AND the Mind should be mounted with standard layers
	chain.ExpectContainer(func(c *setup.ContainerAssertion) {
		c.IsRunning()
		c.HasMount("/mind")
	})
	chain.ExpectMind(func(m *setup.MindAssertion) {
		m.HasLayer("identity")
		m.HasFile("identity/default.md")
	})
}

func TestDefaults_DefaultConfigIsLoadable(t *testing.T) {
	// GIVEN a fresh SPWN_HOME with the default config created
	tmpDir := t.TempDir()
	t.Setenv("SPWN_HOME", tmpDir)

	if err := universe.CreateDefaultConfig(); err != nil {
		t.Fatalf("CreateDefault failed: %v", err)
	}

	// WHEN the manifest is loaded
	m, err := universe.LoadManifest("default")
	if err != nil {
		t.Fatalf("Load default failed: %v", err)
	}

	// THEN it should have the expected foundation defaults
	if m.Physics.Constants.CPU != foundation.DefaultCPU {
		t.Errorf("Expected CPU %d, got %d", foundation.DefaultCPU, m.Physics.Constants.CPU)
	}
	if m.Physics.Constants.Memory != foundation.DefaultMemory {
		t.Errorf("Expected memory %q, got %q", foundation.DefaultMemory, m.Physics.Constants.Memory)
	}
}

func TestDefaults_DefaultAgentIsValid(t *testing.T) {
	// GIVEN a fresh SPWN_HOME with the default agent initialized
	tmpDir := t.TempDir()
	t.Setenv("SPWN_HOME", tmpDir)
	os.MkdirAll(filepath.Join(tmpDir, "agents"), 0755)

	_, err := agentDomain.InitMind("default")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// WHEN validating the agent
	err = agentDomain.ValidateMind("default")

	// THEN it should pass validation
	if err != nil {
		t.Fatalf("Validate after Init failed: %v", err)
	}
}

func TestDefaults_IdempotentRerun(t *testing.T) {
	// GIVEN default config and agent are created once
	tmpDir := t.TempDir()
	t.Setenv("SPWN_HOME", tmpDir)
	os.MkdirAll(filepath.Join(tmpDir, "agents"), 0755)

	if err := universe.CreateDefaultConfig(); err != nil {
		t.Fatalf("First CreateDefault failed: %v", err)
	}
	if _, err := agentDomain.InitMind("default"); err != nil {
		t.Fatalf("First Init failed: %v", err)
	}

	// WHEN creating them again (idempotent re-run)
	universe.CreateDefaultConfig()
	agentDomain.InitMind("default")

	// THEN the config and agent should still be valid
	if _, err := universe.LoadManifest("default"); err != nil {
		t.Fatalf("Load after idempotent create failed: %v", err)
	}
	if err := agentDomain.ValidateMind("default"); err != nil {
		t.Fatalf("Validate after idempotent init failed: %v", err)
	}
}

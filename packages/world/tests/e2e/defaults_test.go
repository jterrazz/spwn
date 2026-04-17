//go:build e2e

package e2e

import (
	"os"
	"path/filepath"
	"testing"

	"spwn.sh/packages/agent"
	"spwn.sh/packages/world"
	"spwn.sh/packages/world/tests/e2e/setup"
)

func TestDefaults_SpawnWorksWithoutInit(t *testing.T) {
	// Given - default config and default agent are created (simulating ensureDefaults)
	world.CreateDefaultConfig()
	agent.InitMind("default")

	// When - a world is spawned with no agent
	chain := setup.NewSpawnBuilder(t).
		NoAgent().
		Execute()

	// Then - the state should show one live world with physics and faculties
	chain.ExpectState(func(s *setup.StateAssertion) {
		s.WorldCount(1)
		s.WorldStatus(world.StatusRunning)
	})
	chain.ExpectContainer(func(c *setup.ContainerAssertion) {
		c.IsRunning()
		c.HasFile("/world/physics.md")
		c.HasFile("/world/faculties.md")
	})
}

func TestDefaults_SpawnWithDefaultAgent(t *testing.T) {
	// Given - default config and default agent
	world.CreateDefaultConfig()
	agent.InitMind("default")

	// When - a world is spawned with the default agent
	chain := setup.NewSpawnBuilder(t).
		WithAgent("default").
		Execute()

	// Then - the state should track the agent
	chain.ExpectState(func(s *setup.StateAssertion) {
		s.WorldCount(1)
		s.HasAgent("default")
	})

	// AND the Mind should be mounted with standard layers
	chain.ExpectContainer(func(c *setup.ContainerAssertion) {
		c.IsRunning()
		c.HasMount("/agents")
	})
	chain.ExpectMind(func(m *setup.MindAssertion) {
		m.HasLayer("identity")
		m.HasFile("identity/profile.md")
	})
}

func TestDefaults_DefaultConfigIsLoadable(t *testing.T) {
	// Given - a fresh SPWN_HOME with the default config created
	tmpDir := t.TempDir()
	t.Setenv("SPWN_HOME", tmpDir)

	if err := world.CreateDefaultConfig(); err != nil {
		t.Fatalf("CreateDefault failed: %v", err)
	}

	// When - the manifest is loaded, it should parse cleanly
	if _, err := world.LoadManifest("default"); err != nil {
		t.Fatalf("Load default failed: %v", err)
	}
}

func TestDefaults_DefaultAgentIsValid(t *testing.T) {
	// Given - a fresh SPWN_HOME with the default agent initialized
	tmpDir := t.TempDir()
	t.Setenv("SPWN_HOME", tmpDir)
	os.MkdirAll(filepath.Join(tmpDir, "agents"), 0755)

	_, err := agent.InitMind("default")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// When - validating the agent
	err = agent.ValidateMind("default")

	// Then - it should pass validation
	if err != nil {
		t.Fatalf("Validate after Init failed: %v", err)
	}
}

func TestDefaults_IdempotentRerun(t *testing.T) {
	// Given - default config and agent are created once
	tmpDir := t.TempDir()
	t.Setenv("SPWN_HOME", tmpDir)
	os.MkdirAll(filepath.Join(tmpDir, "agents"), 0755)

	if err := world.CreateDefaultConfig(); err != nil {
		t.Fatalf("First CreateDefault failed: %v", err)
	}
	if _, err := agent.InitMind("default"); err != nil {
		t.Fatalf("First Init failed: %v", err)
	}

	// When - creating them again (idempotent re-run)
	world.CreateDefaultConfig()
	agent.InitMind("default")

	// Then - the config and agent should still be valid
	if _, err := world.LoadManifest("default"); err != nil {
		t.Fatalf("Load after idempotent create failed: %v", err)
	}
	if err := agent.ValidateMind("default"); err != nil {
		t.Fatalf("Validate after idempotent init failed: %v", err)
	}
}

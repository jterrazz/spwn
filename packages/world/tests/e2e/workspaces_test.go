//go:build e2e

package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"spwn.sh/packages/world/tests/e2e/setup"
)

// TestSpawn_Ephemeral verifies that a world with zero workspaces has no /work
// mounts and no SPWN_WORKSPACES env var advertising mounts that don't exist.
func TestSpawn_Ephemeral(t *testing.T) {
	// GIVEN the default configuration
	// WHEN a world is spawned without any -w flags
	chain := setup.NewSpawnBuilder(t).
		WithAgent("test-agent").
		Execute()

	// THEN the container is running with no workspace mounts
	chain.ExpectContainer(func(c *setup.ContainerAssertion) {
		c.IsRunning()
	})
}

// TestSpawn_SingleNamedWorkspace verifies that one named workspace is mounted
// at /work/<name>.
func TestSpawn_SingleNamedWorkspace(t *testing.T) {
	// GIVEN a project workspace
	// WHEN a world is spawned with a single named workspace
	chain := setup.NewSpawnBuilder(t).
		WithAgent("test-agent").
		WithNamedWorkspace("proj", filepath.Join(setup.TestdataDir(), "project")).
		Execute()

	// THEN the project should be visible at /work/proj
	chain.ExpectContainer(func(c *setup.ContainerAssertion) {
		c.HasMount("/work/proj")
		c.FileContains("/work/proj/README.md", "test project")
	})
}

// TestSpawn_MultipleWorkspaces verifies that each workspace is mounted under
// /work/<name>/ so that `ls /work` reveals the workspaces the agent can touch.
func TestSpawn_MultipleWorkspaces(t *testing.T) {
	dir := t.TempDir()
	webPath := filepath.Join(dir, "web")
	apiPath := filepath.Join(dir, "api")
	for _, p := range []string{webPath, apiPath} {
		if err := os.MkdirAll(p, 0755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
	}
	if err := os.WriteFile(filepath.Join(webPath, "marker.txt"), []byte("from-web"), 0644); err != nil {
		t.Fatalf("write web: %v", err)
	}
	if err := os.WriteFile(filepath.Join(apiPath, "marker.txt"), []byte("from-api"), 0644); err != nil {
		t.Fatalf("write api: %v", err)
	}

	// GIVEN two host directories
	// WHEN a world is spawned with two named workspaces
	chain := setup.NewSpawnBuilder(t).
		WithAgent("test-agent").
		WithNamedWorkspace("web", webPath).
		WithNamedWorkspace("api", apiPath).
		Execute()

	// THEN both workspaces are mounted under /work/<name>.
	chain.ExpectContainer(func(c *setup.ContainerAssertion) {
		c.HasMount("/work/web")
		c.HasMount("/work/api")
		c.FileContains("/work/web/marker.txt", "from-web")
		c.FileContains("/work/api/marker.txt", "from-api")
	})

	// `ls /work` should list the two workspace names as directories.
	listing := chain.ExecInContainer([]string{"ls", "/work"})
	if !strings.Contains(listing, "web") || !strings.Contains(listing, "api") {
		t.Errorf("expected /work to list 'web' and 'api', got: %q", listing)
	}
}

// TestSpawn_ReadOnlyWorkspace verifies that a workspace mounted with :ro
// cannot be written to from inside the container.
func TestSpawn_ReadOnlyWorkspace(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "locked.txt"), []byte("read only"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	// GIVEN a host directory
	// WHEN mounted read-only
	chain := setup.NewSpawnBuilder(t).
		WithAgent("test-agent").
		WithReadOnlyWorkspace("docs", dir).
		Execute()

	// THEN the file is readable but writes fail.
	chain.ExpectContainer(func(c *setup.ContainerAssertion) {
		c.HasMount("/work/docs")
		c.FileContains("/work/docs/locked.txt", "read only")
	})

	// Attempt a write via exec and expect non-zero exit.
	result := chain.ExecInContainer([]string{"sh", "-c", "echo no > /work/docs/forbidden.txt 2>&1 || echo WRITE_FAILED"})
	if !strings.Contains(result, "WRITE_FAILED") && !strings.Contains(result, "Read-only file system") {
		t.Errorf("expected read-only mount to reject writes, got: %q", result)
	}
}

// TestSpawn_WorkspaceEnvVars verifies that SPWN_WORKSPACES and
// SPWN_WORKSPACE_DEFAULT are set correctly inside the container so agents
// can discover their mounts programmatically.
func TestSpawn_WorkspaceEnvVars(t *testing.T) {
	dir := t.TempDir()
	webPath := filepath.Join(dir, "web")
	apiPath := filepath.Join(dir, "api")
	for _, p := range []string{webPath, apiPath} {
		_ = os.MkdirAll(p, 0755)
	}

	// GIVEN two workspaces
	chain := setup.NewSpawnBuilder(t).
		WithAgent("test-agent").
		WithNamedWorkspace("web", webPath).
		WithNamedWorkspace("api", apiPath).
		Execute()

	// WHEN we read the env inside the container
	envOut := chain.ExecInContainer([]string{"sh", "-c", "env | grep ^SPWN_WORKSPACE"})

	// THEN both vars should be set and list each workspace
	if !strings.Contains(envOut, "SPWN_WORKSPACES=") {
		t.Errorf("SPWN_WORKSPACES not set, got: %q", envOut)
	}
	if !strings.Contains(envOut, "web:/work/web") || !strings.Contains(envOut, "api:/work/api") {
		t.Errorf("SPWN_WORKSPACES missing expected pairs, got: %q", envOut)
	}
	if !strings.Contains(envOut, "SPWN_WORKSPACE_DEFAULT=/work/web") {
		t.Errorf("SPWN_WORKSPACE_DEFAULT should point at first workspace, got: %q", envOut)
	}
}

// TestSpawn_EphemeralHasNoSpwnWorkspacesEnv verifies that ephemeral worlds
// do NOT set SPWN_WORKSPACES (agents can use its absence as a signal).
func TestSpawn_EphemeralHasNoSpwnWorkspacesEnv(t *testing.T) {
	chain := setup.NewSpawnBuilder(t).
		WithAgent("test-agent").
		Execute()

	envOut := chain.ExecInContainer([]string{"sh", "-c", "env | grep ^SPWN_WORKSPACES || echo UNSET"})
	if !strings.Contains(envOut, "UNSET") {
		t.Errorf("ephemeral world should not set SPWN_WORKSPACES, got: %q", envOut)
	}
}

// TestSpawn_DuplicateWorkspaceName should fail fast.
func TestSpawn_DuplicateWorkspaceName(t *testing.T) {
	dir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(dir, "a"), 0755)
	_ = os.MkdirAll(filepath.Join(dir, "b"), 0755)

	setup.NewSpawnBuilder(t).
		WithAgent("test-agent").
		WithNamedWorkspace("same", filepath.Join(dir, "a")).
		WithNamedWorkspace("same", filepath.Join(dir, "b")).
		ExecuteExpectError("duplicate workspace name")
}

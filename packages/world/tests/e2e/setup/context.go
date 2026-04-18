//go:build e2e

package setup

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"spwn.sh/packages/agent"
	"spwn.sh/packages/architect"
	"spwn.sh/packages/world"
	"spwn.sh/packages/world/labels"
	"spwn.sh/packages/world/runtimestate"
)

const TestImage = "spwn-test:latest"

// TestContext provides isolated E2E test infrastructure.
type TestContext struct {
	T       *testing.T
	BaseDir string
	Image   string
	Backend world.Backend
	State   *world.Store
	Arc     *architect.Architect
	Spawned []string // world IDs to clean up
}

// NewTestContext creates an isolated test environment with temp dirs and real Docker.
func NewTestContext(t *testing.T) *TestContext {
	t.Helper()

	baseDir := t.TempDir()
	t.Setenv("SPWN_HOME", baseDir)

	// Stamp every container spawned by this test with a unique label
	// so List()/Get() only see THIS test's containers, not leftovers
	// from prior runs or concurrently-running tests. The architect
	// reads labels.TestRunEnv inside Spawn() to apply the label.
	t.Setenv(labels.TestRunEnv, fmt.Sprintf("e2e-%s-%d", t.Name(), time.Now().UnixNano()))

	// Create required subdirectories
	os.MkdirAll(filepath.Join(baseDir, "worlds"), 0755)
	os.MkdirAll(filepath.Join(baseDir, "agents"), 0755)

	docker, err := world.NewDocker()
	if err != nil {
		t.Fatalf("Docker must be running for E2E tests: %v", err)
	}

	store, err := runtimestate.NewStoreWith(docker, filepath.Join(baseDir, "world-states"))
	if err != nil {
		t.Fatalf("Failed to create runtimestate store: %v", err)
	}

	ctx := &TestContext{
		T:       t,
		BaseDir: baseDir,
		Image:   TestImage,
		Backend: docker,
		State:   store,
		Arc:     architect.New(docker, store),
	}

	// Verify test image exists
	exists, err := docker.ImageExists(context.Background(), TestImage)
	if err != nil {
		t.Fatalf("Failed to check test image: %v", err)
	}
	if !exists {
		t.Fatalf("Test image %s not found. Run 'make build-test-image' first.", TestImage)
	}

	t.Cleanup(func() {
		// Bound each Destroy with a timeout. Without this, a single
		// hung Docker API call (seen on OrbStack under load after
		// ~15+ rapid spawn/destroy cycles) would wedge the whole
		// test process and every subsequent test would see "e2e
		// [setup failed]" with an unrelated goroutine dump. 30s is
		// generous for a graceful container teardown and bounds the
		// worst case at len(Spawned) * 30s.
		for _, id := range ctx.Spawned {
			destroyCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			_, _ = ctx.Arc.Destroy(destroyCtx, id)
			cancel()
		}
	})

	return ctx
}

// TrackWorld adds a world ID for cleanup.
func (tc *TestContext) TrackWorld(id string) {
	tc.Spawned = append(tc.Spawned, id)
}

// LoadState lists every world the daemon currently knows about
// (sourced from Docker container labels + hydrated from runtimestate).
func (tc *TestContext) LoadState() []world.World {
	tc.T.Helper()
	worlds, err := tc.State.List()
	if err != nil {
		tc.T.Fatalf("Failed to load state: %v", err)
	}
	return worlds
}

// InitAgent creates an agent Mind in the temp directory.
func (tc *TestContext) InitAgent(name string) {
	tc.T.Helper()
	_, err := agent.InitMind(name)
	if err != nil {
		tc.T.Fatalf("Failed to init agent %q: %v", name, err)
	}
}

// MockOutput represents the JSON recorded by the mock Claude binary.
type MockOutput struct {
	MindExists       bool   `json:"mind_exists"`
	MindPersonas     bool   `json:"mind_personas"`
	PhysicsExists    bool   `json:"physics_exists"`
	FacultiesExists  bool   `json:"faculties_exists"`
	WorkspaceExists  bool   `json:"workspace_exists"`
	PhysicsContent   string `json:"physics_content"`
	FacultiesContent string `json:"faculties_content"`
	SessionID        string `json:"session_id"`
	Resume           bool   `json:"resume"`
	PID              int    `json:"pid"`
	ExitCode         int    `json:"exit_code"`
}

// ReadMockOutput reads and parses the mock claude output from inside a container.
func (tc *TestContext) ReadMockOutput(containerID string) *MockOutput {
	tc.T.Helper()
	output, err := tc.Backend.ExecOutput(context.Background(), containerID, []string{"cat", "/tmp/claude-mock.json"})
	if err != nil {
		tc.T.Fatalf("Failed to read mock output: %v", err)
	}

	var mock MockOutput
	if err := json.Unmarshal([]byte(output), &mock); err != nil {
		tc.T.Fatalf("Failed to parse mock output: %v\nRaw: %s", err, output)
	}
	return &mock
}

// ExecInContainer runs a command inside a container and returns stdout.
func (tc *TestContext) ExecInContainer(containerID string, cmd []string) string {
	tc.T.Helper()
	output, err := tc.Backend.ExecOutput(context.Background(), containerID, cmd)
	if err != nil {
		tc.T.Fatalf("Failed to exec in container: %v", err)
	}
	return output
}

// FileExistsInContainer checks if a file exists inside a container.
func (tc *TestContext) FileExistsInContainer(containerID, path string) bool {
	tc.T.Helper()
	_, err := tc.Backend.ExecOutput(context.Background(), containerID, []string{"test", "-f", path})
	return err == nil
}

// DirExistsInContainer checks if a directory exists inside a container.
func (tc *TestContext) DirExistsInContainer(containerID, path string) bool {
	tc.T.Helper()
	_, err := tc.Backend.ExecOutput(context.Background(), containerID, []string{"test", "-d", path})
	return err == nil
}

// ReadFileInContainer reads a file from inside a container.
func (tc *TestContext) ReadFileInContainer(containerID, path string) string {
	tc.T.Helper()
	return tc.ExecInContainer(containerID, []string{"cat", path})
}

// WaitFor polls a condition until it returns true or times out.
// Replaces time.Sleep with proper polling.
func WaitFor(t *testing.T, timeout time.Duration, interval time.Duration, desc string, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(interval)
	}
	t.Fatalf("timed out waiting for: %s", desc)
}

// TryReadMockOutput attempts to read the mock output without fataling.
// Returns nil if the file doesn't exist or can't be parsed yet.
func (tc *TestContext) TryReadMockOutput(containerID string) *MockOutput {
	output, err := tc.Backend.ExecOutput(context.Background(), containerID, []string{"cat", "/tmp/claude-mock.json"})
	if err != nil {
		return nil
	}

	var mock MockOutput
	if err := json.Unmarshal([]byte(output), &mock); err != nil {
		return nil
	}
	return &mock
}

// TestdataDir returns the absolute path to tests/fixtures/testdata/ in the repo.
func TestdataDir() string {
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.work")); err == nil {
			return filepath.Join(dir, "tests", "fixtures", "testdata")
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return filepath.Join("tests", "fixtures", "testdata")
}

//go:build e2e

package setup

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/jterrazz/universe/internal/architect"
	"github.com/jterrazz/universe/internal/backend"
	"github.com/jterrazz/universe/internal/config"
	"github.com/jterrazz/universe/internal/mind"
	"github.com/jterrazz/universe/internal/state"
)

const TestImage = "universe-test:latest"

// TestContext provides isolated E2E test infrastructure.
type TestContext struct {
	T       *testing.T
	BaseDir string
	Image   string
	Backend backend.Backend
	State   *state.Store
	Arc     *architect.Architect
	Spawned []string // universe IDs to clean up
}

// NewTestContext creates an isolated test environment with temp dirs and real Docker.
func NewTestContext(t *testing.T) *TestContext {
	t.Helper()

	baseDir := t.TempDir()
	t.Setenv("UNIVERSE_HOME", baseDir)

	// Create required subdirectories
	os.MkdirAll(filepath.Join(baseDir, "universes"), 0755)
	os.MkdirAll(filepath.Join(baseDir, "agents"), 0755)

	docker, err := backend.NewDocker()
	if err != nil {
		t.Fatalf("Docker must be running for E2E tests: %v", err)
	}

	store, err := state.NewStoreAt(filepath.Join(baseDir, "state.json"))
	if err != nil {
		t.Fatalf("Failed to create state store: %v", err)
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
		bg := context.Background()
		for _, id := range ctx.Spawned {
			ctx.Arc.Destroy(bg, id)
		}
	})

	return ctx
}

// TrackUniverse adds a universe ID for cleanup.
func (tc *TestContext) TrackUniverse(id string) {
	tc.Spawned = append(tc.Spawned, id)
}

// LoadState reads the state.json file and returns the list of universes.
func (tc *TestContext) LoadState() []config.Universe {
	tc.T.Helper()
	universes, err := tc.State.List()
	if err != nil {
		tc.T.Fatalf("Failed to load state: %v", err)
	}
	return universes
}

// InitAgent creates an agent Mind in the temp directory.
func (tc *TestContext) InitAgent(name string) {
	tc.T.Helper()
	_, err := mind.Init(name)
	if err != nil {
		tc.T.Fatalf("Failed to init agent %q: %v", name, err)
	}
}

// MockOutput represents the JSON recorded by the mock Claude binary.
type MockOutput struct {
	MindExists      bool   `json:"mind_exists"`
	MindPersonas    bool   `json:"mind_personas"`
	PhysicsExists   bool   `json:"physics_exists"`
	FacultiesExists bool   `json:"faculties_exists"`
	WorkspaceExists bool   `json:"workspace_exists"`
	PhysicsContent  string `json:"physics_content"`
	FacultiesContent string `json:"faculties_content"`
	PID             int    `json:"pid"`
	ExitCode        int    `json:"exit_code"`
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

// TestdataDir returns the absolute path to __tests__/testdata/ in the repo.
func TestdataDir() string {
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return filepath.Join(dir, "__tests__", "testdata")
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return filepath.Join("__tests__", "testdata")
}

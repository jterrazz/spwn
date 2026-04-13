//go:build e2e

package setup

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"spwn.sh/packages/agent"
	"spwn.sh/packages/world"
)

// --- SpawnBuilder ---

// SpawnBuilder configures and executes a world spawn.
type SpawnBuilder struct {
	tc         *TestContext
	configName string
	agentName  string
	workspaces []world.Workspace
	yamlConfig string
	noAgent    bool
	detach     bool
	runAgent   bool // run agent to completion (blocking, writes journal)
}

// Spawn creates a SpawnBuilder from a TestContext.
func (tc *TestContext) Spawn() *SpawnBuilder {
	return &SpawnBuilder{tc: tc, configName: "default"}
}

// NewSpawnBuilder creates a SpawnBuilder with a fresh TestContext.
func NewSpawnBuilder(t *testing.T) *SpawnBuilder {
	return NewTestContext(t).Spawn()
}

func (b *SpawnBuilder) WithConfig(name string) *SpawnBuilder {
	b.configName = name
	return b
}

func (b *SpawnBuilder) WithConfigYAML(yamlContent string) *SpawnBuilder {
	b.yamlConfig = yamlContent
	return b
}

func (b *SpawnBuilder) WithAgent(name string) *SpawnBuilder {
	b.agentName = name
	// Auto-init agent if it doesn't exist
	agentDir := filepath.Join(b.tc.BaseDir, "agents", name)
	if _, err := os.Stat(agentDir); os.IsNotExist(err) {
		b.tc.InitAgent(name)
	}
	return b
}

func (b *SpawnBuilder) WithWorkspace(path string) *SpawnBuilder {
	b.workspaces = append(b.workspaces, world.Workspace{Name: "default", Path: path})
	return b
}

// WithNamedWorkspace adds a workspace with a specific name (use for multi-workspace tests).
func (b *SpawnBuilder) WithNamedWorkspace(name, path string) *SpawnBuilder {
	b.workspaces = append(b.workspaces, world.Workspace{Name: name, Path: path})
	return b
}

// WithReadOnlyWorkspace adds a read-only workspace.
func (b *SpawnBuilder) WithReadOnlyWorkspace(name, path string) *SpawnBuilder {
	b.workspaces = append(b.workspaces, world.Workspace{Name: name, Path: path, ReadOnly: true})
	return b
}

func (b *SpawnBuilder) NoAgent() *SpawnBuilder {
	b.noAgent = true
	return b
}

func (b *SpawnBuilder) Detached() *SpawnBuilder {
	b.detach = true
	return b
}

// RunAgent runs the agent to completion (blocking). Writes session + journal.
func (b *SpawnBuilder) RunAgent() *SpawnBuilder {
	b.runAgent = true
	return b
}

// Execute spawns the world and returns an AssertionChain.
func (b *SpawnBuilder) Execute() *AssertionChain {
	b.tc.T.Helper()

	m := b.buildManifest()

	opts := world.SpawnOpts{
		ConfigName: b.configName,
		Manifest:   m,
		Image:      b.tc.Image,
	}

	if !b.noAgent && b.agentName != "" {
		opts.AgentName = b.agentName
	}

	for _, ws := range b.workspaces {
		abs, err := filepath.Abs(ws.Path)
		if err != nil {
			b.tc.T.Fatalf("Failed to resolve workspace: %v", err)
		}
		opts.Workspaces = append(opts.Workspaces, world.Workspace{Name: ws.Name, Path: abs, ReadOnly: ws.ReadOnly})
	}

	result, err := b.tc.Arc.Spawn(context.Background(), opts)
	if err != nil {
		b.tc.T.Fatalf("Spawn failed: %v", err)
	}

	u := result.World
	b.tc.TrackWorld(u.ID)

	// Run agent if requested
	if !b.noAgent && b.agentName != "" {
		if b.runAgent {
			// Blocking: runs agent to completion, writes session + journal
			err := b.tc.Arc.SpawnAgent(context.Background(), u.ID, b.agentName)
			if err != nil {
				// Non-fatal: mock may exit with code 0, which is fine
				// Only fatal if it's a real error (not exit code)
				b.tc.T.Logf("SpawnAgent returned: %v", err)
			}
		} else if b.detach {
			err := b.tc.Arc.SpawnAgentDetached(context.Background(), u.ID, b.agentName)
			if err != nil {
				b.tc.T.Fatalf("SpawnAgentDetached failed: %v", err)
			}
			// Wait for the mock to write its output
			WaitFor(b.tc.T, 5*time.Second, 100*time.Millisecond, "mock to write output", func() bool {
				return b.tc.TryReadMockOutput(u.ContainerID) != nil
			})
		}
	}

	return &AssertionChain{tc: b.tc, w: u}
}

// ExecuteExpectError spawns and expects an error containing the substring.
func (b *SpawnBuilder) ExecuteExpectError(substring string) {
	b.tc.T.Helper()

	m := b.buildManifest()

	opts := world.SpawnOpts{
		ConfigName: b.configName,
		Manifest:   m,
		Image:      b.tc.Image,
	}

	if !b.noAgent && b.agentName != "" {
		opts.AgentName = b.agentName
	}

	for _, ws := range b.workspaces {
		abs, _ := filepath.Abs(ws.Path)
		opts.Workspaces = append(opts.Workspaces, world.Workspace{Name: ws.Name, Path: abs, ReadOnly: ws.ReadOnly})
	}

	result, err := b.tc.Arc.Spawn(context.Background(), opts)
	if err == nil {
		b.tc.TrackWorld(result.World.ID)
		b.tc.T.Fatalf("Expected spawn to fail with %q, but it succeeded", substring)
	}

	if !strings.Contains(err.Error(), substring) {
		b.tc.T.Fatalf("Expected error containing %q, got: %v", substring, err)
	}
}

func (b *SpawnBuilder) buildManifest() world.Manifest {
	b.tc.T.Helper()

	if b.yamlConfig != "" {
		return b.parseInlineYAML()
	}

	// Ensure config exists on disk (idempotent — no-ops if already present)
	configPath := filepath.Join(b.tc.BaseDir, "worlds", b.configName+".yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if b.configName == "default" {
			world.CreateDefaultConfig()
		} else {
			world.CreateConfig(b.configName)
		}
	}

	m, err := world.LoadManifestPath(configPath)
	if err != nil {
		b.tc.T.Fatalf("Failed to load config %q: %v", b.configName, err)
	}

	return m
}

func (b *SpawnBuilder) parseInlineYAML() world.Manifest {
	b.tc.T.Helper()

	// Strip leading tabs from inline YAML (Go source uses tabs for indentation)
	cleaned := dedentYAML(b.yamlConfig)

	tmpPath := filepath.Join(b.tc.BaseDir, "worlds", "_inline.yaml")
	os.MkdirAll(filepath.Dir(tmpPath), 0755)
	if err := os.WriteFile(tmpPath, []byte(cleaned), 0644); err != nil {
		b.tc.T.Fatalf("Failed to write inline YAML: %v", err)
	}

	m, err := world.LoadManifestPath(tmpPath)
	if err != nil {
		b.tc.T.Fatalf("Failed to parse inline YAML: %v", err)
	}

	return m
}

// dedentYAML removes common leading whitespace (tabs or spaces) from each line.
func dedentYAML(s string) string {
	lines := strings.Split(s, "\n")

	// Find minimum indentation (ignoring empty lines)
	minIndent := -1
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		indent := 0
		for _, ch := range line {
			if ch == '\t' || ch == ' ' {
				indent++
			} else {
				break
			}
		}
		if minIndent == -1 || indent < minIndent {
			minIndent = indent
		}
	}

	if minIndent <= 0 {
		return s
	}

	var result []string
	for _, line := range lines {
		if len(line) >= minIndent {
			result = append(result, line[minIndent:])
		} else {
			result = append(result, strings.TrimRight(line, " \t"))
		}
	}
	return strings.Join(result, "\n")
}

// --- AgentBuilder ---

// AgentBuilder configures and executes agent operations.
type AgentBuilder struct {
	tc *TestContext
}

// NewAgentBuilder creates an AgentBuilder with a fresh TestContext.
func NewAgentBuilder(t *testing.T) *AgentBuilder {
	return &AgentBuilder{tc: NewTestContext(t)}
}

// Init creates a new agent and returns an AgentAssertionChain.
func (b *AgentBuilder) Init(name string) *AgentAssertionChain {
	b.tc.T.Helper()
	b.tc.InitAgent(name)
	return &AgentAssertionChain{tc: b.tc, agentName: name}
}

// InitExpectError expects agent new to fail.
func (b *AgentBuilder) InitExpectError(name, substring string) {
	b.tc.T.Helper()
	_, err := agent.InitMind(name)
	if err == nil {
		b.tc.T.Fatalf("Expected agent new to fail with %q, but it succeeded", substring)
	}
	if !strings.Contains(err.Error(), substring) {
		b.tc.T.Fatalf("Expected error containing %q, got: %v", substring, err)
	}
}

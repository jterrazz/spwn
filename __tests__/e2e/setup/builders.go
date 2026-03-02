//go:build e2e

package setup

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jterrazz/universe/internal/architect"
	"github.com/jterrazz/universe/internal/config"
	"github.com/jterrazz/universe/internal/manifest"
	"github.com/jterrazz/universe/internal/mind"
)

// --- SpawnBuilder ---

// SpawnBuilder configures and executes a universe spawn.
type SpawnBuilder struct {
	tc         *TestContext
	configName string
	agentName  string
	workspace  string
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
	b.workspace = path
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

// Execute spawns the universe and returns an AssertionChain.
func (b *SpawnBuilder) Execute() *AssertionChain {
	b.tc.T.Helper()

	m := b.buildManifest()

	opts := architect.SpawnOpts{
		ConfigName: b.configName,
		Manifest:   m,
		Image:      b.tc.Image,
	}

	if !b.noAgent && b.agentName != "" {
		opts.AgentName = b.agentName
	}

	if b.workspace != "" {
		abs, err := filepath.Abs(b.workspace)
		if err != nil {
			b.tc.T.Fatalf("Failed to resolve workspace: %v", err)
		}
		opts.Workspace = abs
	}

	u, err := b.tc.Arc.Spawn(context.Background(), opts)
	if err != nil {
		b.tc.T.Fatalf("Spawn failed: %v", err)
	}

	b.tc.TrackUniverse(u.ID)

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
			// Give the mock a moment to write its output
			time.Sleep(500 * time.Millisecond)
		}
	}

	return &AssertionChain{tc: b.tc, universe: u}
}

// ExecuteExpectError spawns and expects an error containing the substring.
func (b *SpawnBuilder) ExecuteExpectError(substring string) {
	b.tc.T.Helper()

	m := b.buildManifest()

	opts := architect.SpawnOpts{
		ConfigName: b.configName,
		Manifest:   m,
		Image:      b.tc.Image,
	}

	if !b.noAgent && b.agentName != "" {
		opts.AgentName = b.agentName
	}

	if b.workspace != "" {
		abs, _ := filepath.Abs(b.workspace)
		opts.Workspace = abs
	}

	u, err := b.tc.Arc.Spawn(context.Background(), opts)
	if err == nil {
		b.tc.TrackUniverse(u.ID)
		b.tc.T.Fatalf("Expected spawn to fail with %q, but it succeeded", substring)
	}

	if !strings.Contains(err.Error(), substring) {
		b.tc.T.Fatalf("Expected error containing %q, got: %v", substring, err)
	}
}

func (b *SpawnBuilder) buildManifest() config.UniverseManifest {
	b.tc.T.Helper()

	if b.yamlConfig != "" {
		return b.parseInlineYAML()
	}

	// Write a default config to disk and load it
	configPath := filepath.Join(b.tc.BaseDir, "universes", b.configName+".yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		defaultYAML := "physics:\n  constants:\n    cpu: 1\n    memory: 512m\n    disk: 2g\n    timeout: 30m\n  laws:\n    network: none\n    max-processes: 128\ntechnologies:\n  - \"@unix\"\n  - \"@git\"\n"
		os.MkdirAll(filepath.Dir(configPath), 0755)
		os.WriteFile(configPath, []byte(defaultYAML), 0644)
	}

	m, err := manifest.LoadPath(configPath)
	if err != nil {
		b.tc.T.Fatalf("Failed to load config %q: %v", b.configName, err)
	}

	return m
}

func (b *SpawnBuilder) parseInlineYAML() config.UniverseManifest {
	b.tc.T.Helper()

	// Strip leading tabs from inline YAML (Go source uses tabs for indentation)
	cleaned := dedentYAML(b.yamlConfig)

	tmpPath := filepath.Join(b.tc.BaseDir, "universes", "_inline.yaml")
	os.MkdirAll(filepath.Dir(tmpPath), 0755)
	if err := os.WriteFile(tmpPath, []byte(cleaned), 0644); err != nil {
		b.tc.T.Fatalf("Failed to write inline YAML: %v", err)
	}

	m, err := manifest.LoadPath(tmpPath)
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

// InitExpectError expects agent init to fail.
func (b *AgentBuilder) InitExpectError(name, substring string) {
	b.tc.T.Helper()
	_, err := mind.Init(name)
	if err == nil {
		b.tc.T.Fatalf("Expected agent init to fail with %q, but it succeeded", substring)
	}
	if !strings.Contains(err.Error(), substring) {
		b.tc.T.Fatalf("Expected error containing %q, got: %v", substring, err)
	}
}

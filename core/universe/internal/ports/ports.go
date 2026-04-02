// Package ports defines the canonical interfaces (ports) for the universe domain.
// Implementations live in adapter packages; domain code depends only on these interfaces.
package ports

import (
	"context"
	"io"
	"time"
)

// ---------------------------------------------------------------------------
// Runtime
// ---------------------------------------------------------------------------

// Runtime abstracts the AI agent runtime (Claude Code, Pi, Codex, etc.)
type Runtime interface {
	// Spawn starts an agent inside a running container, blocking until completion.
	Spawn(ctx context.Context, cfg SpawnConfig) (int, error)
	// SpawnDetached starts an agent in the background.
	SpawnDetached(ctx context.Context, cfg SpawnConfig) error
	// Name returns the runtime identifier (e.g., "claude-code", "pi").
	Name() string
}

// SpawnConfig holds parameters for launching an agent inside a container.
type SpawnConfig struct {
	ContainerID string
	Backend     Backend
	MindPath    string
	SessionID   string
	Resume      bool
	Env         []string
	TTY         bool
	Stdin       io.Reader
	Stdout      io.Writer
	Stderr      io.Writer
}

// ---------------------------------------------------------------------------
// Backend
// ---------------------------------------------------------------------------

// Backend abstracts the container runtime (Docker, Podman, K8s, etc.)
// This is the canonical port definition. The existing backend.Backend interface
// in core/universe/internal/backend should be kept in sync with this interface.
type Backend interface {
	Create(ctx context.Context, cfg ContainerConfig) (string, error)
	Start(ctx context.Context, containerID string) error
	Stop(ctx context.Context, containerID string) error
	Remove(ctx context.Context, containerID string) error
	Exec(ctx context.Context, containerID string, cfg ExecConfig) (int, error)
	ExecDetached(ctx context.Context, containerID string, cfg ExecConfig) error
	ExecOutput(ctx context.Context, containerID string, cmd []string) (string, error)
	CopyTo(ctx context.Context, containerID string, destPath string, content []byte) error
	IsRunning(ctx context.Context, containerID string) (bool, error)
	ImageExists(ctx context.Context, image string) (bool, error)
	EnsureImage(ctx context.Context, tag string, dockerfile []byte, logw io.Writer) error
	Logs(ctx context.Context, containerID string, cfg LogsConfig) (io.ReadCloser, error)
}

// ---------------------------------------------------------------------------
// Provider
// ---------------------------------------------------------------------------

// Provider abstracts the LLM provider (Anthropic, OpenAI, Google, etc.)
type Provider interface {
	Name() string
	RequiredEnvVars() []string
}

// ---------------------------------------------------------------------------
// Memory
// ---------------------------------------------------------------------------

// Memory abstracts agent Mind persistence (filesystem, git, S3, etc.)
type Memory interface {
	Init(name string) (string, error)
	Validate(name string) error
	AgentDir(name string) string
	List() ([]AgentSummary, error)
	Inspect(name string) (*AgentDetail, error)
	Export(name string, outputPath string, excludeLayers []string) (string, error)
	Import(name string, archivePath string) error
}

// ---------------------------------------------------------------------------
// Store
// ---------------------------------------------------------------------------

// Store abstracts state persistence (JSON file, SQLite, Postgres, etc.)
type Store interface {
	List() ([]WorldRecord, error)
	Get(id string) (*WorldRecord, error)
	Save(w WorldRecord) error
	Delete(id string) error
	UpdateStatus(id string, status string) error
}

// ---------------------------------------------------------------------------
// Channel
// ---------------------------------------------------------------------------

// Channel abstracts communication channels (currently CLI only).
type Channel interface {
	Name() string
	Send(ctx context.Context, msg ChannelMessage) error
	Receive(ctx context.Context) (<-chan ChannelMessage, error)
	Close() error
}

// ---------------------------------------------------------------------------
// Skill
// ---------------------------------------------------------------------------

// Skill abstracts skill discovery and management (local, marketplace, git).
type Skill interface {
	List(ctx context.Context) ([]SkillInfo, error)
	Install(ctx context.Context, source string) error
	Remove(ctx context.Context, name string) error
}

// ---------------------------------------------------------------------------
// Tool
// ---------------------------------------------------------------------------

// Tool abstracts tool registration and invocation (built-in, MCP, gate bridges).
type Tool interface {
	Name() string
	Invoke(ctx context.Context, args []string) (ToolResult, error)
	Capabilities() []string
}

// ---------------------------------------------------------------------------
// Supporting types
// ---------------------------------------------------------------------------

// ContainerConfig defines how to create a container.
// Fields are aligned with the existing backend.ContainerConfig.
type ContainerConfig struct {
	Image       string
	Name        string
	CPU         int64
	Memory      int64
	PidsLimit   int64
	NetworkMode string
	Binds       []string
	Env         []string
	ExtraHosts  []string
}

// ExecConfig defines a command to run inside a container.
// Aligned with the existing backend.ExecConfig.
type ExecConfig struct {
	Cmd []string
	Env []string
	TTY bool
}

// LogsConfig controls log streaming behavior.
// Aligned with the existing backend.LogsConfig.
type LogsConfig struct {
	Follow bool
	Tail   string
}

// ChannelMessage represents a message sent through a Channel.
type ChannelMessage struct {
	From    string
	To      string
	Content string
	Time    time.Time
}

// SkillInfo describes an installed or available skill.
type SkillInfo struct {
	Name        string
	Version     string
	Description string
	Source      string // "local", "marketplace", "git"
}

// ToolResult is the outcome of a Tool invocation.
type ToolResult struct {
	Output   string
	ExitCode int
	Error    string
}

// AgentSummary is a lightweight view of a stored agent/mind.
type AgentSummary struct {
	Name       string
	LayerCount int
}

// AgentDetail provides full information about a stored agent/mind.
type AgentDetail struct {
	Name   string
	Layers map[string][]string // layer name -> file list
}

// WorldRecord is the persistence representation of a universe instance.
// Aligned with the existing models.World type.
type WorldRecord struct {
	ID          string
	Config      string
	Status      string
	Agent       string
	AgentID     string
	Backend     string
	ContainerID string
	Workspace   string
	MindPath    string
	GateDir     string
	CreatedAt   time.Time
}

package config

import "time"

// Interaction defines a bridge from an external MCP server to a CLI command inside the universe.
type Interaction struct {
	Source       string   // MCP server ID, e.g. "mcp/slack"
	As           string   // CLI command name inside universe, e.g. "slack-send"
	Capabilities []string // Subset of capabilities to expose
	Description  string   // Human-readable for physics.md (optional)
}

// UniverseConfig holds the configuration for creating a universe.
type UniverseConfig struct {
	Image        string
	Mind         string
	Workspace    string
	Memory       string
	CPU          float64
	Timeout      time.Duration
	Interactions []Interaction
	GateDir      string // Host directory containing gate.sock, mounted at /gate
}

// Universe represents a running or stopped universe.
type Universe struct {
	ID        string
	Status    UniverseStatus
	Image     string
	Mind      string
	CreatedAt time.Time
}

// UniverseStatus represents the state of a universe.
type UniverseStatus string

const (
	StatusCreated UniverseStatus = "created"
	StatusRunning UniverseStatus = "running"
	StatusStopped UniverseStatus = "stopped"
)

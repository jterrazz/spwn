package models

import (
	"time"
)

// Manifest is the parsed representation of a world config YAML.
type Manifest struct {
	Tools   []string `yaml:"-"`
	Plugins []string `yaml:"-"`
}

// Workspace is a single host directory mounted into a world. A world may have
// zero or more workspaces. When a world has zero workspaces it is "ephemeral"
// and works inside the image's pre-baked /workspace dir.
type Workspace struct {
	Name     string `json:"name"`              // Mount subdirectory under /workspaces (e.g. "web", "api").
	Path     string `json:"path"`              // Absolute host path.
	ReadOnly bool   `json:"readonly,omitempty"`
}

// AgentRecord represents a single agent within a world colony.
type AgentRecord struct {
	Name      string `json:"name"`
	AgentID   string `json:"agent_id"`
	Role      string `json:"role"`               // "chief", "manager", "worker", or "npc"
	Ephemeral bool   `json:"ephemeral,omitempty"` // true for NPC-style throwaway agents
	Status    Status `json:"status"`
}

// World represents a running or stopped world instance.
type World struct {
	ID          string        `json:"id"`
	Name        string        `json:"name,omitempty"` // Optional display name; when empty UIs fall back to the ID.
	Config      string        `json:"config"`
	Agent       string        `json:"agent,omitempty"`
	AgentID     string        `json:"agent_id,omitempty"`
	Backend     string        `json:"backend"`
	ContainerID string        `json:"container_id"`
	Workspaces  []Workspace   `json:"workspaces,omitempty"`
	// Legacy single-workspace field. Retained so old state files unmarshal cleanly.
	// The state store migrates this into Workspaces on load and clears it.
	Workspace    string            `json:"workspace,omitempty"`
	Organization string            `json:"organization,omitempty"` // optional organization name
	SessionIDs   map[string]string `json:"session_ids,omitempty"`  // agent name → runtime session ID
	Status      Status            `json:"status"`
	CreatedAt   time.Time     `json:"created_at"`
	Agents      []AgentRecord `json:"agents,omitempty"` // multi-agent support
	Manifest    Manifest      `json:"manifest,omitempty"`
}

// PrimaryWorkspacePath returns the first workspace's host path, or empty if
// the world is ephemeral (no host mounts).
func (w World) PrimaryWorkspacePath() string {
	if len(w.Workspaces) == 0 {
		return ""
	}
	return w.Workspaces[0].Path
}

// Status tracks the lifecycle state of a world.
type Status string

const (
	StatusCreating  Status = "creating"
	StatusRunning   Status = "running"
	StatusIdle      Status = "idle"
	StatusStopped   Status = "stopped"
	StatusDestroyed Status = "destroyed"
)

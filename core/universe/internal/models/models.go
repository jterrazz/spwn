package models

import (
	"time"

	"spwn.sh/core/gate"
)

// Manifest is the parsed representation of a universe config YAML.
type Manifest struct {
	Physics  PhysicsManifest   `yaml:"physics"`
	Elements []string          `yaml:"-"`
	Gate     []gate.Bridge `yaml:"-"`
}

// PhysicsManifest defines the physical constraints of a universe.
type PhysicsManifest struct {
	Constants ConstantsManifest `yaml:"constants"`
	Laws      LawsManifest      `yaml:"laws"`
}

// ConstantsManifest defines fixed resource limits.
type ConstantsManifest struct {
	CPU     int    `yaml:"cpu"`
	Memory  string `yaml:"memory"`
	Disk    string `yaml:"disk"`
	Timeout string `yaml:"timeout"`
}

// LawsManifest defines invariant rules.
type LawsManifest struct {
	MaxProcesses int `yaml:"max-processes"`
}

// AgentRecord represents a single agent within a universe colony.
type AgentRecord struct {
	Name    string `json:"name"`
	AgentID string `json:"agent_id"`
	Tier    string `json:"tier"`   // "governor" or "citizen"
	Status  Status `json:"status"`
}

// World represents a running or stopped universe instance.
type World struct {
	ID          string        `json:"id"`
	Config      string        `json:"config"`
	Agent       string        `json:"agent,omitempty"`
	AgentID     string        `json:"agent_id,omitempty"`
	Backend     string        `json:"backend"`
	ContainerID string        `json:"container_id"`
	Workspace   string        `json:"workspace,omitempty"`
	MindPath    string        `json:"mind_path,omitempty"`
	GateDir     string        `json:"gate_dir,omitempty"`
	Status      Status        `json:"status"`
	CreatedAt   time.Time     `json:"created_at"`
	Agents      []AgentRecord `json:"agents,omitempty"` // multi-agent support
	Manifest    Manifest      `json:"-"`
}

// Status tracks the lifecycle state of a universe.
type Status string

const (
	StatusCreating  Status = "creating"
	StatusRunning   Status = "running"
	StatusIdle      Status = "idle"
	StatusStopped   Status = "stopped"
	StatusDestroyed Status = "destroyed"
)

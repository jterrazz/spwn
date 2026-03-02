package config

import "time"

// UniverseManifest is the parsed representation of a universe config YAML.
type UniverseManifest struct {
	Physics  PhysicsManifest `yaml:"physics"`
	Elements []string        `yaml:"-"`
	Gate     []GateBridge    `yaml:"-"`
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
	Network      string `yaml:"network"`
	MaxProcesses int    `yaml:"max-processes"`
}

// GateBridge represents an element bridged from the Substrate.
type GateBridge struct {
	Source       string   `yaml:"source"`
	As           string   `yaml:"as"`
	Capabilities []string `yaml:"capabilities"`
}

// Universe represents a running or stopped universe instance.
type Universe struct {
	ID          string           `json:"id"`
	Config      string           `json:"config"`
	Agent       string           `json:"agent,omitempty"`
	AgentID     string           `json:"agent_id,omitempty"`
	Backend     string           `json:"backend"`
	ContainerID string           `json:"container_id"`
	Workspace   string           `json:"workspace,omitempty"`
	MindPath    string           `json:"mind_path,omitempty"`
	GateDir     string           `json:"gate_dir,omitempty"`
	Status      UniverseStatus   `json:"status"`
	CreatedAt   time.Time        `json:"created_at"`
	Manifest    UniverseManifest `json:"-"`
}

// UniverseStatus tracks the lifecycle state of a universe.
type UniverseStatus string

const (
	StatusCreating  UniverseStatus = "creating"
	StatusRunning   UniverseStatus = "running"
	StatusIdle      UniverseStatus = "idle"
	StatusStopped   UniverseStatus = "stopped"
	StatusDestroyed UniverseStatus = "destroyed"
)

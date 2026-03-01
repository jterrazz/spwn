package config

import "time"

// UniverseManifest represents a parsed universe.yaml.
type UniverseManifest struct {
	Physics PhysicsManifest `yaml:"physics" json:"physics"`
	Gate    []GateBridge    `yaml:"gate,omitempty" json:"gate,omitempty"`
}

// PhysicsManifest defines the physics of a universe.
type PhysicsManifest struct {
	Origin    string            `yaml:"origin" json:"origin"`
	Constants ConstantsManifest `yaml:"constants,omitempty" json:"constants,omitempty"`
	Laws      LawsManifest      `yaml:"laws,omitempty" json:"laws,omitempty"`
	Elements  []string          `yaml:"elements,omitempty" json:"elements,omitempty"`
}

// ConstantsManifest defines resource limits.
type ConstantsManifest struct {
	CPU     int    `yaml:"cpu,omitempty" json:"cpu,omitempty"`
	Memory  string `yaml:"memory,omitempty" json:"memory,omitempty"`
	Disk    string `yaml:"disk,omitempty" json:"disk,omitempty"`
	Timeout string `yaml:"timeout,omitempty" json:"timeout,omitempty"`
}

// LawsManifest defines invariant rules.
type LawsManifest struct {
	Network      string `yaml:"network,omitempty" json:"network,omitempty"`
	MaxProcesses int    `yaml:"max-processes,omitempty" json:"max_processes,omitempty"`
}

// GateBridge defines an MCP bridge configuration.
type GateBridge struct {
	Source       string   `yaml:"source" json:"source"`
	As           string   `yaml:"as" json:"as"`
	Capabilities []string `yaml:"capabilities,omitempty" json:"capabilities,omitempty"`
}

// MindManifest represents a parsed mind.yaml.
type MindManifest struct {
	Name      string   `yaml:"name" json:"name"`
	Personas  []string `yaml:"personas,omitempty" json:"personas,omitempty"`
	Skills    []string `yaml:"skills,omitempty" json:"skills,omitempty"`
	Knowledge []string `yaml:"knowledge,omitempty" json:"knowledge,omitempty"`
	Playbooks []string `yaml:"playbooks,omitempty" json:"playbooks,omitempty"`
}

// Universe represents a live or stopped universe instance.
type Universe struct {
	ID          string         `json:"id"`
	Origin      string         `json:"origin"`
	Agent       string         `json:"agent,omitempty"`
	Backend     string         `json:"backend"`
	Status      UniverseStatus `json:"status"`
	ContainerID string         `json:"container_id"`
	Workspace   string         `json:"workspace,omitempty"`
	MindPath    string         `json:"mind_path,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	Manifest    UniverseManifest `json:"manifest"`
}

// UniverseStatus represents the current state of a universe.
type UniverseStatus string

const (
	StatusCreating  UniverseStatus = "creating"
	StatusRunning   UniverseStatus = "running"
	StatusIdle      UniverseStatus = "idle"
	StatusStopped   UniverseStatus = "stopped"
	StatusDestroyed UniverseStatus = "destroyed"
)

// SpawnOptions captures all parameters for creating a universe.
type SpawnOptions struct {
	Manifest  UniverseManifest
	Workspace string
	AgentName string
	Detach    bool
}

// Defaults.
const (
	DefaultOrigin     = "ubuntu:24.04"
	DefaultCPU        = 1
	DefaultMemory     = "512m"
	DefaultDisk       = "2g"
	DefaultTimeout    = "30m"
	DefaultNetwork    = "none"
	DefaultMaxProcs   = 128
	DefaultBackend    = "docker"
	BaseImage         = "universe-base:latest"
)

// Directories.
const (
	UniverseBaseDir = ".universe"
	AgentsSubDir    = "agents"
	StateFileName   = "universes.json"
)

// MindLayers lists the six standard Mind directories.
var MindLayers = []string{
	"personas",
	"skills",
	"knowledge",
	"playbooks",
	"journal",
	"sessions",
}

package config

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

// Universe config defaults.
const (
	DefaultCPU      = 1
	DefaultMemory   = "512m"
	DefaultDisk     = "2g"
	DefaultTimeout  = "30m"
	DefaultNetwork  = "none"
	DefaultMaxProcs = 128
	DefaultBackend  = "docker"
	BaseImage       = "universe-base:latest"
)

// Directory layout constants.
const (
	UniverseBaseDir = ".universe"
	UniversesSubDir = "universes"
	AgentsSubDir    = "agents"
	StateFileName   = "state.json"
)

// MindLayers defines the six-layer Mind structure.
var MindLayers = []string{"personas", "skills", "knowledge", "playbooks", "journal", "sessions"}

// UniverseManifest is the parsed representation of a universe config YAML.
type UniverseManifest struct {
	Physics      PhysicsManifest `yaml:"physics"`
	Technologies []string        `yaml:"-"`
	Gate         []GateBridge    `yaml:"-"`
}

// PhysicsManifest defines the physical constraints of a universe.
type PhysicsManifest struct {
	Constants ConstantsManifest `yaml:"constants"`
	Laws      LawsManifest      `yaml:"laws"`
	Elements  []ElementMount    `yaml:"elements"`
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

// ElementMount represents raw matter in the universe.
type ElementMount struct {
	Name string
	Path string
}

// GateBridge represents a technology bridged from the Substrate.
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

// BaseDir returns the path to ~/.universe/.
func BaseDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, UniverseBaseDir)
}

// UniversesDir returns the path to ~/.universe/universes/.
func UniversesDir() string {
	return filepath.Join(BaseDir(), UniversesSubDir)
}

// AgentsDir returns the path to ~/.universe/agents/.
func AgentsDir() string {
	return filepath.Join(BaseDir(), AgentsSubDir)
}

// StatePath returns the path to ~/.universe/state.json.
func StatePath() string {
	return filepath.Join(BaseDir(), StateFileName)
}

// GenerateUniverseID returns an ID like u-default-84721.
func GenerateUniverseID(configName string) string {
	return fmt.Sprintf("u-%s-%s", configName, randDigits(5))
}

// GenerateAgentID returns an ID like a-leonardo-52103.
func GenerateAgentID(agentName string) string {
	return fmt.Sprintf("a-%s-%s", agentName, randDigits(5))
}

func randDigits(n int) string {
	s := ""
	for i := 0; i < n; i++ {
		d, _ := rand.Int(rand.Reader, big.NewInt(10))
		s += fmt.Sprintf("%d", d.Int64())
	}
	return s
}

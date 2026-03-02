package config

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

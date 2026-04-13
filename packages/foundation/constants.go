package foundation

// Default physics constants.
const (
	DefaultCPU      = 1
	DefaultMemory   = "512m"
	DefaultDisk     = "2g"
	DefaultTimeout  = "30m"
	DefaultMaxProcs = 128
	DefaultBackend  = "docker"
	WorldImage = "spwn/world:latest"
)

// Architect daemon constants.
const (
	ArchitectContainerName = "spwn-architect"
	ArchitectImage         = "spwn/architect:latest"
)

// Image versioning constants.
const (
	WorldImageVersion     = "1.1.0"
	ArchitectImageVersion = "1.1.0"
	ImageVersionLabel     = "sh.spwn.image-version"
)

// Directory layout constants.
const (
	SpwnBaseDir       = ".spwn"
	WorldsSubDir      = "worlds"
	AgentsSubDir      = "agents"
	StateFileName     = "state.json"
	ClawStateFileName = "claw.json"
	SkillsSubDir      = "skills"
	ClawSubDir        = "claw"
	CredentialsSubDir   = "credentials"
	ActivityFileName  = "activity.jsonl"
	TeamsSubDir       = "teams"
	OrganizationsSubDir = "organizations"
)

// MindLayers defines the five-layer Mind structure.
var MindLayers = []string{"core", "skills", "knowledge", "playbooks", "journal"}

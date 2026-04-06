package foundation

// Default physics constants.
const (
	DefaultCPU      = 1
	DefaultMemory   = "512m"
	DefaultDisk     = "2g"
	DefaultTimeout  = "30m"
	DefaultMaxProcs = 128
	DefaultBackend  = "docker"
	BaseImage       = "spwn-base:latest"
	GodImage        = "spwn-god:latest"
)

// Architect daemon constants.
const (
	ArchitectContainerName = "spwn-architect"
	ArchitectImage         = "spwn-architect:latest"
)

// Directory layout constants.
const (
	SpwnBaseDir       = ".spwn"
	WorldsSubDir      = "worlds"
	AgentsSubDir      = "agents"
	StateFileName     = "state.json"
	OrgFileName       = "org.yaml"
	ClawStateFileName = "claw.json"
	SkillsSubDir      = "skills"
	ClawSubDir        = "claw"
	KnowledgeSubDir   = "knowledge"
	ActivityFileName  = "activity.jsonl"
	TeamsSubDir       = "teams"
)

// MindLayers defines the six-layer Mind structure.
var MindLayers = []string{"identity", "skills", "memory/knowledge", "memory/playbooks", "memory/journal", "sessions"}

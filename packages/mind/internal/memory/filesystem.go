package memory

import (
	"spwn.sh/packages/mind/internal/mind"
)

// AgentInfo describes an agent's Mind structure.
type AgentInfo = mind.AgentInfo

// FilesystemMemory implements the Memory port using the local filesystem.
type FilesystemMemory struct {
	basePath string // defaults to ~/.spwn/agents/
}

// NewFilesystem creates a new FilesystemMemory adapter.
func NewFilesystem(basePath string) *FilesystemMemory {
	return &FilesystemMemory{basePath: basePath}
}

// BasePath returns the configured base path for agent storage.
func (f *FilesystemMemory) BasePath() string {
	return f.basePath
}

// Init scaffolds a new Mind with all 6 layers.
func (f *FilesystemMemory) Init(name string) (string, error) {
	return mind.Init(name)
}

// Validate checks that a Mind directory exists and has the core layer.
func (f *FilesystemMemory) Validate(name string) error {
	return mind.Validate(name)
}

// List returns all agents in the agents directory.
func (f *FilesystemMemory) List() ([]AgentInfo, error) {
	return mind.List()
}

// Inspect returns detailed information about an agent's Mind.
func (f *FilesystemMemory) Inspect(name string) (*AgentInfo, error) {
	return mind.Inspect(name)
}

// LayerCount returns how many layers have at least one file.
func (f *FilesystemMemory) LayerCount(info *AgentInfo) int {
	return mind.LayerCount(info)
}

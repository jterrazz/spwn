package mind

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jterrazz/universe/internal/config"
)

// AgentInfo describes an agent's Mind structure.
type AgentInfo struct {
	Name   string              `json:"name"`
	Path   string              `json:"path"`
	Layers map[string][]string `json:"layers"`
}

// AgentDir returns the path to ~/.universe/agents/{name}/.
func AgentDir(name string) string {
	return filepath.Join(config.AgentsDir(), name)
}

// Init scaffolds a new Mind with all 6 layers.
func Init(name string) (string, error) {
	dir := AgentDir(name)
	if _, err := os.Stat(dir); err == nil {
		return "", fmt.Errorf("agent %q already exists", name)
	}

	for _, layer := range config.MindLayers {
		if err := os.MkdirAll(filepath.Join(dir, layer), 0755); err != nil {
			return "", fmt.Errorf("create %s: %w", layer, err)
		}
	}

	// Create default persona
	persona := `# Default Persona

## Role
You are a general-purpose assistant. You operate autonomously within the physics
of your universe.

## Operating Principles
- Read /universe/physics.md at startup to understand the world's constraints
- Read /universe/faculties.md to understand available technologies
- Stay within the Laws — they describe what's physically possible
`
	personaPath := filepath.Join(dir, "personas", "default.md")
	if err := os.WriteFile(personaPath, []byte(persona), 0644); err != nil {
		return "", fmt.Errorf("create persona: %w", err)
	}

	return dir, nil
}

// Validate checks that a Mind directory exists and has the personas layer.
func Validate(name string) error {
	dir := AgentDir(name)
	info, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("agent %q not found", name)
	}
	if !info.IsDir() {
		return fmt.Errorf("agent %q is not a directory", name)
	}

	personas := filepath.Join(dir, "personas")
	if _, err := os.Stat(personas); err != nil {
		return fmt.Errorf("agent %q is missing the personas/ layer", name)
	}
	return nil
}

// List returns all agents in ~/.universe/agents/.
func List() ([]AgentInfo, error) {
	dir := config.AgentsDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var agents []AgentInfo
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		info, err := Inspect(e.Name())
		if err != nil {
			continue
		}
		agents = append(agents, *info)
	}
	return agents, nil
}

// Inspect returns detailed information about an agent's Mind.
func Inspect(name string) (*AgentInfo, error) {
	dir := AgentDir(name)
	if _, err := os.Stat(dir); err != nil {
		return nil, fmt.Errorf("agent %q not found", name)
	}

	info := &AgentInfo{
		Name:   name,
		Path:   dir,
		Layers: make(map[string][]string),
	}

	for _, layer := range config.MindLayers {
		layerDir := filepath.Join(dir, layer)
		entries, err := os.ReadDir(layerDir)
		if err != nil {
			info.Layers[layer] = nil
			continue
		}
		var files []string
		for _, e := range entries {
			if !e.IsDir() {
				files = append(files, e.Name())
			}
		}
		info.Layers[layer] = files
	}

	return info, nil
}

// LayerCount returns how many layers have at least one file.
func LayerCount(info *AgentInfo) int {
	count := 0
	for _, files := range info.Layers {
		if len(files) > 0 {
			count++
		}
	}
	return count
}

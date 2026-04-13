package mind

import (
	"fmt"
	"os"
	"path/filepath"

	"spwn.sh/packages/foundation"
)

// AgentInfo describes an agent's Mind structure.
type AgentInfo struct {
	Name   string              `json:"name"`
	Path   string              `json:"path"`
	Team   string              `json:"team,omitempty"`
	Layers map[string][]string `json:"layers"`
}

// AgentDir returns the path to ~/.spwn/agents/{name}/.
func AgentDir(name string) string {
	return filepath.Join(foundation.AgentsDir(), name)
}

// Init scaffolds a new Mind with all 6 layers.
func Init(name string) (string, error) {
	dir := AgentDir(name)
	if _, err := os.Stat(dir); err == nil {
		return "", fmt.Errorf("agent %q already exists", name)
	}

	for _, layer := range foundation.MindLayers {
		if err := os.MkdirAll(filepath.Join(dir, layer), 0755); err != nil {
			return "", fmt.Errorf("create %s: %w", layer, err)
		}
	}

	// Create default profile
	profile := `# Default Profile

You are a spwn agent — a persistent AI worker living inside an isolated world.

## Your Identity
- You have a Mind that persists across sessions at /mind (core, skills, knowledge, playbooks, journal)
- Your identity defines your purpose and values — you are reading it now
- You evolve through experience: dream to analyze tasks, learn from outcomes, update your knowledge

## Your World
- Read /universe/physics.md to understand your world's constants and laws
- Read /universe/faculties.md for available tools
- Check /world/AGENT.md for your specific role and instructions
- Your workspace is at /workspace

## Communication
- Check your inbox at /world/inbox/{your-name}/ for messages from other agents
- Send messages to other agents by writing to /world/inbox/{their-name}/
- Save important learnings to /mind/knowledge/

## Behavior
- Be concise and action-oriented — execute tasks directly
- Use your full Unix shell access (bash, git, curl, etc.)
- Stay within the Laws — they describe what is physically possible
`
	profilePath := filepath.Join(dir, "core", "profile.md")
	if err := os.WriteFile(profilePath, []byte(profile), 0644); err != nil {
		return "", fmt.Errorf("create profile: %w", err)
	}

	return dir, nil
}

// Validate checks that a Mind directory exists and has the core layer.
func Validate(name string) error {
	dir := AgentDir(name)
	info, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("agent %q not found", name)
	}
	if !info.IsDir() {
		return fmt.Errorf("agent %q is not a directory", name)
	}

	coreDir := filepath.Join(dir, "core")
	if _, err := os.Stat(coreDir); err != nil {
		return fmt.Errorf("agent %q is missing the core/ layer", name)
	}
	return nil
}

// List returns all agents in ~/.spwn/agents/.
func List() ([]AgentInfo, error) {
	dir := foundation.AgentsDir()
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

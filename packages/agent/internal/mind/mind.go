package mind

import (
	"spwn.sh/packages/platform"
	"fmt"
	"os"
	"path/filepath"
	"strings"

)

// defaultAgentYAMLTmpl is the baseline agent.yaml written by
// Init/Repair so `spwn check` passes immediately after
// `agent create`. Matches the shape of
// packages/project/internal/scaffold/templates/agent.yaml.tmpl but
// parameterised by name.
const defaultAgentYAMLTmpl = `# Agent composition — the source of truth for the agent's packages
# and runtime. When the agent is deployed in a world alongside others,
# the union of every member's packages is what gets baked into the
# resulting container.

name: __NAME__
description: A blank-slate agent — replace this one-liner with what __NAME__ is for.

runtime:
  backend: "@spwn/claude-code"

dependencies:
  - "@spwn/unix"
  - "@spwn/git"
  - "@spwn/python"
`

// defaultAgentMDTmpl is the baseline source AGENTS.md written by
// Init/Repair. Mirrors packages/project/internal/scaffold/templates/
// AGENTS.md.tmpl. This is the provider-neutral agent prompt file; a
// runtime-specific renderer (e.g. packages/compile/runtimes/
// claudecode) is what eventually turns it into CLAUDE.md inside the
// container.
const defaultAgentMDTmpl = `# __NAME__

You are **__NAME__**, an agent running inside a spwn world.

## Your identity

Before doing anything else, read your identity:

@identity/profile.md

## Your world

- ` + "`/world/physics.md`" + `    - the rules of this world (network, filesystem, communication topology)
- ` + "`/world/faculties.md`" + `  - every tool installed and verified
- ` + "`/world/AGENTS.md`" + `     - the operating manual every spwn agent reads
- ` + "`/work/`" + `               - mounted workspaces, this is where you make changes

## Conventions

1. Read your identity first. It shapes how you respond.
2. Save important discoveries to ` + "`./knowledge/`" + ` so you remember them next time.
3. After significant work, consider promoting a pattern to ` + "`./playbooks/`" + `.
4. Before committing changes, run the project's existing tests if they exist.
5. Never modify ` + "`/world/`" + ` files - they are read-only system context.
`

func renderTmpl(tmpl, name string) []byte {
	return []byte(strings.ReplaceAll(tmpl, "__NAME__", name))
}

// AgentInfo describes an agent's Mind structure.
type AgentInfo struct {
	Name   string              `json:"name"`
	Path   string              `json:"path"`
	Team   string              `json:"team,omitempty"`
	Layers map[string][]string `json:"layers"`
}

// AgentDir returns the path to ~/.spwn/agents/{name}/.
func AgentDir(name string) string {
	return filepath.Join(platform.AgentsDir(), name)
}

// Init scaffolds a new Mind with all 6 layers.
func Init(name string) (string, error) {
	dir := AgentDir(name)
	if _, err := os.Stat(dir); err == nil {
		return "", fmt.Errorf("agent %q already exists", name)
	}

	for _, layer := range platform.MindLayers {
		if err := os.MkdirAll(filepath.Join(dir, layer), 0755); err != nil {
			return "", fmt.Errorf("create %s: %w", layer, err)
		}
	}

	// Create default profile
	profile := `# Default Profile

You are a spwn agent - a persistent AI worker living inside an isolated world.

## Your Identity
- You have a Mind that persists across sessions at /mind (identity, skills, knowledge, playbooks, journal)
- Your identity defines your purpose and values - you are reading it now
- You evolve through experience: dream to analyze tasks, learn from outcomes, update your knowledge

## Your World
- Read /world/physics.md to understand your world's rules (network, filesystem, communication)
- Read /world/faculties.md for available tools
- Check /world/AGENT.md for your specific role and instructions
- Your workspace is at /workspace

## Communication
- Check your inbox at /world/inbox/{your-name}/ for messages from other agents
- Send messages to other agents by writing to /world/inbox/{their-name}/
- Save important learnings to /mind/knowledge/

## Behavior
- Be concise and action-oriented - execute tasks directly
- Use your full Unix shell access (bash, git, curl, etc.)
- Stay within the Laws - they describe what is physically possible
`
	profilePath := filepath.Join(dir, "identity", "profile.md")
	if err := os.WriteFile(profilePath, []byte(profile), 0644); err != nil {
		return "", fmt.Errorf("create profile: %w", err)
	}

	// Write the baseline agent.yaml and AGENTS.md so `spwn check`
	// passes immediately after `agent create`. Both files are
	// required by the project validator (ruleAgentStructure).
	if err := os.WriteFile(filepath.Join(dir, "agent.yaml"), renderTmpl(defaultAgentYAMLTmpl, name), 0644); err != nil {
		return "", fmt.Errorf("create agent.yaml: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "AGENTS.md"), renderTmpl(defaultAgentMDTmpl, name), 0644); err != nil {
		return "", fmt.Errorf("create AGENTS.md: %w", err)
	}

	return dir, nil
}

// Repair re-creates any missing layer directories and the default
// profile for an already-existing Mind. Unlike Init, Repair never
// errors when the agent directory already exists — it is idempotent
// and only writes what is missing, making it safe for --force
// re-scaffold platform.
func Repair(name string) error {
	dir := AgentDir(name)
	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("agent %q not found", name)
		}
		return err
	}

	for _, layer := range platform.MindLayers {
		if err := os.MkdirAll(filepath.Join(dir, layer), 0755); err != nil {
			return fmt.Errorf("create %s: %w", layer, err)
		}
	}

	profilePath := filepath.Join(dir, "identity", "profile.md")
	if _, err := os.Stat(profilePath); err != nil && os.IsNotExist(err) {
		profile := `# Default Profile

You are a spwn agent - a persistent AI worker living inside an isolated world.
`
		if err := os.WriteFile(profilePath, []byte(profile), 0644); err != nil {
			return fmt.Errorf("create profile: %w", err)
		}
	}

	// Re-scaffold agent.yaml / AGENTS.md when missing so --force can
	// rescue a partially-deleted agent tree.
	agentYAMLPath := filepath.Join(dir, "agent.yaml")
	if _, err := os.Stat(agentYAMLPath); err != nil && os.IsNotExist(err) {
		if err := os.WriteFile(agentYAMLPath, renderTmpl(defaultAgentYAMLTmpl, name), 0644); err != nil {
			return fmt.Errorf("create agent.yaml: %w", err)
		}
	}
	entryPath := filepath.Join(dir, "AGENTS.md")
	if _, err := os.Stat(entryPath); err != nil && os.IsNotExist(err) {
		if err := os.WriteFile(entryPath, renderTmpl(defaultAgentMDTmpl, name), 0644); err != nil {
			return fmt.Errorf("create AGENTS.md: %w", err)
		}
	}
	return nil
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

	identityDir := filepath.Join(dir, "identity")
	if _, err := os.Stat(identityDir); err != nil {
		return fmt.Errorf("agent %q is missing the identity/ layer", name)
	}
	return nil
}

// List returns all agents in ~/.spwn/agents/.
func List() ([]AgentInfo, error) {
	dir := platform.AgentsDir()
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

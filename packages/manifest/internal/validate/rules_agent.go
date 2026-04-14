package validate

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// agentYAML mirrors the subset of agent.yaml we validate. We keep it
// deliberately minimal and local — packages/agent owns the richer
// schema used at runtime; this struct only cares about whether the
// file is well-formed and whether the fields that load-bearing for
// validation rules are present.
type agentYAML struct {
	Name    string   `yaml:"name"`
	Runtime string   `yaml:"runtime"`
	Tools   []string `yaml:"tools"`
}

// ruleAgentYAMLParses checks that every agent's agent.yaml is
// well-formed and has the mandatory fields (name, runtime).
func ruleAgentYAMLParses(in Input) []Issue {
	if in.Manifest == nil {
		return nil
	}
	var out []Issue
	for i, name := range in.Manifest.Agents {
		if i >= len(in.AgentPaths) || i >= len(in.AgentExists) || !in.AgentExists[i] {
			continue
		}
		agentDir := in.AgentPaths[i]
		yamlPath := filepath.Join(agentDir, "agent.yaml")
		data, err := os.ReadFile(yamlPath)
		if err != nil {
			// Missing agent.yaml is already caught by
			// ruleAgentDirs; don't double-report.
			if os.IsNotExist(err) {
				continue
			}
			out = append(out, Issue{
				Level:   LevelError,
				Path:    relPath(in.Root, yamlPath),
				Message: fmt.Sprintf("cannot read agent.yaml: %v", err),
			})
			continue
		}

		var parsed agentYAML
		if err := yaml.Unmarshal(data, &parsed); err != nil {
			out = append(out, Issue{
				Level:   LevelError,
				Path:    relPath(in.Root, yamlPath),
				Message: "agent.yaml is not valid YAML: " + err.Error(),
			})
			continue
		}

		if parsed.Name == "" {
			out = append(out, Issue{
				Level:   LevelError,
				Path:    relPath(in.Root, yamlPath) + "#name",
				Message: "agent.yaml is missing the name field",
				Hint:    "add `name: " + name + "`",
			})
		} else if parsed.Name != name {
			out = append(out, Issue{
				Level:   LevelWarning,
				Path:    relPath(in.Root, yamlPath) + "#name",
				Message: fmt.Sprintf("agent.yaml name %q does not match directory name %q", parsed.Name, name),
				Hint:    "rename the directory or update name: in agent.yaml",
			})
		}

		if parsed.Runtime == "" {
			out = append(out, Issue{
				Level:   LevelError,
				Path:    relPath(in.Root, yamlPath) + "#runtime",
				Message: "agent.yaml is missing the runtime field",
				Hint:    "add `runtime: claude-code`",
			})
		}
	}
	return out
}

// ruleAgentToolsSubsetOfWorld checks that every tool declared in
// agent.yaml is also declared in the world's world.yaml. An agent
// cannot reach for a tool the world does not provide.
func ruleAgentToolsSubsetOfWorld(in Input) []Issue {
	if in.Manifest == nil || !in.WorldExists {
		return nil
	}
	worldTools, err := loadWorldTools(in.WorldPath)
	if err != nil {
		// Parse errors are reported by ruleWorldYAMLParses; skip here.
		return nil
	}
	worldSet := make(map[string]struct{}, len(worldTools))
	for _, t := range worldTools {
		worldSet[t] = struct{}{}
	}

	var out []Issue
	for i, name := range in.Manifest.Agents {
		if i >= len(in.AgentPaths) || i >= len(in.AgentExists) || !in.AgentExists[i] {
			continue
		}
		agentYamlPath := filepath.Join(in.AgentPaths[i], "agent.yaml")
		data, err := os.ReadFile(agentYamlPath)
		if err != nil {
			continue
		}
		var parsed agentYAML
		if err := yaml.Unmarshal(data, &parsed); err != nil {
			continue
		}
		for _, tool := range parsed.Tools {
			if _, ok := worldSet[tool]; !ok {
				out = append(out, Issue{
					Level:   LevelError,
					Path:    relPath(in.Root, agentYamlPath) + "#tools",
					Message: fmt.Sprintf("agent %q declares tool %q which is not available in world %q", name, tool, in.Manifest.World),
					Hint:    fmt.Sprintf("add %q to ./spwn/worlds/%s.yaml tools:", tool, in.Manifest.World),
				})
			}
		}
	}
	return out
}

// ruleDuplicateAgents catches the same agent listed twice in the
// manifest. The runtime would silently use the first; validator
// flags it before the user hits that confusion.
func ruleDuplicateAgents(in Input) []Issue {
	if in.Manifest == nil {
		return nil
	}
	seen := map[string]int{}
	var out []Issue
	for _, name := range in.Manifest.Agents {
		seen[name]++
		if seen[name] == 2 {
			out = append(out, Issue{
				Level:   LevelError,
				Path:    "spwn.yaml#agents",
				Message: "agent " + name + " is listed more than once",
				Hint:    "remove the duplicate entry",
			})
		}
	}
	return out
}

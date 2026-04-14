package validate

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ruleRuntimeSupported checks that every agent declares a runtime
// that the host actually knows how to spawn. Callers inject the
// supported runtime list via Input.SupportedRuntimes; nil means
// "don't validate" (skip, don't false-error).
func ruleRuntimeSupported(in Input) []Issue {
	if in.Manifest == nil || len(in.SupportedRuntimes) == 0 {
		return nil
	}
	supported := make(map[string]struct{}, len(in.SupportedRuntimes))
	for _, r := range in.SupportedRuntimes {
		supported[r] = struct{}{}
	}

	var out []Issue
	for i := range in.Manifest.Agents {
		if i >= len(in.AgentPaths) || i >= len(in.AgentExists) || !in.AgentExists[i] {
			continue
		}
		yamlPath := filepath.Join(in.AgentPaths[i], "agent.yaml")
		data, err := os.ReadFile(yamlPath)
		if err != nil {
			continue
		}
		var parsed agentYAML
		if err := yaml.Unmarshal(data, &parsed); err != nil {
			continue
		}
		if parsed.Runtime == "" {
			continue // covered by ruleAgentYAMLParses
		}
		if _, ok := supported[parsed.Runtime]; !ok {
			out = append(out, Issue{
				Level:   LevelError,
				Path:    relPath(in.Root, yamlPath) + "#runtime",
				Message: fmt.Sprintf("runtime %q is not supported", parsed.Runtime),
				Hint:    "supported: " + joinStrings(in.SupportedRuntimes),
			})
		}
	}
	return out
}

func joinStrings(ss []string) string {
	out := ""
	for i, s := range ss {
		if i > 0 {
			out += ", "
		}
		out += s
	}
	return out
}

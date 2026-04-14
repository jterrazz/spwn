package validate

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// worldYAML mirrors the subset of world.yaml we validate. Like
// agentYAML above, this is a deliberate minimal view — packages/world
// owns the full schema at runtime.
type worldYAML struct {
	Physics struct {
		Constants struct {
			CPU     int    `yaml:"cpu"`
			Memory  string `yaml:"memory"`
			Disk    string `yaml:"disk"`
			Timeout string `yaml:"timeout"`
		} `yaml:"constants"`
	} `yaml:"physics"`
	Tools []string `yaml:"tools"`
}

// ruleWorldYAMLParses checks that the world config file is well-formed
// and has the physics.constants block.
func ruleWorldYAMLParses(in Input) []Issue {
	if !in.WorldExists {
		return nil
	}
	data, err := os.ReadFile(in.WorldPath)
	if err != nil {
		return []Issue{{
			Level:   LevelError,
			Path:    relPath(in.Root, in.WorldPath),
			Message: fmt.Sprintf("cannot read world config: %v", err),
		}}
	}
	var parsed worldYAML
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		return []Issue{{
			Level:   LevelError,
			Path:    relPath(in.Root, in.WorldPath),
			Message: "world config is not valid YAML: " + err.Error(),
		}}
	}

	var out []Issue
	if parsed.Physics.Constants.CPU <= 0 {
		out = append(out, Issue{
			Level:   LevelError,
			Path:    relPath(in.Root, in.WorldPath) + "#physics.constants.cpu",
			Message: "cpu must be a positive integer",
			Hint:    "set physics.constants.cpu to the number of cores (e.g. 2)",
		})
	}
	if parsed.Physics.Constants.Memory == "" {
		out = append(out, Issue{
			Level:   LevelError,
			Path:    relPath(in.Root, in.WorldPath) + "#physics.constants.memory",
			Message: "memory is required",
			Hint:    "set physics.constants.memory (e.g. 1g)",
		})
	}
	return out
}

// loadWorldTools is a helper shared by other rules that need the
// flattened tool list from world.yaml.
func loadWorldTools(worldPath string) ([]string, error) {
	data, err := os.ReadFile(worldPath)
	if err != nil {
		return nil, err
	}
	var parsed worldYAML
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		return nil, err
	}
	return parsed.Tools, nil
}

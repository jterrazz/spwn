package mind

import (
	"fmt"
	"os"
	"path/filepath"

	"spwn.sh/core/foundation"
	"gopkg.in/yaml.v3"
)

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

	// Read team from profile.yaml (if exists).
	profilePath := filepath.Join(dir, "profile.yaml")
	if data, err := os.ReadFile(profilePath); err == nil {
		var p struct {
			Team string `yaml:"team"`
		}
		if yaml.Unmarshal(data, &p) == nil {
			info.Team = p.Team
		}
	}

	for _, layer := range foundation.MindLayers {
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

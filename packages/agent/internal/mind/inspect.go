package mind

import (
	"fmt"
	"path/filepath"
	"spwn.sh/packages/platform"

	"gopkg.in/yaml.v3"
)

// Inspect returns detailed information about an agent's Mind.
func Inspect(name string) (*AgentInfo, error) {
	dir := AgentDir(name)
	files := platform.DataFiles()
	defer files.Close()
	if _, err := files.Stat(dir); err != nil {
		return nil, fmt.Errorf("agent %q not found", name)
	}

	info := &AgentInfo{
		Name:   name,
		Path:   dir,
		Layers: make(map[string][]string),
	}

	manifestPath := filepath.Join(dir, "agent.yaml")
	if data, err := files.ReadFile(manifestPath); err == nil {
		var p struct {
			Team string `yaml:"team"`
		}
		if yaml.Unmarshal(data, &p) == nil {
			info.Team = p.Team
		}
	}

	for _, layer := range platform.MindLayers {
		layerDir := filepath.Join(dir, layer)
		entries, err := files.ReadDir(layerDir)
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

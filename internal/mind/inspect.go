package mind

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jterrazz/universe/internal/config"
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

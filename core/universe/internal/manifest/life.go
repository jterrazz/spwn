package manifest

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// LifeManifest declares an agent's identity structure and requirements.
// Optional: if life.yaml doesn't exist in the agent directory, the agent dir is used as-is.
type LifeManifest struct {
	Soul struct {
		Personas []string `yaml:"personas"`
	} `yaml:"soul"`
	Mind struct {
		Skills    []string `yaml:"skills"`
		Knowledge []string `yaml:"knowledge"`
		Playbooks []string `yaml:"playbooks"`
	} `yaml:"mind"`
	Body struct {
		Requires []string `yaml:"requires"`
	} `yaml:"body"`
}

// LoadLife reads life.yaml from an agent directory.
// Returns nil (no error) if life.yaml doesn't exist — it's optional.
func LoadLife(agentDir string) (*LifeManifest, error) {
	path := filepath.Join(agentDir, "life.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read life manifest: %w", err)
	}

	var life LifeManifest
	if err := yaml.Unmarshal(data, &life); err != nil {
		return nil, fmt.Errorf("parse life manifest: %w", err)
	}

	return &life, nil
}

// ValidateBody checks that all elements in body.requires are available in the universe.
// availableElements should be the expanded element list from the universe manifest.
func ValidateBody(life *LifeManifest, availableElements []string) error {
	if life == nil || len(life.Body.Requires) == 0 {
		return nil
	}

	available := make(map[string]bool)
	for _, e := range availableElements {
		available[e] = true
	}

	var missing []string
	for _, req := range life.Body.Requires {
		// Expand @packs to check individual binaries
		expanded := ExpandElements([]string{req})
		for _, e := range expanded {
			if !available[e] {
				missing = append(missing, e)
			}
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("agent requires element(s) not provided by this universe: %s.\nHint: Add them to the universe config's elements, or remove them from life.yaml body.requires",
			strings.Join(missing, ", "))
	}

	return nil
}

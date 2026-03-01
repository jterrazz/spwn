package manifest

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/jterrazz/universe/internal/config"
)

// ElementPacks maps pack names to their expanded binaries.
var ElementPacks = map[string][]string{
	"ubuntu": {"bash", "sh", "ls", "cat", "cp", "mv", "rm", "mkdir", "rmdir", "chmod", "chown", "grep", "sed", "awk", "find", "xargs", "curl", "wget"},
	"node":   {"node", "npm", "npx"},
	"python": {"python3", "pip3"},
	"build":  {"make", "gcc", "g++"},
}

// Load reads a universe.yaml file and returns a UniverseManifest with defaults applied.
func Load(path string) (config.UniverseManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return config.UniverseManifest{}, fmt.Errorf("reading manifest: %w", err)
	}

	var m config.UniverseManifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return config.UniverseManifest{}, fmt.Errorf("parsing manifest: %w", err)
	}

	ApplyDefaults(&m)
	return m, nil
}

// Discover searches for universe.yaml in the given directories (in order) and returns
// the first one found. Returns an empty manifest with defaults if none found.
func Discover(dirs ...string) (config.UniverseManifest, string, error) {
	for _, dir := range dirs {
		path := filepath.Join(dir, "universe.yaml")
		if _, err := os.Stat(path); err == nil {
			m, err := Load(path)
			if err != nil {
				return config.UniverseManifest{}, "", err
			}
			return m, path, nil
		}
	}

	// No manifest found — return defaults.
	m := config.UniverseManifest{}
	ApplyDefaults(&m)
	return m, "", nil
}

// ApplyDefaults fills in zero-value fields with built-in defaults.
func ApplyDefaults(m *config.UniverseManifest) {
	if m.Physics.Origin == "" {
		m.Physics.Origin = config.DefaultOrigin
	}
	if m.Physics.Constants.CPU == 0 {
		m.Physics.Constants.CPU = config.DefaultCPU
	}
	if m.Physics.Constants.Memory == "" {
		m.Physics.Constants.Memory = config.DefaultMemory
	}
	if m.Physics.Constants.Disk == "" {
		m.Physics.Constants.Disk = config.DefaultDisk
	}
	if m.Physics.Constants.Timeout == "" {
		m.Physics.Constants.Timeout = config.DefaultTimeout
	}
	if m.Physics.Laws.Network == "" {
		m.Physics.Laws.Network = config.DefaultNetwork
	}
	if m.Physics.Laws.MaxProcesses == 0 {
		m.Physics.Laws.MaxProcesses = config.DefaultMaxProcs
	}
}

// MergeFlags overrides manifest values with CLI flag values.
func MergeFlags(m *config.UniverseManifest, origin string) {
	if origin != "" {
		m.Physics.Origin = origin
	}
}

// ExpandElements expands any pack names in the elements list into individual binaries
// and deduplicates the result.
func ExpandElements(elements []string) []string {
	seen := make(map[string]bool)
	var expanded []string

	for _, elem := range elements {
		if pack, ok := ElementPacks[elem]; ok {
			for _, bin := range pack {
				if !seen[bin] {
					seen[bin] = true
					expanded = append(expanded, bin)
				}
			}
		} else {
			if !seen[elem] {
				seen[elem] = true
				expanded = append(expanded, elem)
			}
		}
	}

	return expanded
}

// Validate checks that a manifest is valid for spawning.
func Validate(m *config.UniverseManifest) error {
	if m.Physics.Origin == "" {
		return fmt.Errorf("physics.origin is required")
	}

	validNetworks := map[string]bool{"none": true, "bridge": true, "host": true}
	if !validNetworks[m.Physics.Laws.Network] {
		return fmt.Errorf("physics.laws.network must be one of: none, bridge, host (got %q)", m.Physics.Laws.Network)
	}

	if m.Physics.Constants.CPU < 0 {
		return fmt.Errorf("physics.constants.cpu must be positive")
	}

	return nil
}

package manifest

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jterrazz/universe/internal/config"
	"gopkg.in/yaml.v3"
)

// TechnologyPacks maps @pack names to their constituent binaries.
var TechnologyPacks = map[string][]string{
	"@unix":   {"bash", "sh", "ls", "cat", "cp", "mv", "rm", "mkdir", "rmdir", "chmod", "chown", "grep", "sed", "awk", "find", "xargs", "curl", "wget"},
	"@git":    {"git"},
	"@node":   {"node", "npm", "npx"},
	"@python": {"python3", "pip3"},
	"@build":  {"make", "gcc", "g++"},
}

// rawManifest is the intermediate YAML structure before conversion to UniverseManifest.
type rawManifest struct {
	Physics struct {
		Constants config.ConstantsManifest `yaml:"constants"`
		Laws      config.LawsManifest      `yaml:"laws"`
		Elements  []map[string]string      `yaml:"elements"`
	} `yaml:"physics"`
	Technologies yaml.Node `yaml:"technologies"`
}

// Load reads a named universe config from ~/.universe/universes/{name}.yaml.
func Load(name string) (config.UniverseManifest, error) {
	path := filepath.Join(config.UniversesDir(), name+".yaml")
	return LoadPath(path)
}

// LoadPath reads a universe config from an explicit file path.
func LoadPath(path string) (config.UniverseManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return config.UniverseManifest{}, fmt.Errorf("read config %s: %w", path, err)
	}

	var raw rawManifest
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return config.UniverseManifest{}, fmt.Errorf("parse config %s: %w", path, err)
	}

	m := config.UniverseManifest{
		Physics: config.PhysicsManifest{
			Constants: raw.Physics.Constants,
			Laws:      raw.Physics.Laws,
		},
	}

	// Parse elements
	for _, elem := range raw.Physics.Elements {
		for name, path := range elem {
			m.Physics.Elements = append(m.Physics.Elements, config.ElementMount{Name: name, Path: path})
		}
	}

	// Parse technologies (mixed list of strings + gate subsection)
	if raw.Technologies.Kind == yaml.MappingNode || raw.Technologies.Kind == yaml.SequenceNode {
		m.Technologies, m.Gate = parseTechnologies(&raw.Technologies)
	}

	ApplyDefaults(&m)
	return m, nil
}

// parseTechnologies handles the mixed technologies YAML:
//
//	technologies:
//	  - @unix
//	  - @git
//	  - jq
//	  gate:
//	    - source: mcp/slack
//	      as: slack-send
//	      capabilities: [send]
func parseTechnologies(node *yaml.Node) ([]string, []config.GateBridge) {
	var techs []string
	var gates []config.GateBridge

	if node.Kind == yaml.SequenceNode {
		for _, item := range node.Content {
			if item.Kind == yaml.ScalarNode {
				techs = append(techs, item.Value)
			}
		}
	} else if node.Kind == yaml.MappingNode {
		for i := 0; i < len(node.Content)-1; i += 2 {
			key := node.Content[i]
			val := node.Content[i+1]
			if key.Value == "gate" && val.Kind == yaml.SequenceNode {
				for _, gateItem := range val.Content {
					var gb config.GateBridge
					gateItem.Decode(&gb)
					gates = append(gates, gb)
				}
			}
		}
	}

	return techs, gates
}

// ListConfigs returns the names of all universe configs in ~/.universe/universes/.
func ListConfigs() ([]string, error) {
	dir := config.UniversesDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".yaml") {
			names = append(names, strings.TrimSuffix(e.Name(), ".yaml"))
		}
	}
	return names, nil
}

// CreateDefault creates a default.yaml in ~/.universe/universes/.
func CreateDefault() error {
	dir := config.UniversesDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	path := filepath.Join(dir, "default.yaml")
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("default.yaml already exists")
	}

	content := `# Default universe config
# Edit to change default behavior. See: universe config inspect default

physics:
  constants:
    cpu: 1
    memory: 512m
    disk: 2g
    timeout: 30m

  laws:
    network: none
    max-processes: 128

technologies:
  - @unix
  - @git
`
	return os.WriteFile(path, []byte(content), 0644)
}

// CreateConfig scaffolds a new named config.
func CreateConfig(name string) error {
	dir := config.UniversesDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	path := filepath.Join(dir, name+".yaml")
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("config %q already exists", name)
	}

	content := fmt.Sprintf(`# Universe config: %s

physics:
  constants:
    cpu: 1
    memory: 512m
    disk: 2g
    timeout: 30m

  laws:
    network: none
    max-processes: 128

technologies:
  - @unix
  - @git
`, name)
	return os.WriteFile(path, []byte(content), 0644)
}

// ExpandTechnologies expands @packs into individual binaries and deduplicates.
func ExpandTechnologies(techs []string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, t := range techs {
		if binaries, ok := TechnologyPacks[t]; ok {
			for _, b := range binaries {
				if !seen[b] {
					seen[b] = true
					result = append(result, b)
				}
			}
		} else if !seen[t] {
			seen[t] = true
			result = append(result, t)
		}
	}
	return result
}

// ApplyDefaults fills zero-value fields with built-in defaults.
func ApplyDefaults(m *config.UniverseManifest) {
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

// Validate checks that a manifest is well-formed.
func Validate(m config.UniverseManifest) error {
	switch m.Physics.Laws.Network {
	case "none", "bridge", "host":
	default:
		return fmt.Errorf("invalid network law %q (must be none, bridge, or host)", m.Physics.Laws.Network)
	}
	if m.Physics.Constants.CPU < 0 {
		return fmt.Errorf("CPU must be >= 0")
	}
	return nil
}

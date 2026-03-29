package manifest

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jterrazz/spwn/core/gate"
	"github.com/jterrazz/spwn/core/universe/internal/models"
	"github.com/jterrazz/spwn/core/foundation"
	"gopkg.in/yaml.v3"
)

// ElementPacks maps @pack names to their constituent binaries.
var ElementPacks = map[string][]string{
	"@unix":   {"bash", "sh", "ls", "cat", "cp", "mv", "rm", "mkdir", "rmdir", "chmod", "chown", "grep", "sed", "awk", "find", "xargs", "curl", "wget"},
	"@git":    {"git"},
	"@node":   {"node", "npm", "npx"},
	"@python": {"python3", "pip3"},
	"@build":  {"make", "gcc", "g++"},
}

// rawManifest is the intermediate YAML structure before conversion to UniverseManifest.
type rawManifest struct {
	Physics struct {
		Constants models.ConstantsManifest `yaml:"constants"`
		Laws      models.LawsManifest      `yaml:"laws"`
		Elements  yaml.Node                `yaml:"elements"`
	} `yaml:"physics"`
	Gate []gate.Bridge `yaml:"gate"`
}

// Load reads a named universe config from ~/.spwn/universes/{name}.yaml.
func Load(name string) (models.Manifest, error) {
	path := filepath.Join(foundation.UniversesDir(), name+".yaml")
	return LoadPath(path)
}

// LoadPath reads a universe config from an explicit file path.
func LoadPath(path string) (models.Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return models.Manifest{}, fmt.Errorf("read config %s: %w", path, err)
	}

	var raw rawManifest
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return models.Manifest{}, fmt.Errorf("parse config %s: %w", path, err)
	}

	m := models.Manifest{
		Physics: models.PhysicsManifest{
			Constants: raw.Physics.Constants,
			Laws:      raw.Physics.Laws,
		},
		Gate: raw.Gate,
	}

	// Parse elements (plain list of strings)
	if raw.Physics.Elements.Kind == yaml.SequenceNode {
		m.Elements = parseElements(&raw.Physics.Elements)
	}

	ApplyDefaults(&m)
	return m, nil
}

// parseElements extracts element names from a YAML sequence node.
func parseElements(node *yaml.Node) []string {
	var elems []string
	for _, item := range node.Content {
		if item.Kind == yaml.ScalarNode {
			elems = append(elems, item.Value)
		}
	}
	return elems
}

// ListConfigs returns the names of all universe configs in ~/.spwn/universes/.
func ListConfigs() ([]string, error) {
	dir := foundation.UniversesDir()
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

// CreateDefault creates a default.yaml in ~/.spwn/universes/.
func CreateDefault() error {
	dir := foundation.UniversesDir()
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

  elements:
    - "@unix"
    - "@git"
`
	return os.WriteFile(path, []byte(content), 0644)
}

// CreateConfig scaffolds a new named config.
func CreateConfig(name string) error {
	dir := foundation.UniversesDir()
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

  elements:
    - "@unix"
    - "@git"
`, name)
	return os.WriteFile(path, []byte(content), 0644)
}

// ExpandElements expands @packs into individual binaries and deduplicates.
func ExpandElements(elems []string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, e := range elems {
		if binaries, ok := ElementPacks[e]; ok {
			for _, b := range binaries {
				if !seen[b] {
					seen[b] = true
					result = append(result, b)
				}
			}
		} else if !seen[e] {
			seen[e] = true
			result = append(result, e)
		}
	}
	return result
}

// ApplyDefaults fills zero-value fields with built-in defaults.
func ApplyDefaults(m *models.Manifest) {
	if m.Physics.Constants.CPU == 0 {
		m.Physics.Constants.CPU = foundation.DefaultCPU
	}
	if m.Physics.Constants.Memory == "" {
		m.Physics.Constants.Memory = foundation.DefaultMemory
	}
	if m.Physics.Constants.Disk == "" {
		m.Physics.Constants.Disk = foundation.DefaultDisk
	}
	if m.Physics.Constants.Timeout == "" {
		m.Physics.Constants.Timeout = foundation.DefaultTimeout
	}
	if m.Physics.Laws.Network == "" {
		m.Physics.Laws.Network = foundation.DefaultNetwork
	}
	if m.Physics.Laws.MaxProcesses == 0 {
		m.Physics.Laws.MaxProcesses = foundation.DefaultMaxProcs
	}
}

// Validate checks that a manifest is well-formed.
func Validate(m models.Manifest) error {
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

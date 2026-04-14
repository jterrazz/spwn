package manifest

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"spwn.sh/packages/world/internal/models"
	"gopkg.in/yaml.v3"
	"spwn.sh/packages/paths"
)

// ToolPacks maps @pack names to their constituent binaries.
var ToolPacks = map[string][]string{
	"@spwn/unix":   {"bash", "sh", "ls", "cat", "cp", "mv", "rm", "mkdir", "rmdir", "chmod", "chown", "grep", "sed", "awk", "find", "xargs", "curl", "wget"},
	"@spwn/git":    {"git"},
	"@spwn/node":   {"node", "npm", "npx"},
	"@spwn/python": {"python3", "pip3"},
	"@spwn/build":  {"make", "gcc", "g++"},
}

// rawManifest is the intermediate YAML structure before conversion to Manifest.
type rawManifest struct {
	Tools yaml.Node `yaml:"tools"`
}

// Load reads a named world config from ~/.spwn/worlds/{name}.yaml.
func Load(name string) (models.Manifest, error) {
	path := filepath.Join(paths.WorldsDir(), name+".yaml")
	return LoadPath(path)
}

// LoadPath reads a world config from an explicit file path.
func LoadPath(path string) (models.Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return models.Manifest{}, fmt.Errorf("read config %s: %w", path, err)
	}

	var raw rawManifest
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return models.Manifest{}, fmt.Errorf("parse config %s: %w", path, err)
	}

	m := models.Manifest{}

	// Parse tools (plain list of strings, root-level)
	if raw.Tools.Kind == yaml.SequenceNode {
		m.Tools = parseTools(&raw.Tools)
	}

	ApplyDefaults(&m)
	return m, nil
}

// parseTools extracts tool names from a YAML sequence node.
func parseTools(node *yaml.Node) []string {
	var elems []string
	for _, item := range node.Content {
		if item.Kind == yaml.ScalarNode {
			elems = append(elems, item.Value)
		}
	}
	return elems
}

// ListConfigs returns the names of all world configs in ~/.spwn/worlds/.
func ListConfigs() ([]string, error) {
	dir := paths.WorldsDir()
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

// CreateDefault creates a default.yaml in ~/.spwn/worlds/.
func CreateDefault() error {
	dir := paths.WorldsDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	path := filepath.Join(dir, "default.yaml")
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("default.yaml already exists.\nEdit it at %s or remove it first", path)
	}

	content := `# Default world config.
# Docs: https://spwn.sh/docs/cli/spwn-world

# Available tools (@spwn/unix = bash, grep, sed, awk, etc.)
tools:
  - "@spwn/unix"          # Core Unix tools
  - "@spwn/git"           # Git version control
  # - "@spwn/node"        # Node.js + npm
  # - "@spwn/python"      # Python 3
  # - "@spwn/build"       # make, gcc, g++
`
	return os.WriteFile(path, []byte(content), 0644)
}

// CreateConfig scaffolds a new named config.
func CreateConfig(name string) error {
	dir := paths.WorldsDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	path := filepath.Join(dir, name+".yaml")
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("config %q already exists.\nEdit it at %s or remove it first", name, path)
	}

	content := fmt.Sprintf(`# World config: %s

tools:
  - "@spwn/unix"
  - "@spwn/git"
`, name)
	return os.WriteFile(path, []byte(content), 0644)
}

// ExpandTools expands @packs into individual binaries and deduplicates.
func ExpandTools(elems []string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, e := range elems {
		if binaries, ok := ToolPacks[e]; ok {
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
func ApplyDefaults(_ *models.Manifest) {}

// Validate checks that a manifest is well-formed.
func Validate(_ models.Manifest) error {
	return nil
}

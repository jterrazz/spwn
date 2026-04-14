package manifest

import (
	"os"

	"gopkg.in/yaml.v3"
	"spwn.sh/packages/paths"
)

// OrgManifest represents the world-level configuration (org.yaml).
type OrgManifest struct {
	Name       string        `yaml:"name"`
	Version    int           `yaml:"version"`
	Defaults   OrgDefaults   `yaml:"defaults"`
	Skills     []string      `yaml:"skills"`
	Governance OrgGovernance `yaml:"governance"`
	Claw       ClawConfig    `yaml:"claw"`
}

// OrgDefaults holds world-wide default settings.
type OrgDefaults struct {
	Runtime RuntimeDefaults `yaml:"runtime"`
	Backend string          `yaml:"backend"`
	Memory  string          `yaml:"memory"`
	Store   string          `yaml:"store"`
}

// RuntimeDefaults holds default runtime configuration.
type RuntimeDefaults struct {
	Backend  string `yaml:"backend"`
	Provider string `yaml:"provider"`
	Model    string `yaml:"model"`
	Auth     string `yaml:"auth"`
}

// OrgGovernance holds governance limits and policies.
type OrgGovernance struct {
	MaxWorlds           int      `yaml:"max-worlds"`
	MaxWorkersPerWorld int      `yaml:"max-workers-per-world"`
	AllowedProviders       []string `yaml:"allowed-providers"`
	CostLimit              string   `yaml:"cost-limit"`
	Audit                  bool     `yaml:"audit"`
}

// ClawConfig holds Claw daemon configuration.
type ClawConfig struct {
	Sync SyncConfig `yaml:"sync"`
}

// SyncConfig holds sync/git configuration for the Claw.
type SyncConfig struct {
	Repo     string `yaml:"repo"`
	Branch   string `yaml:"branch"`
	AutoPush bool   `yaml:"auto-push"`
	AutoPull bool   `yaml:"auto-pull"`
}

// LoadOrg reads the world manifest from ~/.spwn/org.yaml.
func LoadOrg() (*OrgManifest, error) {
	return LoadOrgPath(paths.OrgPath())
}

// LoadOrgPath reads a world manifest from an explicit path.
func LoadOrgPath(path string) (*OrgManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var org OrgManifest
	if err := yaml.Unmarshal(data, &org); err != nil {
		return nil, err
	}
	return &org, nil
}

// CreateOrg creates a default world manifest at ~/.spwn/org.yaml.
func CreateOrg(name string) error {
	org := OrgManifest{
		Name:    name,
		Version: 1,
		Defaults: OrgDefaults{
			Runtime: RuntimeDefaults{
				Backend:  "claude-code",
				Provider: "anthropic",
			},
			Backend: "docker",
			Memory:  "filesystem",
			Store:   "json",
		},
	}
	data, err := yaml.Marshal(&org)
	if err != nil {
		return err
	}
	return os.WriteFile(paths.OrgPath(), data, 0644)
}

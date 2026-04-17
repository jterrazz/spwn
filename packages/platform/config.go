package platform

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ConfigFileName is the user-level config filename under ~/.spwn/.
const ConfigFileName = "config.yaml"

// CurrentConfigAPIVersion is the schema version emitted by Save and
// required by Load. Bump when adding breaking fields; migrations
// handle upgrades.
const CurrentConfigAPIVersion = "spwn/v2"

// Config is the parsed ~/.spwn/config.yaml.
//
// Cascade: env vars override file values; a project spwn.yaml can
// override runtime defaults on top. Unset fields mean "use the
// built-in default" — callers should read via the accessor methods
// on this struct so zero-values resolve consistently.
type Config struct {
	// APIVersion is the config schema version. Must match
	// CurrentConfigAPIVersion. Scheme mirrors the k8s apiVersion
	// convention so the schema version is obviously NOT a semver of
	// the user's preferences.
	APIVersion string `yaml:"apiVersion"`

	// Runtime captures defaults for `runtime.backend` / provider / model.
	// Each agent's agent.yaml can override per-agent; this is what a
	// freshly-scaffolded agent starts with.
	Runtime RuntimeConfig `yaml:"runtime,omitempty"`

	// Telemetry gates usage reporting. Opt-in, not opt-out — enabled
	// only when the user explicitly sets Enabled: true.
	Telemetry TelemetryConfig `yaml:"telemetry,omitempty"`

	// Update controls self-update behavior (CLI update channel +
	// version-check cadence).
	Update UpdateConfig `yaml:"update,omitempty"`

	// Onboarded records whether the user has completed the first-run
	// walkthrough. Replaces the legacy `.onboarding-complete` marker
	// file going forward; the marker file is still checked for
	// backward-compat on existing installs.
	Onboarded bool `yaml:"onboarded,omitempty"`
}

// RuntimeConfig captures user-level runtime defaults.
type RuntimeConfig struct {
	// DefaultBackend is the runtime adapter used when an agent
	// doesn't declare one (e.g. "spwn:claude-code").
	DefaultBackend string `yaml:"default_backend,omitempty"`
	// DefaultProvider is the model provider (e.g. "anthropic").
	DefaultProvider string `yaml:"default_provider,omitempty"`
	// DefaultModel is the specific model id (e.g. "claude-4-7-sonnet").
	DefaultModel string `yaml:"default_model,omitempty"`
}

// TelemetryConfig controls usage reporting.
type TelemetryConfig struct {
	Enabled bool `yaml:"enabled"`
}

// UpdateConfig controls self-update behavior.
type UpdateConfig struct {
	// Channel is "stable" or "edge". Defaults to stable when unset.
	Channel string `yaml:"channel,omitempty"`
}

// DefaultConfig returns the baseline config written on first run.
func DefaultConfig() Config {
	return Config{
		APIVersion: CurrentConfigAPIVersion,
		Runtime: RuntimeConfig{
			DefaultBackend: "spwn:claude-code",
		},
		Telemetry: TelemetryConfig{Enabled: false},
		Update:    UpdateConfig{Channel: "stable"},
		Onboarded: false,
	}
}

// ConfigPath returns the absolute path to the user config file,
// honouring SPWN_HOME.
func ConfigPath() string {
	return filepath.Join(BaseDir(), ConfigFileName)
}

// LoadConfig reads ~/.spwn/config.yaml. Returns the default config
// (not an error) when the file is missing so first-run callers can
// proceed unchanged. Malformed YAML IS an error — silent fallback
// would hide user edits.
func LoadConfig() (Config, error) {
	path := ConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return Config{}, fmt.Errorf("read %s: %w", path, err)
	}
	var c Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		return Config{}, fmt.Errorf("parse %s: %w", path, err)
	}
	if c.APIVersion == "" {
		c.APIVersion = CurrentConfigAPIVersion
	}
	return c, nil
}

// SaveConfig writes the config deterministically (stable key order,
// no timestamp). Creates ~/.spwn/ if missing.
func SaveConfig(c Config) error {
	if c.APIVersion == "" {
		c.APIVersion = CurrentConfigAPIVersion
	}
	path := ConfigPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(path), err)
	}
	data, err := yaml.Marshal(&c)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	header := []byte("# spwn user config — edit freely; spwn re-reads on every CLI invocation.\n" +
		"# Docs: https://spwn.sh/docs/config\n")
	if err := os.WriteFile(path, append(header, data...), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

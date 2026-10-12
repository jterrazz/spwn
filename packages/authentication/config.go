package authentication

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"

	"spwn.sh/packages/platform"
)

// Method is the user-facing style of credential for a provider:
// OAuth (subscription-flavored, short-lived) or API key (long-lived
// secret). Sources like "keychain" or "env var" are different ways of
// delivering one of these two methods — callers who want to pick
// between an OAuth token and an API key talk in these terms.
//
// Empty Method means "auto-select" — the resolver falls back to the
// per-provider discovery order. This is the default for users who
// never ran `spwn auth use`.
type Method string

const (
	MethodOAuth  Method = "oauth"
	MethodAPIKey Method = "api_key"
)

// ProviderPref captures the user's intent for one provider. Persisted
// in auth.yaml so choices survive across machines (if the file is
// shared) and across spwn upgrades. Everything here is user-writable;
// the resolver reads this before picking a credential.
type ProviderPref struct {
	// Method is the preferred credential method for this provider.
	// Empty means "use whichever the detection order finds first".
	Method Method `yaml:"method,omitempty"`
	// Disabled=true tells the resolver to act as if this provider has
	// no credentials, even if env vars or keychain entries are present.
	// Useful to opt a whole provider out without deleting its creds.
	Disabled bool `yaml:"disabled,omitempty"`
}

// Config is the root of auth.yaml. Version is written so future
// Format changes can migrate in-place.
type Config struct {
	Version int `yaml:"version"`
	// DefaultProvider names the provider spwn uses when multiple are
	// Authenticated and no runtime is pinned at the project/agent
	// Level. Empty means "no preference" — the resolver then either
	// Silently picks the only connected provider or errors with a
	// Disambiguation hint when more than one is present. Soft
	// Preference only; disabled providers are skipped regardless.
	DefaultProvider Provider                  `yaml:"default_provider,omitempty"`
	Providers       map[Provider]ProviderPref `yaml:"providers,omitempty"`
}

const currentConfigVersion = 1

var configMu sync.Mutex

func configPath() string {
	return filepath.Join(platform.UserDir(), "auth.yaml")
}

// LoadConfig reads auth.yaml and returns a validated Config. A missing
// file is not an error — callers get an empty (but initialised) config
// and can proceed. Once loaded, legacy `.disabled-<provider>` marker
// files in CredentialsDir() are folded into the returned Config and
// also persisted back so subsequent reads don't need to check them.
func LoadConfig() *Config {
	configMu.Lock()
	defer configMu.Unlock()
	c, migrated := loadLocked()
	if migrated {
		_ = saveLocked(c)
	}
	return c
}

func loadLocked() (*Config, bool) {
	c := &Config{Version: currentConfigVersion, Providers: map[Provider]ProviderPref{}}

	data, err := os.ReadFile(configPath())
	if err == nil {
		_ = yaml.Unmarshal(data, c)
		if c.Providers == nil {
			c.Providers = map[Provider]ProviderPref{}
		}
		if c.Version == 0 {
			c.Version = currentConfigVersion
		}
	}

	migrated := migrateDisabledMarkers(c)

	return c, migrated
}

func migrateDisabledMarkers(c *Config) bool {
	dir := platform.CredentialsDir()
	changed := false
	for _, p := range []Provider{ProviderAnthropic, ProviderOpenAI, ProviderGoogle} {
		marker := filepath.Join(dir, ".disabled-"+string(p))
		if _, err := os.Stat(marker); err != nil {
			continue
		}
		pref := c.Providers[p]
		if !pref.Disabled {
			pref.Disabled = true
			c.Providers[p] = pref
			changed = true
		}
		_ = os.Remove(marker)
	}
	return changed
}

// SaveConfig persists a Config to auth.yaml atomically (tmp + rename)
// so a crash mid-write can't leave a corrupt file. Creates the parent
// directory when needed.
func SaveConfig(c *Config) error {
	configMu.Lock()
	defer configMu.Unlock()
	return saveLocked(c)
}

func saveLocked(c *Config) error {
	if c == nil {
		return fmt.Errorf("nil config")
	}
	if c.Version == 0 {
		c.Version = currentConfigVersion
	}
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	path := configPath()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Pref returns the ProviderPref for a provider. Always returns a
// valid value; absent entries read as the zero value (auto-method,
// not disabled).
func (c *Config) Pref(p Provider) ProviderPref {
	if c == nil || c.Providers == nil {
		return ProviderPref{}
	}
	return c.Providers[p]
}

// SetPref replaces the ProviderPref for a provider. Zero-valued prefs
// are written as empty-map entries rather than dropped so a future
// reader can distinguish "never touched" from "explicitly reset" if
// needed. Callers then SaveConfig to persist.
func (c *Config) SetPref(p Provider, pref ProviderPref) {
	if c.Providers == nil {
		c.Providers = map[Provider]ProviderPref{}
	}
	c.Providers[p] = pref
}

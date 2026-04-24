package auth

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

// currentConfigVersion is bumped whenever the YAML shape changes in a
// way that requires migration on load.
const currentConfigVersion = 1

// configMu serialises read-modify-write cycles so `spwn auth disable`
// racing with `spwn auth use` (or with a parallel test) doesn't drop
// one of the updates. The file is tiny; holding the lock for I/O is
// cheap and simpler than optimistic retries.
var configMu sync.Mutex

// configPath returns the absolute path to the user's auth config. It
// honours SPWN_HOME via platform.UserDir so tests and container-side
// callers see the right file.
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

// loadLocked does the read + migration without taking configMu (caller
// holds it). Returns (cfg, migrated) where migrated=true means the
// caller should persist the result to durably absorb legacy state.
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

	// Absorb legacy `.disabled-<provider>` marker files. These were
	// written by older `DisableProvider` impls; treat them as the
	// authoritative source on first read, then rely on YAML forever
	// after.
	migrated := migrateDisabledMarkers(c)

	return c, migrated
}

// migrateDisabledMarkers folds legacy `.disabled-<provider>` files
// under CredentialsDir() into the Config and deletes the marker files.
// Returns true when anything changed (so the caller persists).
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

// saveLocked is the non-locking body of SaveConfig. Callers inside
// this file hold configMu themselves to chain a load + mutate + save
// atomically.
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

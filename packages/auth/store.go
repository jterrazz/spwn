package auth

import (
	"os"
	"path/filepath"
	"strings"

	"spwn.sh/packages/platform"
)

// SaveToken persists a token to the auth cache file.
func SaveToken(token string) error {
	path := tokenPath()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(strings.TrimSpace(token)), 0600)
}

// ClearToken removes the cached token.
func ClearToken() error {
	return os.Remove(tokenPath())
}

// DisableProvider marks a provider as disabled — the resolver will
// behave as though it had no credentials. Backed by the auth.yaml
// user config so the choice survives across spwn upgrades and is
// visible to any tool that reads the same file.
func DisableProvider(p Provider) error {
	return mutateConfig(func(c *Config) {
		pref := c.Pref(p)
		pref.Disabled = true
		c.SetPref(p, pref)
	})
}

// EnableProvider re-enables a previously-disabled provider. Idempotent
// — calling it on an already-enabled provider is a cheap no-op aside
// from the file rewrite.
func EnableProvider(p Provider) error {
	return mutateConfig(func(c *Config) {
		pref := c.Pref(p)
		pref.Disabled = false
		c.SetPref(p, pref)
	})
}

// IsProviderDisabled reports whether the user has opted this provider
// out. Reads through LoadConfig, which transparently migrates legacy
// `.disabled-<provider>` marker files on first call.
func IsProviderDisabled(p Provider) bool {
	return LoadConfig().Pref(p).Disabled
}

// mutateConfig runs fn against the current Config under configMu, then
// persists the result. Keeps concurrent DisableProvider / SetActiveMethod
// callers from clobbering each other's writes.
func mutateConfig(fn func(*Config)) error {
	configMu.Lock()
	defer configMu.Unlock()
	c, _ := loadLocked()
	fn(c)
	return saveLocked(c)
}

// SetActiveMethod records the user's preferred credential method for a
// provider. Empty Method reverts to auto-selection. Persists via the
// same auth.yaml file as DisableProvider.
func SetActiveMethod(p Provider, m Method) error {
	return mutateConfig(func(c *Config) {
		pref := c.Pref(p)
		pref.Method = m
		c.SetPref(p, pref)
	})
}

// ActiveMethod reports the user's preferred credential method for a
// provider, or the empty Method when none was chosen (auto-select).
func ActiveMethod(p Provider) Method {
	return LoadConfig().Pref(p).Method
}

// SetDefaultProvider records the user's preferred provider for when
// Multiple are authenticated simultaneously. Pass the empty Provider
// To clear the preference. Persists via auth.yaml.
//
// Soft preference only: it does not disable any other provider or
// Override an agent.yaml / spwn.yaml runtime pin. Consumed by the
// Runtime resolver in the CLI layer to break ambiguity ties silently
// Instead of erroring.
func SetDefaultProvider(p Provider) error {
	return mutateConfig(func(c *Config) {
		c.DefaultProvider = p
	})
}

// DefaultProvider returns the user's preferred provider when multiple
// Are authenticated, or the empty Provider when none was chosen.
func DefaultProvider() Provider {
	return LoadConfig().DefaultProvider
}

// ReadCachedToken reads the cached token from disk.
func ReadCachedToken() string {
	data, err := os.ReadFile(tokenPath())
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func tokenPath() string {
	return platform.BaseDir() + "/.auth-token"
}

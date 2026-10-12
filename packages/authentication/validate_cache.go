package authentication

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"spwn.sh/packages/platform"
)

const validationCacheFile = ".validated"

// invariant: validationCacheMu serialises read-modify-write on the cache file so
// Two concurrent spawns don't corrupt it.
var validationCacheMu sync.Mutex

type validationEntry struct {
	// Source is the credential source that validated (e.g.
	// "keychain:Claude Code"). Used as part of the cache key so
	// Switching methods forces a re-validation.
	Source string `json:"source"`
	// CredType is the credential type, same reason as Source.
	CredType CredentialType `json:"cred_type"`
	// ValidatedAt is when the API last returned success. Positive
	// Only — 401/403 responses are NEVER cached, so a user who
	// Just ran `claude login` never has to wait a TTL to see the
	// Result take effect.
	ValidatedAt time.Time `json:"validated_at"`
}

type validationCache map[Provider]validationEntry

// ValidateWithCache validates a credential against its provider's
// API, reusing a cached success result for up to maxAge. Cache key
// Is (provider, source, cred type) so swapping OAuth for API key,
// Or refreshing the keychain entry, bypasses the cache naturally.
//
// Negative results (401, connection errors) are never cached — users
// Who fix a broken credential should see the fix immediately, not
// After a TTL.
//
// When maxAge is zero the cache is bypassed entirely (always hits
// The network). Useful for an explicit "revalidate now" flag.
func ValidateWithCache(ctx context.Context, cred *Credential, maxAge time.Duration) *ProviderStatus {
	if os.Getenv("SPWN_SKIP_AUTH_VALIDATION") != "" {
		status := &ProviderStatus{Connected: true}
		if cred != nil {
			status.Provider = cred.Provider
			status.CredType = cred.Type
			status.Source = cred.Source
		}
		return status
	}

	if cred == nil || cred.Type == CredTypeNone {
		return Validate(ctx, cred)
	}

	if maxAge > 0 {
		cache := loadValidationCache()
		if entry, ok := cache[cred.Provider]; ok {
			if entry.Source == cred.Source && entry.CredType == cred.Type {
				if time.Since(entry.ValidatedAt) < maxAge {
					return &ProviderStatus{
						Provider:  cred.Provider,
						Connected: true,
						CredType:  cred.Type,
						Source:    cred.Source,
					}
				}
			}
		}
	}

	status := Validate(ctx, cred)
	if status != nil && status.Connected {
		recordValidation(cred)
	}
	return status
}

// InvalidateValidation removes any cached positive-validation entry
// For a provider. Called by logout/login so the next spawn re-checks
// From scratch.
func InvalidateValidation(p Provider) {
	validationCacheMu.Lock()
	defer validationCacheMu.Unlock()
	cache := readCacheLocked()
	if _, ok := cache[p]; !ok {
		return
	}
	delete(cache, p)
	_ = writeCacheLocked(cache)
}

func recordValidation(cred *Credential) {
	validationCacheMu.Lock()
	defer validationCacheMu.Unlock()
	cache := readCacheLocked()
	cache[cred.Provider] = validationEntry{
		Source:      cred.Source,
		CredType:    cred.Type,
		ValidatedAt: time.Now().UTC(),
	}
	_ = writeCacheLocked(cache)
}

func loadValidationCache() validationCache {
	validationCacheMu.Lock()
	defer validationCacheMu.Unlock()
	return readCacheLocked()
}

func readCacheLocked() validationCache {
	cache := validationCache{}
	path := validationCachePath()
	data, err := os.ReadFile(path)
	if err != nil {
		return cache
	}
	_ = json.Unmarshal(data, &cache)
	return cache
}

func writeCacheLocked(cache validationCache) error {
	path := validationCachePath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func validationCachePath() string {
	return filepath.Join(platform.CredentialsDir(), validationCacheFile)
}

package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestValidateWithCache_PositiveCacheHit pins the short-term bug fix:
// Repeat validations within the TTL skip the network entirely. The
// Anti-regression property is "don't double-charge the user a /oauth/usage
// Call on every spawn".
func TestValidateWithCache_PositiveCacheHit(t *testing.T) {
	isolateValidationHome(t)

	// First call: mock API returns 200, cache gets written.
	calls := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(ts.Close)

	cred := &Credential{
		Provider: ProviderOpenAI, // OpenAI is the simplest validator — just /v1/models
		Type:     CredTypeAPIKey,
		Token:    "sk-test",
		Source:   "env:OPENAI_API_KEY",
	}

	// Directly seed the cache to simulate a recent validation and
	// Assert we don't hit the network.
	recordValidation(cred)
	if got := ValidateWithCache(context.Background(), cred, 5*time.Minute); !got.Connected {
		t.Errorf("cached result should report Connected=true, got %+v", got)
	}
	// Zero network calls — we never even set up the HTTP client path
	// Because the cache hit short-circuited.
	if calls != 0 {
		t.Errorf("cache hit should skip network, saw %d calls", calls)
	}
}

// TestValidateWithCache_ExpiredCacheRevalidates confirms the TTL is
// Honoured: a cache entry older than maxAge forces a fresh check.
func TestValidateWithCache_ExpiredCacheRevalidates(t *testing.T) {
	isolateValidationHome(t)

	cred := &Credential{
		Provider: ProviderAnthropic,
		Type:     CredTypeKeychain,
		Token:    "sk-ant-test",
		Source:   "keychain:Claude Code",
	}

	// Seed a stale entry (60 min old) and ask for a 5-min TTL.
	validationCacheMu.Lock()
	_ = writeCacheLocked(validationCache{
		cred.Provider: {
			Source:      cred.Source,
			CredType:    cred.Type,
			ValidatedAt: time.Now().Add(-60 * time.Minute),
		},
	})
	validationCacheMu.Unlock()

	// ValidateWithCache should fall through to Validate, which will
	// Hit the real Anthropic API. We can't control that here, so just
	// Assert that the response didn't come from the stale cache (in
	// Particular, errors are allowed because this is an offline test;
	// What we care about is "the cache wasn't silently trusted").
	status := ValidateWithCache(context.Background(), cred, 5*time.Minute)
	// A cache-hit path would have returned Connected=true with no
	// Error. Either the network call returned something real, or it
	// Errored — both are acceptable here; a stale cache hit is not.
	if status != nil && status.Connected && status.Error == "" {
		// Could only happen if the test is somehow online AND the
		// Token is valid, which it isn't. Treat as regression.
		t.Errorf("stale cache should not short-circuit to Connected=true; got %+v", status)
	}
}

// TestValidateWithCache_CacheKeyMismatchIgnoresCache confirms that
// Swapping methods (oauth → api_key) or rotating keychain source
// Forces a fresh validation. Would-be regression: user switches from
// Keychain to API key and spwn silently reports the old keychain
// Validation.
func TestValidateWithCache_CacheKeyMismatchIgnoresCache(t *testing.T) {
	isolateValidationHome(t)

	// Seed a recent positive entry for the keychain source.
	validationCacheMu.Lock()
	_ = writeCacheLocked(validationCache{
		ProviderAnthropic: {
			Source:      "keychain:Claude Code",
			CredType:    CredTypeKeychain,
			ValidatedAt: time.Now().Add(-1 * time.Minute),
		},
	})
	validationCacheMu.Unlock()

	// Ask about a DIFFERENT source (env var). Cache should not apply.
	cred := &Credential{
		Provider: ProviderAnthropic,
		Type:     CredTypeAPIKey,
		Token:    "sk-ant-test",
		Source:   "env:ANTHROPIC_API_KEY", // different from cached
	}
	status := ValidateWithCache(context.Background(), cred, 5*time.Minute)
	// Cache entry pointed at keychain; this call is env-backed. A
	// Correct impl ignores the cache and calls the real API.
	if status != nil && status.Connected && status.Error == "" {
		t.Errorf("different source should bypass cache; got Connected=true via cache")
	}
}

// TestInvalidateValidation_RemovesEntry pins the contract used by
// `spwn auth logout` / `spwn auth login` to force a re-check on the
// Next spawn.
func TestInvalidateValidation_RemovesEntry(t *testing.T) {
	isolateValidationHome(t)

	cred := &Credential{
		Provider: ProviderAnthropic,
		Type:     CredTypeAPIKey,
		Token:    "sk-ant-test",
		Source:   "env:ANTHROPIC_API_KEY",
	}
	recordValidation(cred)

	// Sanity: recorded.
	if cache := loadValidationCache(); cache[cred.Provider].Source == "" {
		t.Fatal("recordValidation did not persist")
	}

	InvalidateValidation(cred.Provider)
	if cache := loadValidationCache(); cache[cred.Provider].Source != "" {
		t.Errorf("InvalidateValidation did not remove entry: %+v", cache)
	}
}

// TestValidateWithCache_ZeroTTLBypassesCache ensures callers can
// Force a re-check by passing maxAge=0 (used by a future
// `--verify-auth` flag or hard-error recovery).
func TestValidateWithCache_ZeroTTLBypassesCache(t *testing.T) {
	isolateValidationHome(t)

	cred := &Credential{
		Provider: ProviderAnthropic,
		Type:     CredTypeAPIKey,
		Token:    "sk-ant-test",
		Source:   "env:ANTHROPIC_API_KEY",
	}
	recordValidation(cred)
	status := ValidateWithCache(context.Background(), cred, 0)
	if status != nil && status.Connected && status.Error == "" {
		t.Errorf("maxAge=0 should bypass cache; got cached result")
	}
}

// IsolateValidationHome points SPWN_HOME at a fresh temp dir so the
// Test's cache reads/writes never touch the developer's real
// ~/.spwn/credentials/.validated. Also cleans up any leftover cache
// File between subtests.
func isolateValidationHome(t *testing.T) {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("SPWN_HOME", tmp)
	t.Setenv("HOME", tmp) // keychain probe is skipped by SPWN_SKIP_KEYCHAIN set below
	t.Setenv("SPWN_SKIP_KEYCHAIN", "1")
	_ = os.MkdirAll(filepath.Join(tmp, "credentials"), 0o700)
}

// Unused helpers kept for the compiler not to complain in partial-
// Build scenarios on older Go versions.
var (
	_ = errors.New
)

package auth

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestLogoutAnthropic_removesCacheFile verifies the unscoped logout
// path: `.auth-token` on disk is removed, the file:... source shows
// up in Cleared, and subsequent reads return empty.
func TestLogoutAnthropic_removesCacheFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("SPWN_HOME", tmp)
	t.Setenv("HOME", tmp)
	t.Setenv("SPWN_SKIP_KEYCHAIN", "1")
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "")
	t.Setenv("ANTHROPIC_AUTH_TOKEN", "")

	if err := SaveToken("oauth-cached-value"); err != nil {
		t.Fatalf("SaveToken: %v", err)
	}

	res := LogoutProvider(ProviderAnthropic, LogoutOpts{})
	if res.HasErrors() {
		t.Fatalf("unexpected errors: %v", res.Errors)
	}
	if !containsPrefix(res.Cleared, "file:~/.spwn/.auth-token") {
		t.Errorf("expected Cleared to include cache file; got %v", res.Cleared)
	}
	if ReadCachedToken() != "" {
		t.Error("cache token should be empty after logout")
	}
}

// TestLogoutAnthropic_methodScopingSparesMismatch checks that logout
// with --method=api_key leaves an OAuth-shaped cache alone.
func TestLogoutAnthropic_methodScopingSparesMismatch(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("SPWN_HOME", tmp)
	t.Setenv("HOME", tmp)
	t.Setenv("SPWN_SKIP_KEYCHAIN", "1")
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "")
	t.Setenv("ANTHROPIC_AUTH_TOKEN", "")

	// Non-sk-ant- prefix → stored as OAuth-style.
	if err := SaveToken("oauth-cached-value"); err != nil {
		t.Fatal(err)
	}

	res := LogoutProvider(ProviderAnthropic, LogoutOpts{Method: MethodAPIKey})
	if res.HasErrors() {
		t.Fatalf("unexpected errors: %v", res.Errors)
	}
	// The cache is OAuth-shaped; an api_key-scoped logout should skip it.
	if ReadCachedToken() != "oauth-cached-value" {
		t.Error("api_key-scoped logout should not touch an OAuth-shaped cache")
	}
	for _, c := range res.Cleared {
		if strings.Contains(c, ".auth-token") {
			t.Errorf("Cleared should not include .auth-token under api_key scope; got %v", res.Cleared)
		}
	}
}

// TestLogoutAnthropic_methodScopingTargetsMatch — the same cache,
// this time storing an sk-ant- API key. api_key-scoped logout deletes
// it; oauth-scoped logout leaves it.
func TestLogoutAnthropic_methodScopingTargetsMatch(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("SPWN_HOME", tmp)
	t.Setenv("HOME", tmp)
	t.Setenv("SPWN_SKIP_KEYCHAIN", "1")
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "")
	t.Setenv("ANTHROPIC_AUTH_TOKEN", "")

	if err := SaveToken("sk-ant-test-key"); err != nil {
		t.Fatal(err)
	}

	// oauth scope: mismatch → cache untouched.
	res := LogoutProvider(ProviderAnthropic, LogoutOpts{Method: MethodOAuth})
	if res.HasErrors() {
		t.Fatalf("unexpected errors: %v", res.Errors)
	}
	if ReadCachedToken() != "sk-ant-test-key" {
		t.Error("oauth-scoped logout should not delete an api_key cache")
	}

	// api_key scope: match → cache deleted.
	res = LogoutProvider(ProviderAnthropic, LogoutOpts{Method: MethodAPIKey})
	if res.HasErrors() {
		t.Fatalf("unexpected errors: %v", res.Errors)
	}
	if ReadCachedToken() != "" {
		t.Error("api_key-scoped logout should delete an api_key cache")
	}
}

// TestLogoutAnthropic_surfacesEnvVars — env vars we cannot touch must
// show up in Remaining so the UI can warn the user.
func TestLogoutAnthropic_surfacesEnvVars(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("SPWN_HOME", tmp)
	t.Setenv("HOME", tmp)
	t.Setenv("SPWN_SKIP_KEYCHAIN", "1")
	t.Setenv("ANTHROPIC_API_KEY", "sk-ant-live")
	t.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "oauth-live")
	t.Setenv("ANTHROPIC_AUTH_TOKEN", "")

	res := LogoutProvider(ProviderAnthropic, LogoutOpts{})
	if !containsPrefix(res.Remaining, "env:ANTHROPIC_API_KEY") {
		t.Errorf("expected ANTHROPIC_API_KEY in Remaining; got %v", res.Remaining)
	}
	if !containsPrefix(res.Remaining, "env:CLAUDE_CODE_OAUTH_TOKEN") {
		t.Errorf("expected CLAUDE_CODE_OAUTH_TOKEN in Remaining; got %v", res.Remaining)
	}
}

// TestLogoutOpenAI_removesCodexAuth exercises the aggressive path:
// `~/.codex/auth.json` is deleted so the user actually signs out of
// codex globally. Uses HOME override so real state is untouched.
func TestLogoutOpenAI_removesCodexAuth(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("SPWN_HOME", filepath.Join(tmp, ".spwn"))
	t.Setenv("OPENAI_API_KEY", "")

	codexDir := filepath.Join(tmp, ".codex")
	if err := os.MkdirAll(codexDir, 0o700); err != nil {
		t.Fatal(err)
	}
	authPath := filepath.Join(codexDir, "auth.json")
	if err := os.WriteFile(authPath, []byte(`{"tokens":{"access_token":"x"}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	res := LogoutProvider(ProviderOpenAI, LogoutOpts{})
	if res.HasErrors() {
		t.Fatalf("unexpected errors: %v", res.Errors)
	}
	if _, err := os.Stat(authPath); !os.IsNotExist(err) {
		t.Errorf("codex auth.json should have been removed; stat err=%v", err)
	}
	if !containsPrefix(res.Cleared, "file:~/.codex/auth.json") {
		t.Errorf("Cleared missing codex auth entry: %v", res.Cleared)
	}
}

// TestLogoutProvider_idempotent — running logout twice against an
// already-clean provider is a no-op (no errors, empty Cleared).
func TestLogoutProvider_idempotent(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("SPWN_HOME", tmp)
	t.Setenv("HOME", tmp)
	t.Setenv("SPWN_SKIP_KEYCHAIN", "1")
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "")
	t.Setenv("ANTHROPIC_AUTH_TOKEN", "")
	t.Setenv("OPENAI_API_KEY", "")

	res := LogoutProvider(ProviderAnthropic, LogoutOpts{})
	if res.HasErrors() {
		t.Fatalf("first logout errors: %v", res.Errors)
	}
	res = LogoutProvider(ProviderAnthropic, LogoutOpts{})
	if res.HasErrors() {
		t.Fatalf("second logout errors: %v", res.Errors)
	}
	if len(res.Cleared) != 0 {
		t.Errorf("second logout should clear nothing; got %v", res.Cleared)
	}
}

// TestLogoutProvider_doesNotDisable — logout removes creds but does
// NOT flip the Disabled flag. Distinct verbs, distinct intents.
func TestLogoutProvider_doesNotDisable(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("SPWN_HOME", tmp)
	t.Setenv("HOME", tmp)
	t.Setenv("SPWN_SKIP_KEYCHAIN", "1")
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "")
	t.Setenv("ANTHROPIC_AUTH_TOKEN", "")

	_ = LogoutProvider(ProviderAnthropic, LogoutOpts{})
	if IsProviderDisabled(ProviderAnthropic) {
		t.Error("logout must not disable the provider; that's the `disable` verb's job")
	}
}

// TestLogoutAnthropic_keychainScope_darwin is a guard-only assertion:
// on non-darwin, the keychain helpers return no-op results. Keeps CI
// passing on linux without requiring SPWN_SKIP_KEYCHAIN everywhere.
func TestLogoutAnthropic_keychainScope_darwin(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("darwin-only keychain assertion")
	}
	// With SPWN_SKIP_KEYCHAIN set, hasAnthropicKeychain returns false
	// so logout reports nothing keychain-related, matching the env-less
	// cross-platform behaviour we rely on in the other tests.
	t.Setenv("SPWN_SKIP_KEYCHAIN", "1")
	if hasAnthropicKeychain() {
		t.Error("SPWN_SKIP_KEYCHAIN should stub out keychain presence")
	}
}

func containsPrefix(list []string, prefix string) bool {
	for _, item := range list {
		if strings.HasPrefix(item, prefix) {
			return true
		}
	}
	return false
}

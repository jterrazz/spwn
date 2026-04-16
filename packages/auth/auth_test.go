package auth

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveAnthropic_EnvAPIKey(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "sk-ant-test-key")
	t.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "")
	t.Setenv("ANTHROPIC_AUTH_TOKEN", "")
	cred := Resolve(ProviderAnthropic)
	if cred.Type != CredTypeAPIKey {
		t.Errorf("expected API key, got %s", cred.Type)
	}
	if cred.Token != "sk-ant-test-key" {
		t.Errorf("expected sk-ant-test-key, got %s", cred.Token)
	}
	if cred.EnvVar != "ANTHROPIC_API_KEY" {
		t.Errorf("expected ANTHROPIC_API_KEY, got %s", cred.EnvVar)
	}
	if cred.Source != "env:ANTHROPIC_API_KEY" {
		t.Errorf("expected env:ANTHROPIC_API_KEY, got %s", cred.Source)
	}
}

func TestResolveAnthropic_EnvOAuth(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "oauth-test-token")
	t.Setenv("ANTHROPIC_AUTH_TOKEN", "")
	cred := Resolve(ProviderAnthropic)
	if cred.Type != CredTypeOAuth {
		t.Errorf("expected OAuth, got %s", cred.Type)
	}
	if cred.EnvVar != "CLAUDE_CODE_OAUTH_TOKEN" {
		t.Errorf("expected CLAUDE_CODE_OAUTH_TOKEN, got %s", cred.EnvVar)
	}
}

func TestResolveAnthropic_CachedFile(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "")
	t.Setenv("ANTHROPIC_AUTH_TOKEN", "")
	t.Setenv("SPWN_SKIP_KEYCHAIN", "1") // prevent keychain from overriding test
	tmpDir := t.TempDir()
	t.Setenv("SPWN_HOME", tmpDir)
	tokenFile := filepath.Join(tmpDir, ".auth-token")
	os.WriteFile(tokenFile, []byte("sk-ant-cached-key"), 0600)
	cred := Resolve(ProviderAnthropic)
	// On machines with a keychain, keychain may take priority - accept either
	if cred.Type == CredTypeKeychain {
		t.Skip("keychain available, takes priority over cached file")
	}
	if cred.Type != CredTypeAPIKey {
		t.Errorf("expected API key from cache, got %s", cred.Type)
	}
}

func TestResolveAnthropic_NoneWhenEmpty(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "")
	t.Setenv("ANTHROPIC_AUTH_TOKEN", "")
	tmpDir := t.TempDir()
	t.Setenv("SPWN_HOME", tmpDir)
	cred := Resolve(ProviderAnthropic)
	// On machines with Claude Code installed, keychain creds may be found
	if cred.Type != CredTypeNone && cred.Type != CredTypeKeychain {
		t.Errorf("expected none or keychain, got %s", cred.Type)
	}
}

func TestResolveOpenAI(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "sk-openai-test")
	cred := Resolve(ProviderOpenAI)
	if cred.Type != CredTypeAPIKey {
		t.Errorf("expected API key, got %s", cred.Type)
	}
}

func TestResolveGoogle(t *testing.T) {
	t.Setenv("GOOGLE_API_KEY", "google-test-key")
	cred := Resolve(ProviderGoogle)
	if cred.Type != CredTypeAPIKey {
		t.Errorf("expected API key, got %s", cred.Type)
	}
}

func TestSaveAndReadToken(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("SPWN_HOME", tmpDir)
	err := SaveToken("test-token-123")
	if err != nil {
		t.Fatal(err)
	}
	got := ReadCachedToken()
	if got != "test-token-123" {
		t.Errorf("expected test-token-123, got %s", got)
	}
}

func TestResolveAnthropic_AuthToken(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "")
	t.Setenv("ANTHROPIC_AUTH_TOKEN", "auth-token-test")
	tmpDir := t.TempDir()
	t.Setenv("SPWN_HOME", tmpDir)
	cred := Resolve(ProviderAnthropic)
	if cred.Type != CredTypeAPIKey {
		t.Errorf("expected API key, got %s", cred.Type)
	}
	if cred.Token != "auth-token-test" {
		t.Errorf("expected auth-token-test, got %s", cred.Token)
	}
	if cred.Source != "env:ANTHROPIC_AUTH_TOKEN" {
		t.Errorf("expected env:ANTHROPIC_AUTH_TOKEN, got %s", cred.Source)
	}
}

func TestResolveAnthropic_CachedOAuthToken(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "")
	t.Setenv("ANTHROPIC_AUTH_TOKEN", "")
	t.Setenv("SPWN_SKIP_KEYCHAIN", "1")
	tmpDir := t.TempDir()
	t.Setenv("SPWN_HOME", tmpDir)
	tokenFile := filepath.Join(tmpDir, ".auth-token")
	// Non-sk-ant- prefix should be treated as OAuth
	os.WriteFile(tokenFile, []byte("oauth-cached-token"), 0600)
	cred := Resolve(ProviderAnthropic)
	if cred.Type == CredTypeKeychain {
		t.Skip("keychain available, takes priority over cached file")
	}
	if cred.Type != CredTypeOAuth {
		t.Errorf("expected OAuth from cache (non-sk-ant- prefix), got %s", cred.Type)
	}
	if cred.Token != "oauth-cached-token" {
		t.Errorf("expected oauth-cached-token, got %s", cred.Token)
	}
	if cred.EnvVar != "CLAUDE_CODE_OAUTH_TOKEN" {
		t.Errorf("expected CLAUDE_CODE_OAUTH_TOKEN envvar, got %s", cred.EnvVar)
	}
}

func TestResolveGoogle_GeminiAlias(t *testing.T) {
	t.Setenv("GOOGLE_API_KEY", "")
	t.Setenv("GEMINI_API_KEY", "gemini-test-key")
	cred := Resolve(ProviderGoogle)
	if cred.Type != CredTypeAPIKey {
		t.Errorf("expected API key, got %s", cred.Type)
	}
	if cred.Token != "gemini-test-key" {
		t.Errorf("expected gemini-test-key, got %s", cred.Token)
	}
	if cred.Source != "env:GEMINI_API_KEY" {
		t.Errorf("expected env:GEMINI_API_KEY, got %s", cred.Source)
	}
}

func TestValidate_NilCredential(t *testing.T) {
	ctx := context.Background()
	status := Validate(ctx, nil)
	if status == nil {
		t.Fatal("expected non-nil status for nil credential")
	}
	if status.Connected {
		t.Error("expected not connected for nil credential")
	}
	if status.Error != "no credentials" {
		t.Errorf("expected 'no credentials', got %s", status.Error)
	}
}

func TestClearToken(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("SPWN_HOME", tmpDir)

	// Save then clear
	if err := SaveToken("temp-token"); err != nil {
		t.Fatal(err)
	}
	got := ReadCachedToken()
	if got != "temp-token" {
		t.Errorf("expected temp-token, got %s", got)
	}
	if err := ClearToken(); err != nil {
		t.Errorf("ClearToken failed: %v", err)
	}
	got = ReadCachedToken()
	if got != "" {
		t.Errorf("expected empty after clear, got %s", got)
	}

	// ClearToken on non-existent file should return an error (os.Remove behavior)
	err := ClearToken()
	if err == nil {
		t.Error("expected error when clearing non-existent token")
	}
}

func TestValidateAnthropic_InvalidKey(t *testing.T) {
	ctx := context.Background()
	cred := &Credential{
		Provider: ProviderAnthropic,
		Type:     CredTypeAPIKey,
		Token:    "sk-ant-INVALID",
		Source:   "test",
		EnvVar:   "ANTHROPIC_API_KEY",
	}
	status := Validate(ctx, cred)
	if status.Provider != ProviderAnthropic {
		t.Errorf("expected anthropic, got %s", status.Provider)
	}
	if status.CredType != CredTypeAPIKey {
		t.Errorf("expected api_key, got %s", status.CredType)
	}
}

func TestValidate_NoCredential(t *testing.T) {
	ctx := context.Background()
	cred := &Credential{
		Provider: ProviderAnthropic,
		Type:     CredTypeNone,
		Source:   "not configured",
	}
	status := Validate(ctx, cred)
	if status.Connected {
		t.Error("expected not connected")
	}
	if status.Error != "no credentials configured" {
		t.Errorf("expected 'no credentials configured', got %s", status.Error)
	}
}

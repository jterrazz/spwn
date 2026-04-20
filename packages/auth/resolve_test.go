package auth

import (
	"testing"
)

// TestDetectMethods_listsAllAnthropicSources checks that the detection
// layer is transparent: when two env vars are set simultaneously, both
// credentials surface. The dashboard depends on this — users need to
// see every credential spwn could use, not just the winner.
func TestDetectMethods_listsAllAnthropicSources(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "sk-ant-api")
	t.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "oauth-tok")
	t.Setenv("ANTHROPIC_AUTH_TOKEN", "")
	t.Setenv("SPWN_SKIP_KEYCHAIN", "1")
	t.Setenv("SPWN_HOME", t.TempDir())

	got := DetectMethods(ProviderAnthropic)
	if len(got) < 2 {
		t.Fatalf("DetectMethods returned %d creds, want ≥2; got=%+v", len(got), got)
	}
	sawAPIKey, sawOAuth := false, false
	for _, cred := range got {
		if cred.Source == "env:ANTHROPIC_API_KEY" && cred.Method() == MethodAPIKey {
			sawAPIKey = true
		}
		if cred.Source == "env:CLAUDE_CODE_OAUTH_TOKEN" && cred.Method() == MethodOAuth {
			sawOAuth = true
		}
	}
	if !sawAPIKey || !sawOAuth {
		t.Errorf("expected both api_key + oauth in detection list; api_key=%v oauth=%v", sawAPIKey, sawOAuth)
	}
}

// TestResolve_ActiveMethodOverridesDiscoveryOrder confirms the user's
// auth.yaml choice wins. Without the preference, api_key would win
// (first in discovery order for Anthropic); with ActiveMethod=oauth,
// the OAuth env var is selected instead.
func TestResolve_ActiveMethodOverridesDiscoveryOrder(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "sk-ant-api")
	t.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "oauth-tok")
	t.Setenv("ANTHROPIC_AUTH_TOKEN", "")
	t.Setenv("SPWN_SKIP_KEYCHAIN", "1")
	t.Setenv("SPWN_HOME", t.TempDir())

	// Baseline: auto-select grabs the API key (discovery order).
	if cred := Resolve(ProviderAnthropic); cred.Method() != MethodAPIKey {
		t.Fatalf("baseline: got %q, want api_key", cred.Method())
	}

	// User switches preference.
	if err := SetActiveMethod(ProviderAnthropic, MethodOAuth); err != nil {
		t.Fatal(err)
	}
	cred := Resolve(ProviderAnthropic)
	if cred.Method() != MethodOAuth {
		t.Errorf("after ActiveMethod=oauth: got %q, want oauth", cred.Method())
	}
	if cred.Token != "oauth-tok" {
		t.Errorf("selected token = %q, want oauth-tok", cred.Token)
	}
}

// TestResolve_ActiveMethodFallsBackWhenMissing checks that a stale
// preference (user asked for api_key but only oauth is detected)
// doesn't strand them — resolution falls back to whatever's available
// rather than erroring out at runtime.
func TestResolve_ActiveMethodFallsBackWhenMissing(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "oauth-tok")
	t.Setenv("ANTHROPIC_AUTH_TOKEN", "")
	t.Setenv("SPWN_SKIP_KEYCHAIN", "1")
	t.Setenv("SPWN_HOME", t.TempDir())

	if err := SetActiveMethod(ProviderAnthropic, MethodAPIKey); err != nil {
		t.Fatal(err)
	}
	cred := Resolve(ProviderAnthropic)
	if cred.Method() != MethodOAuth {
		t.Errorf("missing preferred method: expected fallback to oauth, got %q", cred.Method())
	}
}

// TestResolve_DisabledReturnsNone — a disabled provider must return a
// CredTypeNone sentinel regardless of whether credentials exist. This
// is the explicit-logout-without-deletion behaviour the new UX relies
// on.
func TestResolve_DisabledReturnsNone(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "sk-ant-api")
	t.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "")
	t.Setenv("ANTHROPIC_AUTH_TOKEN", "")
	t.Setenv("SPWN_SKIP_KEYCHAIN", "1")
	t.Setenv("SPWN_HOME", t.TempDir())

	if err := DisableProvider(ProviderAnthropic); err != nil {
		t.Fatal(err)
	}
	cred := Resolve(ProviderAnthropic)
	if cred.Type != CredTypeNone {
		t.Errorf("disabled provider: got Type=%q, want none", cred.Type)
	}
	if cred.Source != "disabled" {
		t.Errorf("disabled provider: got Source=%q, want \"disabled\"", cred.Source)
	}
}

// TestDetectMethods_isNotFilteredByDisabled — the dashboard needs to
// render every known credential even when the provider is disabled,
// so the user can re-enable and resume without re-discovering them.
func TestDetectMethods_isNotFilteredByDisabled(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "sk-ant-api")
	t.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "")
	t.Setenv("ANTHROPIC_AUTH_TOKEN", "")
	t.Setenv("SPWN_SKIP_KEYCHAIN", "1")
	t.Setenv("SPWN_HOME", t.TempDir())

	if err := DisableProvider(ProviderAnthropic); err != nil {
		t.Fatal(err)
	}
	got := DetectMethods(ProviderAnthropic)
	if len(got) == 0 {
		t.Error("DetectMethods should still list creds even when provider is disabled")
	}
}

func TestCredentialMethod(t *testing.T) {
	for _, tt := range []struct {
		credType CredentialType
		want     Method
	}{
		{CredTypeAPIKey, MethodAPIKey},
		{CredTypeOAuth, MethodOAuth},
		{CredTypeKeychain, MethodOAuth}, // keychain always backs OAuth
		{CredTypeNone, ""},
	} {
		c := &Credential{Type: tt.credType}
		if got := c.Method(); got != tt.want {
			t.Errorf("Type=%q: Method()=%q, want %q", tt.credType, got, tt.want)
		}
	}
	if got := (*Credential)(nil).Method(); got != "" {
		t.Errorf("nil cred: Method()=%q, want empty", got)
	}
}

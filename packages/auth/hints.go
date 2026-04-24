package auth

import (
	"fmt"
	"strings"
)

// Hints is the single source of truth for per-(provider × method × state)
// User-facing phrases. One place to edit the wording, one place to keep
// Commands and env var names in sync. Consumed by the `spwn auth`
// Dashboard and by the spawn pre-flight (apps/cli/world).
//
// No raw file paths ever leak through these helpers. Users see
// Actionable verbs (`claude login`, `spwn auth login …`, `export VAR=`)
// Not implementation details (`~/.claude/.credentials.json`,
// `~/.spwn/credentials/…`). If spwn changes where it stores API keys
// Tomorrow, none of these strings flip.

// Method identifies a credential style supported by a provider.
type HintMethod string

const (
	HintMethodOAuth  HintMethod = "oauth"
	HintMethodAPIKey HintMethod = "api_key"
)

// MethodCatalog returns the ordered list of methods a provider
// Supports, in the order the dashboard renders them. Empty for
// Providers with no supported methods (keeps the caller loop-safe).
func MethodCatalog(p Provider) []HintMethod {
	switch p {
	case ProviderAnthropic:
		return []HintMethod{HintMethodOAuth, HintMethodAPIKey}
	case ProviderOpenAI:
		return []HintMethod{HintMethodOAuth, HintMethodAPIKey}
	}
	return nil
}

// NotSetHint returns the exact command a user should run to SET this
// Provider × method combo, when nothing is currently detected. Shown
// Next to a neutral `·` bullet in the dashboard so each unset row
// Doubles as a cheat-sheet for the setup command.
func NotSetHint(p Provider, m HintMethod) string {
	switch p {
	case ProviderAnthropic:
		switch m {
		case HintMethodOAuth:
			return "run `claude login` on the host"
		case HintMethodAPIKey:
			return "spwn auth login anthropic --api-key sk-ant-…"
		}
	case ProviderOpenAI:
		switch m {
		case HintMethodOAuth:
			return "run `codex login` on the host"
		case HintMethodAPIKey:
			return "spwn auth login openai --api-key sk-… or export OPENAI_API_KEY=sk-…"
		}
	}
	return ""
}

// RejectedHint returns the cure for a credential that was detected
// But the provider's API rejected (401 / invalid). Tailored to the
// Source type — env vars get "unset X" so the user knows the fix is
// In their shell, not in a stored file.
func RejectedHint(p Provider, cred *Credential) string {
	if cred == nil {
		return NotSetHint(p, HintMethodOAuth) // generic fallback
	}
	// Env-var backed creds — tell the user which var is stale.
	if strings.HasPrefix(cred.Source, "env:") {
		varName := strings.TrimPrefix(cred.Source, "env:")
		return "unset " + varName + " (stale in your shell) or replace with a fresh credential"
	}
	// OAuth / keychain / file — point at the tool's native login.
	switch p {
	case ProviderAnthropic:
		return "run `claude login` to refresh"
	case ProviderOpenAI:
		return "run `codex login` to refresh"
	}
	return "refresh " + string(p) + " credentials"
}

// NotConfiguredHint is the all-methods-empty case. Gives the user
// The two preferred paths up-front: OAuth via the native tool, or
// API-key via spwn's store.
func NotConfiguredHint(p Provider) string {
	switch p {
	case ProviderAnthropic:
		return "run `claude login` (subscription) or `spwn auth login anthropic --api-key sk-ant-…`"
	case ProviderOpenAI:
		return "run `codex login` (subscription) or export OPENAI_API_KEY=sk-…"
	}
	return "configure credentials for " + string(p)
}

// CredentialError builds the user-visible error string for a
// Pre-flight validation failure. Shape:
//
//	"anthropic credentials rejected (keychain:Claude Code): invalid or expired OAuth token"
//
// The (source) detail is what makes the error diagnostic — tells
// The user exactly which of their credentials spwn picked up.
func CredentialError(p Provider, cred *Credential, status *ProviderStatus) error {
	source := "no credentials configured"
	if cred != nil && cred.Source != "" {
		source = cred.Source
	}
	reason := "validation failed"
	if status != nil && status.Error != "" {
		reason = status.Error
	}
	return fmt.Errorf("%s credentials rejected (%s): %s", p, source, reason)
}

// CredentialFixHint returns the actionable hint for a pre-flight
// Validation failure. Wraps RejectedHint / NotConfiguredHint with a
// Single entry point so the spawn path has one call to make.
func CredentialFixHint(p Provider, cred *Credential) string {
	if cred == nil || cred.Type == CredTypeNone {
		return NotConfiguredHint(p)
	}
	return RejectedHint(p, cred)
}

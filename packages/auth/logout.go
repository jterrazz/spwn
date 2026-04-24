package auth

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"spwn.sh/packages/platform"
)

// LogoutOpts scopes a LogoutProvider call. Zero value = "purge
// everything on disk and in the keychain for this provider".
type LogoutOpts struct {
	// Method, when non-empty, restricts logout to credentials that
	// match this method. E.g. `LogoutOpts{Method: MethodAPIKey}`
	// removes `.auth-token` only when it holds an sk-ant- key and
	// leaves the keychain (OAuth-backed) alone.
	Method Method
}

// LogoutResult summarises what LogoutProvider did. Surfaced to the
// CLI so users see exactly which stores were touched, which remain,
// and anything that failed — credentials are sensitive enough that
// silent partial-failures would be a footgun.
type LogoutResult struct {
	// Cleared lists human-readable descriptions of what was removed
	// (e.g. "keychain:Claude Code", "file:~/.spwn/.auth-token").
	Cleared []string
	// Remaining lists things spwn can SEE but cannot clear on its own
	// — active env vars, mostly. The caller tells the user to unset
	// them manually.
	Remaining []string
	// Errors collects per-store failures. Logout keeps going through
	// every applicable store so a single failure doesn't mask the
	// rest; callers inspect this to decide exit code.
	Errors []error
}

// HasErrors reports whether any store failed. Separate from len(Errors)
// so callers can branch without importing "errors".
func (r *LogoutResult) HasErrors() bool { return len(r.Errors) > 0 }

// LogoutProvider removes all on-disk and keychain credentials for a
// provider, honouring opts.Method to scope by credential style. Env
// vars cannot be unset from the caller's shell; those are recorded
// in LogoutResult.Remaining for the UI to surface.
//
// Does NOT flip the provider's Disabled flag — logout is about
// removing creds, disable is about saying "never use this provider".
// Distinct verbs, distinct intents.
func LogoutProvider(p Provider, opts LogoutOpts) *LogoutResult {
	res := &LogoutResult{}
	switch p {
	case ProviderAnthropic:
		logoutAnthropic(opts, res)
	case ProviderOpenAI:
		logoutOpenAI(opts, res)
	case ProviderGoogle:
		logoutGoogle(opts, res)
	}
	// Always best-effort re-sync the bind-mount credential dir so the
	// Next container spawn doesn't inject a stale token it remembers
	// But that we just cleared on the host.
	_ = SyncCredentials()
	// Clear any positive-validation cache for this provider so the
	// Next spawn re-checks from scratch instead of trusting a stale
	// "last time it worked" marker for creds we just deleted.
	InvalidateValidation(p)
	return res
}

// logoutAnthropic removes:
//   - the macOS keychain entry "Claude Code-credentials" (OAuth)
//   - ~/.spwn/.auth-token, when method-scoped or when unscoped
//   - ~/.spwn/credentials/anthropic/ mirror dir
//
// Honours opts.Method: api_key scope spares the keychain; oauth scope
// spares `.auth-token` when it holds an sk-ant- key.
func logoutAnthropic(opts LogoutOpts, res *LogoutResult) {
	// Keychain — OAuth only.
	if opts.Method == "" || opts.Method == MethodOAuth {
		if hasAnthropicKeychain() {
			if err := deleteAnthropicKeychain(); err != nil {
				res.Errors = append(res.Errors, fmt.Errorf("keychain Claude Code: %w", err))
			} else {
				res.Cleared = append(res.Cleared, "keychain:Claude Code")
			}
		}
	}

	// ~/.spwn/.auth-token — kind depends on prefix.
	tokenPath := platform.BaseDir() + "/.auth-token"
	if cached, err := os.ReadFile(tokenPath); err == nil {
		trimmed := strings.TrimSpace(string(cached))
		isAPIKey := strings.HasPrefix(trimmed, "sk-ant-")
		shouldRemove := opts.Method == "" ||
			(opts.Method == MethodAPIKey && isAPIKey) ||
			(opts.Method == MethodOAuth && !isAPIKey)
		if trimmed != "" && shouldRemove {
			if err := os.Remove(tokenPath); err != nil {
				res.Errors = append(res.Errors, fmt.Errorf("remove %s: %w", tokenPath, err))
			} else {
				res.Cleared = append(res.Cleared, "file:~/.spwn/.auth-token")
			}
		}
	}

	// Mirrored dir under credentials/anthropic/ (written by sync for
	// runtime-specific formats). Safe to nuke unconditionally regardless
	// of method — it's a derived artefact, never the source of truth.
	if opts.Method == "" {
		mirror := filepath.Join(platform.CredentialsDir(), "anthropic")
		if _, err := os.Stat(mirror); err == nil {
			if err := os.RemoveAll(mirror); err != nil {
				res.Errors = append(res.Errors, fmt.Errorf("remove %s: %w", mirror, err))
			} else {
				res.Cleared = append(res.Cleared, "mirror:~/.spwn/credentials/anthropic/")
			}
		}
	}

	// Env vars we can't touch — surface them so the UI warns.
	if opts.Method == "" || opts.Method == MethodAPIKey {
		if v := os.Getenv("ANTHROPIC_API_KEY"); v != "" {
			res.Remaining = append(res.Remaining, "env:ANTHROPIC_API_KEY")
		}
		if v := os.Getenv("ANTHROPIC_AUTH_TOKEN"); v != "" {
			res.Remaining = append(res.Remaining, "env:ANTHROPIC_AUTH_TOKEN")
		}
	}
	if opts.Method == "" || opts.Method == MethodOAuth {
		if v := os.Getenv("CLAUDE_CODE_OAUTH_TOKEN"); v != "" {
			res.Remaining = append(res.Remaining, "env:CLAUDE_CODE_OAUTH_TOKEN")
		}
	}
}

// logoutOpenAI removes:
//   - ~/.codex/auth.json (OAuth from the codex CLI)
//   - ~/.spwn/credentials/openai/auth.json (our mirror)
//
// Nuking ~/.codex/auth.json is aggressive — it signs the user out of
// the codex CLI globally — but it is the explicit intent of `spwn auth
// logout openai`. Users who only want to disable the provider for spwn
// without touching codex should run `spwn auth disable openai` instead.
func logoutOpenAI(opts LogoutOpts, res *LogoutResult) {
	if opts.Method == "" || opts.Method == MethodOAuth {
		home, _ := os.UserHomeDir()
		codexAuth := filepath.Join(home, ".codex", "auth.json")
		if _, err := os.Stat(codexAuth); err == nil {
			if err := os.Remove(codexAuth); err != nil {
				res.Errors = append(res.Errors, fmt.Errorf("remove %s: %w", codexAuth, err))
			} else {
				res.Cleared = append(res.Cleared, "file:~/.codex/auth.json")
			}
		}
		mirror := filepath.Join(platform.CredentialsDir(), "openai", "auth.json")
		if _, err := os.Stat(mirror); err == nil {
			if err := os.Remove(mirror); err != nil {
				res.Errors = append(res.Errors, fmt.Errorf("remove %s: %w", mirror, err))
			} else {
				res.Cleared = append(res.Cleared, "mirror:~/.spwn/credentials/openai/auth.json")
			}
		}
	}
	if opts.Method == "" || opts.Method == MethodAPIKey {
		if v := os.Getenv("OPENAI_API_KEY"); v != "" {
			res.Remaining = append(res.Remaining, "env:OPENAI_API_KEY")
		}
	}
}

// logoutGoogle has no on-disk state to remove — Google creds only
// flow through env vars today. We surface them in Remaining so the
// user knows why the provider is still "connected" after logout.
func logoutGoogle(opts LogoutOpts, res *LogoutResult) {
	if opts.Method == "" || opts.Method == MethodAPIKey {
		for _, key := range []string{"GOOGLE_API_KEY", "GEMINI_API_KEY"} {
			if v := os.Getenv(key); v != "" {
				res.Remaining = append(res.Remaining, "env:"+key)
			}
		}
	}
}

// hasAnthropicKeychain reports whether the "Claude Code-credentials"
// keychain entry exists without reading its contents. On non-darwin or
// when SPWN_SKIP_KEYCHAIN is set, always false.
func hasAnthropicKeychain() bool {
	if runtime.GOOS != "darwin" {
		return false
	}
	if os.Getenv("SPWN_SKIP_KEYCHAIN") != "" {
		return false
	}
	cmd := exec.Command("security", "find-generic-password", "-s", "Claude Code-credentials")
	return cmd.Run() == nil
}

// deleteAnthropicKeychain removes the macOS keychain entry that
// `claude login` writes. Returns nil on darwin when the entry is
// successfully removed; returns an error when deletion itself failed
// (not "not found" — that's handled by hasAnthropicKeychain above).
func deleteAnthropicKeychain() error {
	if runtime.GOOS != "darwin" {
		return errors.New("keychain deletion is macOS-only")
	}
	cmd := exec.Command("security", "delete-generic-password", "-s", "Claude Code-credentials")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

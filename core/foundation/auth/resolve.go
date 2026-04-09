package auth

import (
	"encoding/json"
	"os"
	"os/exec"
	"strings"

	"spwn.sh/core/foundation"
)

// Resolve finds the best available credential for a provider.
// Priority: env vars > cached token file > macOS Keychain
func Resolve(p Provider) *Credential {
	switch p {
	case ProviderAnthropic:
		return resolveAnthropic()
	case ProviderOpenAI:
		return resolveOpenAI()
	case ProviderGoogle:
		return resolveGoogle()
	}
	return nil
}

// ResolveAll returns credentials for all known providers.
func ResolveAll() map[Provider]*Credential {
	result := make(map[Provider]*Credential)
	for _, p := range []Provider{ProviderAnthropic, ProviderOpenAI} {
		result[p] = Resolve(p)
	}
	return result
}

func resolveAnthropic() *Credential {
	// 1. Check ANTHROPIC_API_KEY env var
	if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
		return &Credential{
			Provider: ProviderAnthropic,
			Type:     CredTypeAPIKey,
			Token:    key,
			Source:   "env:ANTHROPIC_API_KEY",
			EnvVar:   "ANTHROPIC_API_KEY",
		}
	}
	// 2. Check CLAUDE_CODE_OAUTH_TOKEN env var
	if token := os.Getenv("CLAUDE_CODE_OAUTH_TOKEN"); token != "" {
		return &Credential{
			Provider: ProviderAnthropic,
			Type:     CredTypeOAuth,
			Token:    token,
			Source:   "env:CLAUDE_CODE_OAUTH_TOKEN",
			EnvVar:   "CLAUDE_CODE_OAUTH_TOKEN",
		}
	}
	// 3. Check ANTHROPIC_AUTH_TOKEN
	if token := os.Getenv("ANTHROPIC_AUTH_TOKEN"); token != "" {
		return &Credential{
			Provider: ProviderAnthropic,
			Type:     CredTypeAPIKey,
			Token:    token,
			Source:   "env:ANTHROPIC_AUTH_TOKEN",
			EnvVar:   "ANTHROPIC_API_KEY",
		}
	}
	// 4. Check cached token file and macOS Keychain.
	//    When both exist and differ, prefer keychain (more likely to be fresh).
	tokenPath := foundation.BaseDir() + "/.auth-token"
	cachedToken := ""
	if data, err := os.ReadFile(tokenPath); err == nil {
		cachedToken = strings.TrimSpace(string(data))
	}
	keychainCred := readKeychainAnthropic()

	// If keychain has a token, prefer it — auto-update the file if stale
	if keychainCred != nil {
		if cachedToken != keychainCred.Token {
			// Keychain is fresher than cached file — update file
			_ = SaveToken(keychainCred.Token)
		}
		return keychainCred
	}

	// No keychain — fall back to cached file
	if cachedToken != "" {
		credType := CredTypeOAuth
		envVar := "CLAUDE_CODE_OAUTH_TOKEN"
		if strings.HasPrefix(cachedToken, "sk-ant-") {
			credType = CredTypeAPIKey
			envVar = "ANTHROPIC_API_KEY"
		}
		return &Credential{
			Provider: ProviderAnthropic,
			Type:     credType,
			Token:    cachedToken,
			Source:   "file:~/.spwn/.auth-token",
			EnvVar:   envVar,
		}
	}
	return &Credential{
		Provider: ProviderAnthropic,
		Type:     CredTypeNone,
		Source:   "not configured",
	}
}

func readKeychainAnthropic() *Credential {
	// Allow tests to skip keychain access
	if os.Getenv("SPWN_SKIP_KEYCHAIN") != "" {
		return nil
	}
	cmd := exec.Command("security", "find-generic-password", "-s", "Claude Code-credentials", "-w")
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	var creds struct {
		ClaudeAIOAuth struct {
			AccessToken string `json:"accessToken"`
		} `json:"claudeAiOauth"`
	}
	if err := json.Unmarshal(out, &creds); err != nil {
		return nil
	}
	if creds.ClaudeAIOAuth.AccessToken == "" {
		return nil
	}
	return &Credential{
		Provider: ProviderAnthropic,
		Type:     CredTypeKeychain,
		Token:    creds.ClaudeAIOAuth.AccessToken,
		Source:   "keychain:Claude Code",
		EnvVar:   "CLAUDE_CODE_OAUTH_TOKEN",
	}
}

func resolveOpenAI() *Credential {
	if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		return &Credential{
			Provider: ProviderOpenAI,
			Type:     CredTypeAPIKey,
			Token:    key,
			Source:   "env:OPENAI_API_KEY",
			EnvVar:   "OPENAI_API_KEY",
		}
	}

	// Check Codex OAuth auth.json (subscription-based, e.g. ChatGPT Plus)
	home, _ := os.UserHomeDir()
	codexAuth := home + "/.codex/auth.json"
	if data, err := os.ReadFile(codexAuth); err == nil {
		var tokens struct {
			Tokens struct {
				AccessToken string `json:"access_token"`
			} `json:"tokens"`
		}
		if json.Unmarshal(data, &tokens) == nil && tokens.Tokens.AccessToken != "" {
			return &Credential{
				Provider: ProviderOpenAI,
				Type:     CredTypeOAuth,
				Token:    tokens.Tokens.AccessToken,
				Source:   "file:~/.codex/auth.json",
				EnvVar:   "OPENAI_API_KEY",
			}
		}
	}

	return &Credential{
		Provider: ProviderOpenAI,
		Type:     CredTypeNone,
		Source:   "not configured",
	}
}

func resolveGoogle() *Credential {
	if key := os.Getenv("GOOGLE_API_KEY"); key != "" {
		return &Credential{
			Provider: ProviderGoogle,
			Type:     CredTypeAPIKey,
			Token:    key,
			Source:   "env:GOOGLE_API_KEY",
			EnvVar:   "GOOGLE_API_KEY",
		}
	}
	// Also check GEMINI_API_KEY as an alias
	if key := os.Getenv("GEMINI_API_KEY"); key != "" {
		return &Credential{
			Provider: ProviderGoogle,
			Type:     CredTypeAPIKey,
			Token:    key,
			Source:   "env:GEMINI_API_KEY",
			EnvVar:   "GOOGLE_API_KEY",
		}
	}
	return &Credential{
		Provider: ProviderGoogle,
		Type:     CredTypeNone,
		Source:   "not configured",
	}
}

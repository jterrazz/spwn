package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"spwn.sh/packages/platform"
)

// ── Resolution ────────────────────────────────────────────────────────

// resolveAnthropic returns the best available Anthropic credential.
// Priority: env vars (API key, OAuth, auth token) > cached token
// file > macOS Keychain.
func resolveAnthropic() *Credential {
	if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
		return &Credential{
			Provider: ProviderAnthropic,
			Type:     CredTypeAPIKey,
			Token:    key,
			Source:   "env:ANTHROPIC_API_KEY",
			EnvVar:   "ANTHROPIC_API_KEY",
		}
	}
	if token := os.Getenv("CLAUDE_CODE_OAUTH_TOKEN"); token != "" {
		return &Credential{
			Provider: ProviderAnthropic,
			Type:     CredTypeOAuth,
			Token:    token,
			Source:   "env:CLAUDE_CODE_OAUTH_TOKEN",
			EnvVar:   "CLAUDE_CODE_OAUTH_TOKEN",
		}
	}
	if token := os.Getenv("ANTHROPIC_AUTH_TOKEN"); token != "" {
		return &Credential{
			Provider: ProviderAnthropic,
			Type:     CredTypeAPIKey,
			Token:    token,
			Source:   "env:ANTHROPIC_AUTH_TOKEN",
			EnvVar:   "ANTHROPIC_API_KEY",
		}
	}
	// Cached token file + macOS Keychain. When both exist and
	// differ, prefer keychain (more likely to be fresh).
	tokenPath := platform.BaseDir() + "/.auth-token"
	cachedToken := ""
	if data, err := os.ReadFile(tokenPath); err == nil {
		cachedToken = strings.TrimSpace(string(data))
	}
	keychainCred := readKeychainAnthropic()

	if keychainCred != nil {
		if cachedToken != keychainCred.Token {
			_ = SaveToken(keychainCred.Token)
		}
		return keychainCred
	}

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

// readKeychainAnthropic pulls the Claude Code-credentials entry
// from the macOS keychain. Returns nil when unavailable (non-darwin,
// no entry, SPWN_SKIP_KEYCHAIN set for tests).
func readKeychainAnthropic() *Credential {
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

// ── Validation ────────────────────────────────────────────────────────

func validateAnthropic(ctx context.Context, cred *Credential) *ProviderStatus {
	status := &ProviderStatus{
		Provider: ProviderAnthropic,
		CredType: cred.Type,
		Source:   cred.Source,
	}

	if cred.Type == CredTypeOAuth || cred.Type == CredTypeKeychain {
		return validateAnthropicOAuth(ctx, cred, status)
	}
	return validateAnthropicAPIKey(ctx, cred, status)
}

func validateAnthropicOAuth(ctx context.Context, cred *Credential, status *ProviderStatus) *ProviderStatus {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.anthropic.com/api/oauth/usage", nil)
	if err != nil {
		status.Error = err.Error()
		return status
	}
	req.Header.Set("Authorization", "Bearer "+cred.Token)
	req.Header.Set("anthropic-beta", "oauth-2025-04-20")
	req.Header.Set("User-Agent", "spwn/1.0")

	resp, err := client.Do(req)
	if err != nil {
		status.Error = fmt.Sprintf("connection failed: %v", err)
		return status
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		status.Error = "invalid or expired OAuth token"
		return status
	}
	if resp.StatusCode != 200 {
		status.Error = fmt.Sprintf("API returned %d: %s", resp.StatusCode, string(body))
		return status
	}

	var usage struct {
		FiveHour *struct {
			Utilization float64 `json:"utilization"`
			ResetsAt    string  `json:"resets_at"`
		} `json:"five_hour"`
		SevenDay *struct {
			Utilization float64 `json:"utilization"`
			ResetsAt    string  `json:"resets_at"`
		} `json:"seven_day"`
		ExtraUsage *struct {
			IsEnabled    bool    `json:"is_enabled"`
			MonthlyLimit float64 `json:"monthly_limit"`
			UsedCredits  float64 `json:"used_credits"`
			Currency     string  `json:"currency"`
		} `json:"extra_usage"`
	}
	if err := json.Unmarshal(body, &usage); err != nil {
		status.Error = "failed to parse usage response"
		return status
	}

	status.Connected = true
	status.Plan = "subscription"
	usageInfo := &UsageInfo{}
	if usage.FiveHour != nil {
		usageInfo.SessionPercent = usage.FiveHour.Utilization * 100
		usageInfo.SessionResetsAt = usage.FiveHour.ResetsAt
	}
	if usage.SevenDay != nil {
		usageInfo.WeeklyPercent = usage.SevenDay.Utilization * 100
		usageInfo.WeeklyResetsAt = usage.SevenDay.ResetsAt
	}
	if usage.ExtraUsage != nil && usage.ExtraUsage.IsEnabled {
		usageInfo.CreditsUsed = usage.ExtraUsage.UsedCredits
		usageInfo.CreditsLimit = usage.ExtraUsage.MonthlyLimit
		usageInfo.Currency = usage.ExtraUsage.Currency
	}
	status.Usage = usageInfo
	return status
}

func validateAnthropicAPIKey(ctx context.Context, cred *Credential, status *ProviderStatus) *ProviderStatus {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.anthropic.com/v1/models", nil)
	if err != nil {
		status.Error = err.Error()
		return status
	}
	req.Header.Set("x-api-key", cred.Token)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := client.Do(req)
	if err != nil {
		status.Error = fmt.Sprintf("connection failed: %v", err)
		return status
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		status.Error = "invalid API key"
		return status
	}
	if resp.StatusCode == 200 {
		status.Connected = true
		status.Plan = "api_key"
		return status
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	status.Error = fmt.Sprintf("API returned %d: %s", resp.StatusCode, string(body))
	return status
}

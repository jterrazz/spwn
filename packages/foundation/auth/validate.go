package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Validate checks if a credential is valid by making a test API call.
// Returns a ProviderStatus with connection info and usage data.
func Validate(ctx context.Context, cred *Credential) *ProviderStatus {
	if cred == nil {
		return &ProviderStatus{
			Connected: false,
			Error:     "no credentials",
		}
	}
	if cred.Type == CredTypeNone {
		return &ProviderStatus{
			Provider:  cred.Provider,
			Connected: false,
			CredType:  CredTypeNone,
			Source:    cred.Source,
			Error:     "no credentials configured",
		}
	}
	switch cred.Provider {
	case ProviderAnthropic:
		return validateAnthropic(ctx, cred)
	case ProviderOpenAI:
		return validateOpenAI(ctx, cred)
	case ProviderGoogle:
		return validateGoogle(ctx, cred)
	}
	return &ProviderStatus{Provider: cred.Provider, Error: "unknown provider"}
}

// ValidateAll checks all providers.
func ValidateAll(ctx context.Context) []ProviderStatus {
	creds := ResolveAll()
	results := make([]ProviderStatus, 0, len(creds))
	for _, cred := range creds {
		status := Validate(ctx, cred)
		results = append(results, *status)
	}
	return results
}

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
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1MB max

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
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1MB max
	status.Error = fmt.Sprintf("API returned %d: %s", resp.StatusCode, string(body))
	return status
}

func validateOpenAI(ctx context.Context, cred *Credential) *ProviderStatus {
	status := &ProviderStatus{
		Provider: ProviderOpenAI,
		CredType: cred.Type,
		Source:   cred.Source,
	}
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.openai.com/v1/models", nil)
	if err != nil {
		status.Error = err.Error()
		return status
	}
	req.Header.Set("Authorization", "Bearer "+cred.Token)
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
	status.Connected = resp.StatusCode == 200
	if status.Connected {
		status.Plan = "api_key"
	}
	return status
}

func validateGoogle(ctx context.Context, cred *Credential) *ProviderStatus {
	status := &ProviderStatus{
		Provider: ProviderGoogle,
		CredType: cred.Type,
		Source:   cred.Source,
	}
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", "https://generativelanguage.googleapis.com/v1/models?key="+cred.Token, nil)
	if err != nil {
		status.Error = err.Error()
		return status
	}
	resp, err := client.Do(req)
	if err != nil {
		status.Error = fmt.Sprintf("connection failed: %v", err)
		return status
	}
	defer resp.Body.Close()
	if resp.StatusCode == 400 || resp.StatusCode == 401 {
		status.Error = "invalid API key"
		return status
	}
	status.Connected = resp.StatusCode == 200
	if status.Connected {
		status.Plan = "api_key"
	}
	return status
}

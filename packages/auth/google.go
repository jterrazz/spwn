package auth

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"
)

// resolveGoogle returns the best available Google credential.
// Priority: GOOGLE_API_KEY > GEMINI_API_KEY (alias).
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

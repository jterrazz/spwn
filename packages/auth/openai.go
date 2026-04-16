package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

// resolveOpenAI returns the best available OpenAI credential.
// Priority: OPENAI_API_KEY env var > Codex OAuth auth.json
// (subscription-based, e.g. ChatGPT Plus).
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

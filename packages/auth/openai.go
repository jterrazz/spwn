package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

// detectOpenAI enumerates every detected OpenAI credential in
// discovery order:
//  1. env OPENAI_API_KEY       (api_key)
//  2. file ~/.codex/auth.json  (oauth — ChatGPT subscription via codex)
//
// Returned credentials flow into pickByPref to honour any user-declared
// method preference (`spwn auth use openai oauth` / `api_key`).
func detectOpenAI() []*Credential {
	var out []*Credential

	if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		out = append(out, &Credential{
			Provider: ProviderOpenAI,
			Type:     CredTypeAPIKey,
			Token:    key,
			Source:   "env:OPENAI_API_KEY",
			EnvVar:   "OPENAI_API_KEY",
		})
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
			out = append(out, &Credential{
				Provider: ProviderOpenAI,
				Type:     CredTypeOAuth,
				Token:    tokens.Tokens.AccessToken,
				Source:   "file:~/.codex/auth.json",
				EnvVar:   "OPENAI_API_KEY",
			})
		}
	}

	return out
}

func resolveOpenAI() *Credential {
	return pickByPref(ProviderOpenAI, detectOpenAI())
}

func validateOpenAI(ctx context.Context, cred *Credential) *ProviderStatus {
	status := &ProviderStatus{
		Provider: ProviderOpenAI,
		CredType: cred.Type,
		Source:   cred.Source,
	}
	// OAuth tokens from `codex login` are ChatGPT-subscription-scoped
	// And do NOT authenticate against the OpenAI /v1/models endpoint
	// The way API keys do — they're a different auth universe. We
	// Can't cheaply verify them from here without hitting codex's
	// Private backend, so we trust the presence of a well-formed
	// Auth.json file and let the runtime surface any real failure at
	// Spawn time. Marking as Connected here means "spwn won't block
	// On this credential," not "we confirmed it works."
	if cred.Type == CredTypeOAuth {
		status.Connected = true
		status.Plan = "subscription (unverified)"
		return status
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

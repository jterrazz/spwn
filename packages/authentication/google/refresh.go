package google

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// DefaultRefreshLeeway is how close to expiry a token gets before
// AccessToken triggers a proactive refresh. 5 min keeps us well
// inside Google's 1h token lifetime even on slow refreshes.
const DefaultRefreshLeeway = 5 * time.Minute

var (
	httpClient = &http.Client{Timeout: 15 * time.Second}

	// In-process serialization of refreshes. Without this, two
	// concurrent gate requests racing on an expiring token can
	// double-spend the refresh_token (Google rotates them on each
	// refresh in some configurations).
	refreshMu sync.Mutex
)

// AccessToken returns a non-expired access token, refreshing on
// demand if the cached one is past its leeway. Reads client.json +
// tokens.json from CacheDir each call (cheap — they're tiny files);
// if the user re-runs `spwn auth login google` the next AccessToken
// call picks up the new tokens automatically.
//
// Returns an empty string + nil error when no tokens exist (caller
// should surface an actionable "run spwn auth login google" hint).
func AccessToken(ctx context.Context) (string, error) {
	t, err := LoadTokens()
	if err != nil {
		return "", err
	}
	if t == nil {
		return "", nil
	}
	if !t.Expired(DefaultRefreshLeeway) {
		return t.AccessToken, nil
	}
	if t.RefreshToken == "" {
		return "", fmt.Errorf("token expired and no refresh_token; re-run `spwn auth login google`")
	}

	refreshMu.Lock()
	defer refreshMu.Unlock()

	// Double-check after taking the lock — another goroutine may
	// have refreshed while we waited.
	t, _ = LoadTokens()
	if t != nil && !t.Expired(DefaultRefreshLeeway) {
		return t.AccessToken, nil
	}

	c, err := LoadClient()
	if err != nil {
		return "", err
	}
	if c == nil {
		return "", fmt.Errorf("no client config; re-run `spwn auth login google`")
	}

	newT, err := refreshTokens(ctx, c, t.RefreshToken)
	if err != nil {
		return "", err
	}
	// Preserve the refresh_token across responses that don't return
	// a new one (most Google refreshes don't).
	if newT.RefreshToken == "" {
		newT.RefreshToken = t.RefreshToken
	}
	if err := SaveTokens(newT); err != nil {
		return "", fmt.Errorf("save refreshed tokens: %w", err)
	}
	return newT.AccessToken, nil
}

// Refresh forces a refresh_token grant regardless of expiry, useful
// for the gate's scheduler to pre-warm tokens. Same return contract
// as AccessToken: empty string when there's nothing to refresh.
func Refresh(ctx context.Context) (bool, error) {
	t, err := LoadTokens()
	if err != nil || t == nil || t.RefreshToken == "" {
		return false, err
	}
	c, err := LoadClient()
	if err != nil || c == nil {
		return false, err
	}
	newT, err := refreshTokens(ctx, c, t.RefreshToken)
	if err != nil {
		return false, err
	}
	if newT.RefreshToken == "" {
		newT.RefreshToken = t.RefreshToken
	}
	if err := SaveTokens(newT); err != nil {
		return false, fmt.Errorf("save refreshed tokens: %w", err)
	}
	return true, nil
}

func refreshTokens(ctx context.Context, c *ClientConfig, refreshToken string) (*Tokens, error) {
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("client_id", c.ClientID)
	if c.ClientSecret != "" {
		form.Set("client_secret", c.ClientSecret)
	}
	form.Set("refresh_token", refreshToken)

	req, err := http.NewRequestWithContext(ctx, "POST", TokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("refresh endpoint %d: %s", resp.StatusCode, body)
	}
	var t Tokens
	if err := json.Unmarshal(body, &t); err != nil {
		return nil, fmt.Errorf("parse refresh response: %w", err)
	}
	if t.AccessToken == "" {
		return nil, fmt.Errorf("refresh response missing access_token")
	}
	return &t, nil
}

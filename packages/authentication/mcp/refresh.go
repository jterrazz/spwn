package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// DefaultRefreshLeeway is the window before token expiry within which
// we proactively refresh. Picked to be larger than typical poll
// intervals (per-spawn, per-talk) so we don't churn refreshes, but
// small enough that we re-issue well before a hard expiry. For
// Notion's 1h tokens this means refresh kicks in around the 55min
// mark — still 5min of headroom for the request itself.
const DefaultRefreshLeeway = 5 * time.Minute

// httpClient is package-private so tests can swap it. Never call
// http.DefaultClient directly from package code — that timeout-less
// client can hang an entire spawn if a token endpoint stalls.
var httpClient = &http.Client{Timeout: 15 * time.Second}

// tokenFile mirrors the on-disk shape mcp2cli writes. Field names
// match RFC 6749's token response so a refresh response decodes
// directly into this struct.
type tokenFile struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type,omitempty"`
	ExpiresIn    int    `json:"expires_in,omitempty"`
	Scope        string `json:"scope,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

// clientFile mirrors mcp2cli's client.json. We only need the few
// fields used during refresh (auth method, id/secret).
type clientFile struct {
	ClientID                string `json:"client_id"`
	ClientSecret            string `json:"client_secret,omitempty"`
	TokenEndpointAuthMethod string `json:"token_endpoint_auth_method,omitempty"`
}

// metadata is the subset of OAuth Authorization Server Metadata
// (RFC 8414) we consume for refresh.
type metadata struct {
	TokenEndpoint string `json:"token_endpoint"`
}

// Refresh refreshes p's tokens via the refresh_token grant if the
// stored access token is expired or will expire within `leeway`.
// Returns whether a refresh actually happened.
//
// Silent on already-fresh tokens (returns false, nil). Silent on
// missing tokens (returns false, nil) so callers can call this
// unconditionally without first checking IsAuthenticated. Errors
// are non-fatal for callers — log and continue.
func Refresh(ctx context.Context, p Provider, leeway time.Duration) (bool, error) {
	tokensPath := ProviderTokenPath(p)
	fi, err := os.Stat(tokensPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("stat tokens: %w", err)
	}

	rawTokens, err := os.ReadFile(tokensPath)
	if err != nil {
		return false, fmt.Errorf("read tokens: %w", err)
	}
	var t tokenFile
	if err := json.Unmarshal(rawTokens, &t); err != nil {
		return false, fmt.Errorf("parse tokens: %w", err)
	}

	// No expires_in → assume non-expiring (some providers).
	if t.ExpiresIn <= 0 {
		return false, nil
	}
	expiry := fi.ModTime().Add(time.Duration(t.ExpiresIn) * time.Second)
	if time.Until(expiry) > leeway {
		return false, nil // still fresh
	}

	if t.RefreshToken == "" {
		return false, fmt.Errorf("token expired and no refresh_token; re-run `spwn auth login %s`", p.Name)
	}

	rawClient, err := os.ReadFile(ProviderClientPath(p))
	if err != nil {
		return false, fmt.Errorf("read client info: %w", err)
	}
	var c clientFile
	if err := json.Unmarshal(rawClient, &c); err != nil {
		return false, fmt.Errorf("parse client info: %w", err)
	}

	tokenEndpoint, err := discoverTokenEndpoint(ctx, p.URL)
	if err != nil {
		return false, fmt.Errorf("discover token endpoint: %w", err)
	}

	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", t.RefreshToken)

	req, err := http.NewRequestWithContext(ctx, "POST", tokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	// Auth: client_secret_basic (default per spec) when we have a secret.
	// Public clients (PKCE without secret) put client_id in the form.
	if c.ClientSecret != "" {
		req.SetBasicAuth(c.ClientID, c.ClientSecret)
	} else if c.ClientID != "" {
		form.Set("client_id", c.ClientID)
		req.Body = io.NopCloser(strings.NewReader(form.Encode()))
		req.ContentLength = int64(len(form.Encode()))
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("token endpoint: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("token endpoint returned %d: %s", resp.StatusCode, body)
	}

	var newT tokenFile
	if err := json.Unmarshal(body, &newT); err != nil {
		return false, fmt.Errorf("parse token response: %w", err)
	}
	if newT.AccessToken == "" {
		return false, fmt.Errorf("token response had no access_token")
	}

	// Preserve refresh_token across rotation policies. Some providers
	// rotate (return a new one); others don't. If absent in the
	// response, we keep the existing refresh_token alive.
	if newT.RefreshToken == "" {
		newT.RefreshToken = t.RefreshToken
	}

	out, err := json.Marshal(newT)
	if err != nil {
		return false, err
	}

	// Atomic replace so a partial write never corrupts tokens.json.
	tmp := tokensPath + ".tmp"
	if err := os.WriteFile(tmp, out, 0o600); err != nil {
		return false, err
	}
	if err := os.Rename(tmp, tokensPath); err != nil {
		_ = os.Remove(tmp)
		return false, err
	}
	return true, nil
}

// RefreshAll iterates the Registry and refreshes any provider whose
// tokens are expired or expiring within leeway. Returns the count of
// successful refreshes and per-provider errors. Per-provider failures
// don't stop iteration — one broken provider shouldn't lock out the rest.
//
// Each provider gets a 30s deadline. Calling RefreshAll from a hot
// path (every world spawn) is safe: it's a no-op when tokens are
// fresh, and bounded when they aren't.
func RefreshAll(ctx context.Context, leeway time.Duration) (refreshed int, errs []error) {
	for _, p := range Registry {
		pCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		ok, err := Refresh(pCtx, p, leeway)
		cancel()
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", p.Name, err))
			continue
		}
		if ok {
			refreshed++
		}
	}
	return refreshed, errs
}

// discoverTokenEndpoint fetches OAuth Authorization Server Metadata
// (RFC 8414) for the given resource server URL. Tries the resource
// URL's well-known first, then falls back to the origin
// (scheme://host) — different providers host the document at
// different levels.
func discoverTokenEndpoint(ctx context.Context, resourceURL string) (string, error) {
	candidates := []string{
		strings.TrimRight(resourceURL, "/") + "/.well-known/oauth-authorization-server",
	}
	if u, err := url.Parse(resourceURL); err == nil && u.Scheme != "" && u.Host != "" {
		origin := u.Scheme + "://" + u.Host + "/.well-known/oauth-authorization-server"
		if origin != candidates[0] {
			candidates = append(candidates, origin)
		}
	}

	var lastErr error
	for _, c := range candidates {
		req, err := http.NewRequestWithContext(ctx, "GET", c, nil)
		if err != nil {
			lastErr = err
			continue
		}
		resp, err := httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("%s: status %d", c, resp.StatusCode)
			continue
		}
		var md metadata
		if err := json.Unmarshal(body, &md); err != nil {
			lastErr = fmt.Errorf("%s: %w", c, err)
			continue
		}
		if md.TokenEndpoint == "" {
			lastErr = fmt.Errorf("%s: empty token_endpoint", c)
			continue
		}
		return md.TokenEndpoint, nil
	}
	if lastErr != nil {
		return "", lastErr
	}
	return "", fmt.Errorf("no candidates produced metadata for %s", resourceURL)
}

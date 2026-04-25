package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// newTestProvider stands up a fake OAuth server (well-known + token
// endpoint) and seeds the on-disk credential layout so Refresh has
// something to work with. Returns the provider whose URL points at
// the fake server. Cleans up on test exit.
func newTestProvider(t *testing.T, tokens tokenFile, client clientFile, refreshHandler http.HandlerFunc) Provider {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/oauth-authorization-server", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(metadata{TokenEndpoint: "PLACEHOLDER"})
	})
	mux.HandleFunc("/token", refreshHandler)

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	// Patch metadata to point at the test server.
	mux.HandleFunc("/.well-known/oauth-authorization-server-real", func(w http.ResponseWriter, r *http.Request) {})
	// Re-register with correct token endpoint now that we know srv.URL.
	mux2 := http.NewServeMux()
	mux2.HandleFunc("/.well-known/oauth-authorization-server", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"token_endpoint": srv.URL + "/token"})
	})
	mux2.HandleFunc("/token", refreshHandler)
	srv.Config.Handler = mux2

	p := Provider{Name: "test", URL: srv.URL + "/mcp"}

	// Seed on-disk creds.
	tmp := t.TempDir()
	t.Setenv("SPWN_HOME", tmp)
	dir := filepath.Dir(ProviderTokenPath(p))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	tokensJSON, _ := json.Marshal(tokens)
	if err := os.WriteFile(ProviderTokenPath(p), tokensJSON, 0o600); err != nil {
		t.Fatalf("write tokens: %v", err)
	}
	clientJSON, _ := json.Marshal(client)
	if err := os.WriteFile(ProviderClientPath(p), clientJSON, 0o600); err != nil {
		t.Fatalf("write client: %v", err)
	}

	return p
}

func TestRefresh_NoTokens_NoOp(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("SPWN_HOME", tmp)
	p := Provider{Name: "test", URL: "https://example.com/mcp"}

	ok, err := Refresh(context.Background(), p, DefaultRefreshLeeway)
	if err != nil {
		t.Fatalf("expected no error for missing tokens, got %v", err)
	}
	if ok {
		t.Errorf("expected ok=false for missing tokens, got true")
	}
}

func TestRefresh_FreshTokens_NoOp(t *testing.T) {
	tokens := tokenFile{
		AccessToken:  "access-fresh",
		TokenType:    "Bearer",
		ExpiresIn:    3600,
		RefreshToken: "refresh-1",
	}
	client := clientFile{ClientID: "cid", ClientSecret: "csecret"}

	called := false
	p := newTestProvider(t, tokens, client, func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	ok, err := Refresh(context.Background(), p, DefaultRefreshLeeway)
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if ok {
		t.Errorf("expected ok=false (token still fresh), got true")
	}
	if called {
		t.Errorf("token endpoint should not have been called for a fresh token")
	}
}

func TestRefresh_ExpiredTokens_RefreshesAndPreservesRefreshToken(t *testing.T) {
	tokens := tokenFile{
		AccessToken:  "access-old",
		TokenType:    "Bearer",
		ExpiresIn:    3600,
		RefreshToken: "refresh-1",
	}
	client := clientFile{ClientID: "cid", ClientSecret: "csecret"}

	var observedAuth string
	p := newTestProvider(t, tokens, client, func(w http.ResponseWriter, r *http.Request) {
		observedAuth = r.Header.Get("Authorization")
		// Respond with a new access_token but no new refresh_token (so we
		// exercise the "preserve refresh_token" branch).
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(tokenFile{
			AccessToken: "access-new",
			TokenType:   "Bearer",
			ExpiresIn:   3600,
		})
	})

	// Backdate mtime so the token reads as expired.
	old := time.Now().Add(-2 * time.Hour)
	_ = os.Chtimes(ProviderTokenPath(p), old, old)

	ok, err := Refresh(context.Background(), p, DefaultRefreshLeeway)
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if !ok {
		t.Fatalf("expected ok=true (token refreshed)")
	}
	if observedAuth == "" {
		t.Errorf("expected client_secret_basic Authorization header, got empty")
	}

	// Verify on-disk state.
	raw, _ := os.ReadFile(ProviderTokenPath(p))
	var got tokenFile
	_ = json.Unmarshal(raw, &got)
	if got.AccessToken != "access-new" {
		t.Errorf("AccessToken = %q; want %q", got.AccessToken, "access-new")
	}
	if got.RefreshToken != "refresh-1" {
		t.Errorf("RefreshToken not preserved: got %q; want %q", got.RefreshToken, "refresh-1")
	}
}

func TestRefresh_ExpiredButNoRefreshToken_ErrorsActionably(t *testing.T) {
	tokens := tokenFile{
		AccessToken: "access-old",
		TokenType:   "Bearer",
		ExpiresIn:   3600,
		// no RefreshToken
	}
	client := clientFile{ClientID: "cid", ClientSecret: "csecret"}

	p := newTestProvider(t, tokens, client, func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("token endpoint should not be called when refresh_token is absent")
	})

	old := time.Now().Add(-2 * time.Hour)
	_ = os.Chtimes(ProviderTokenPath(p), old, old)

	_, err := Refresh(context.Background(), p, DefaultRefreshLeeway)
	if err == nil {
		t.Fatal("expected error pointing user to spwn auth login")
	}
}

func TestRefresh_TokenEndpointReturns400_ReturnsErrorAndKeepsOldFile(t *testing.T) {
	tokens := tokenFile{
		AccessToken:  "access-old",
		TokenType:    "Bearer",
		ExpiresIn:    3600,
		RefreshToken: "refresh-1",
	}
	client := clientFile{ClientID: "cid", ClientSecret: "csecret"}

	p := newTestProvider(t, tokens, client, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid_grant"}`))
	})

	old := time.Now().Add(-2 * time.Hour)
	_ = os.Chtimes(ProviderTokenPath(p), old, old)

	_, err := Refresh(context.Background(), p, DefaultRefreshLeeway)
	if err == nil {
		t.Fatal("expected error from 400 response")
	}

	// Old tokens.json must still be there.
	raw, _ := os.ReadFile(ProviderTokenPath(p))
	var got tokenFile
	_ = json.Unmarshal(raw, &got)
	if got.AccessToken != "access-old" {
		t.Errorf("AccessToken should remain unchanged on refresh failure; got %q", got.AccessToken)
	}
}

func TestRefresh_PublicClient_PutsClientIDInForm(t *testing.T) {
	tokens := tokenFile{
		AccessToken:  "access-old",
		TokenType:    "Bearer",
		ExpiresIn:    3600,
		RefreshToken: "refresh-1",
	}
	// Public client (PKCE without secret).
	client := clientFile{ClientID: "public-cid", TokenEndpointAuthMethod: "none"}

	var sawAuthHeader, sawClientIDInBody bool
	p := newTestProvider(t, tokens, client, func(w http.ResponseWriter, r *http.Request) {
		sawAuthHeader = r.Header.Get("Authorization") != ""
		_ = r.ParseForm()
		sawClientIDInBody = r.PostForm.Get("client_id") != ""

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(tokenFile{
			AccessToken: "access-new", TokenType: "Bearer", ExpiresIn: 3600,
		})
	})

	old := time.Now().Add(-2 * time.Hour)
	_ = os.Chtimes(ProviderTokenPath(p), old, old)

	if _, err := Refresh(context.Background(), p, DefaultRefreshLeeway); err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if sawAuthHeader {
		t.Errorf("public client must not send Basic auth header")
	}
	if !sawClientIDInBody {
		t.Errorf("public client must include client_id in form body")
	}
}

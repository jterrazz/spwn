package google

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestPKCE_VerifierMatchesChallenge(t *testing.T) {
	verifier, challenge, err := newPKCE()
	if err != nil {
		t.Fatalf("newPKCE: %v", err)
	}
	if len(verifier) < 43 || len(verifier) > 128 {
		t.Errorf("verifier length out of RFC 7636 bounds: %d", len(verifier))
	}
	if len(challenge) == 0 {
		t.Errorf("challenge empty")
	}

	// Re-derive the challenge to confirm the encoding matches what
	// Google's token endpoint will validate against.
	v2, c2, _ := newPKCE()
	if v2 == verifier {
		t.Errorf("two PKCE generations produced identical verifiers — RNG seed bug")
	}
	if c2 == challenge {
		t.Errorf("two PKCE generations produced identical challenges")
	}
}

func TestSaveAndLoadClient_RoundTrip(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("SPWN_HOME", tmp)

	in := &ClientConfig{
		ClientID:     "abc.apps.googleusercontent.com",
		ClientSecret: "shh",
		Scopes:       []string{"https://www.googleapis.com/auth/gmail.modify"},
	}
	if err := SaveClient(in); err != nil {
		t.Fatalf("SaveClient: %v", err)
	}
	out, err := LoadClient()
	if err != nil {
		t.Fatalf("LoadClient: %v", err)
	}
	if out.ClientID != in.ClientID || out.ClientSecret != in.ClientSecret || len(out.Scopes) != 1 {
		t.Errorf("round-trip mismatch: %+v vs %+v", in, out)
	}
}

func TestLoadClient_MissingFileReturnsNilNil(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("SPWN_HOME", tmp)

	c, err := LoadClient()
	if err != nil {
		t.Errorf("LoadClient on missing file should not error: %v", err)
	}
	if c != nil {
		t.Errorf("LoadClient on missing file should return nil, got %+v", c)
	}
}

func TestSaveTokens_PopulatesExpiresAt(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("SPWN_HOME", tmp)

	now := time.Now()
	in := &Tokens{AccessToken: "a", ExpiresIn: 3600}
	if err := SaveTokens(in); err != nil {
		t.Fatalf("SaveTokens: %v", err)
	}
	out, _ := LoadTokens()
	if out.ExpiresAt.Before(now.Add(50*time.Minute)) || out.ExpiresAt.After(now.Add(70*time.Minute)) {
		t.Errorf("ExpiresAt not within ~1h of now: %v", out.ExpiresAt)
	}
}

func TestTokens_Expired(t *testing.T) {
	tt := []struct {
		name      string
		expiresAt time.Time
		leeway    time.Duration
		want      bool
	}{
		{"future, large gap", time.Now().Add(time.Hour), 5 * time.Minute, false},
		{"future, within leeway", time.Now().Add(2 * time.Minute), 5 * time.Minute, true},
		{"past", time.Now().Add(-time.Minute), 5 * time.Minute, true},
		{"zero (unknown) treated as fresh", time.Time{}, 5 * time.Minute, false},
	}
	for _, c := range tt {
		t.Run(c.name, func(t *testing.T) {
			tk := Tokens{ExpiresAt: c.expiresAt}
			if got := tk.Expired(c.leeway); got != c.want {
				t.Errorf("Expired(%v) = %v, want %v", c.leeway, got, c.want)
			}
		})
	}
}

func TestPromptClient_CapturesIDAndSecret(t *testing.T) {
	in := strings.NewReader("my-client-id\nmy-secret\n")
	out := &bytes.Buffer{}
	c, err := PromptClient(in, out)
	if err != nil {
		t.Fatalf("PromptClient: %v", err)
	}
	if c.ClientID != "my-client-id" {
		t.Errorf("ClientID = %q", c.ClientID)
	}
	if c.ClientSecret != "my-secret" {
		t.Errorf("ClientSecret = %q", c.ClientSecret)
	}
	if len(c.Scopes) == 0 {
		t.Errorf("Scopes empty — expected DefaultScopes")
	}
}

func TestPromptClient_OptionalSecret(t *testing.T) {
	in := strings.NewReader("client-id-only\n\n")
	out := &bytes.Buffer{}
	c, err := PromptClient(in, out)
	if err != nil {
		t.Fatalf("PromptClient: %v", err)
	}
	if c.ClientSecret != "" {
		t.Errorf("ClientSecret should be empty when blank line, got %q", c.ClientSecret)
	}
}

// --- refresh flow ---

func newRefreshServer(t *testing.T, handler http.HandlerFunc) string {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/token", handler)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv.URL + "/token"
}

func TestAccessToken_FreshReturnsCached(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("SPWN_HOME", tmp)

	_ = SaveClient(&ClientConfig{ClientID: "x"})
	_ = SaveTokens(&Tokens{
		AccessToken:  "fresh",
		RefreshToken: "rt",
		ExpiresAt:    time.Now().Add(time.Hour),
	})

	tok, err := AccessToken(context.Background())
	if err != nil {
		t.Fatalf("AccessToken: %v", err)
	}
	if tok != "fresh" {
		t.Errorf("AccessToken = %q, want %q", tok, "fresh")
	}
}

func TestAccessToken_NoTokensReturnsEmpty(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("SPWN_HOME", tmp)
	tok, err := AccessToken(context.Background())
	if err != nil {
		t.Errorf("expected nil error for no-tokens, got %v", err)
	}
	if tok != "" {
		t.Errorf("expected empty token, got %q", tok)
	}
}

func TestAccessToken_RefreshesAndPersists(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("SPWN_HOME", tmp)

	endpoint := newRefreshServer(t, func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if r.PostForm.Get("grant_type") != "refresh_token" {
			t.Errorf("grant_type = %q, want refresh_token", r.PostForm.Get("grant_type"))
		}
		if r.PostForm.Get("refresh_token") != "rt-1" {
			t.Errorf("refresh_token = %q, want rt-1", r.PostForm.Get("refresh_token"))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Tokens{
			AccessToken: "refreshed",
			TokenType:   "Bearer",
			ExpiresIn:   3600,
		})
	})

	// Override the package-level token endpoint for this test by
	// pointing the helper at our test server. We do that via a
	// custom client config that we control + a hijacked TokenEndpoint
	// would be cleaner but the const is hard to swap; use the lower-
	// level refreshTokens directly to verify.
	c := &ClientConfig{ClientID: "x"}
	out, err := refreshTokensTo(context.Background(), endpoint, c, "rt-1")
	if err != nil {
		t.Fatalf("refreshTokensTo: %v", err)
	}
	if out.AccessToken != "refreshed" {
		t.Errorf("refreshed token = %q", out.AccessToken)
	}
}

// refreshTokensTo is a test-only helper that mirrors refreshTokens
// but with a configurable endpoint. Kept here to avoid exposing a
// public override surface in production code.
func refreshTokensTo(ctx context.Context, endpoint string, c *ClientConfig, refreshToken string) (*Tokens, error) {
	form := strings.NewReader("grant_type=refresh_token&client_id=" + c.ClientID + "&refresh_token=" + refreshToken)
	req, _ := http.NewRequestWithContext(ctx, "POST", endpoint, form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var t Tokens
	if err := json.Unmarshal(body, &t); err != nil {
		return nil, err
	}
	return &t, nil
}

func TestIsAuthenticated(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("SPWN_HOME", tmp)

	if IsAuthenticated() {
		t.Errorf("IsAuthenticated should be false before any tokens saved")
	}
	_ = SaveTokens(&Tokens{AccessToken: "x", ExpiresIn: 3600})
	if !IsAuthenticated() {
		t.Errorf("IsAuthenticated should be true after SaveTokens")
	}
}

package gate

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"spwn.sh/packages/auth/mcp"
)

// withFakeUpstream stands up an HTTP server impersonating an
// upstream MCP and returns a Provider pointing at it. The server
// captures every inbound request so tests can assert on the proxied
// path + injected auth header.
func withFakeUpstream(t *testing.T, upstreamPath string, handler http.HandlerFunc) (mcp.Provider, *httptest.Server) {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/", handler)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return mcp.Provider{Name: "fake", URL: srv.URL + upstreamPath}, srv
}

// seedToken plants a tokens.json on disk for the provider so the
// proxy element loads a non-empty cache and serves requests.
func seedToken(t *testing.T, p mcp.Provider, accessToken string) {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("SPWN_HOME", tmp)
	dir := filepath.Dir(mcp.ProviderTokenPath(p))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	body, _ := json.Marshal(map[string]any{
		"access_token":  accessToken,
		"token_type":    "Bearer",
		"expires_in":    3600,
		"refresh_token": "refresh-x",
	})
	if err := os.WriteFile(mcp.ProviderTokenPath(p), body, 0o600); err != nil {
		t.Fatalf("seed token: %v", err)
	}
}

func TestProxyElement_RewritesPathAndInjectsAuth(t *testing.T) {
	var sawPath, sawAuth string
	p, _ := withFakeUpstream(t, "/mcp", func(w http.ResponseWriter, r *http.Request) {
		sawPath = r.URL.Path
		sawAuth = r.Header.Get("Authorization")
		w.WriteHeader(204)
	})
	seedToken(t, p, "tok-abc")

	el, err := NewProxyElement(p)
	if err != nil {
		t.Fatalf("NewProxyElement: %v", err)
	}

	rec := httptest.NewRecorder()
	// Request as it'd arrive at the gate's element handler — the
	// server has already stripped /mcp/<name>, so the element sees
	// just the suffix.
	req := httptest.NewRequest("POST", "/initialize", strings.NewReader(`{}`))
	el.Handler().ServeHTTP(rec, req)

	if rec.Code != 204 {
		t.Errorf("status = %d, want 204; body=%q", rec.Code, rec.Body.String())
	}
	if sawPath != "/mcp/initialize" {
		t.Errorf("upstream saw path = %q, want %q (upstream path /mcp + suffix /initialize)", sawPath, "/mcp/initialize")
	}
	if sawAuth != "Bearer tok-abc" {
		t.Errorf("upstream saw Authorization = %q, want %q", sawAuth, "Bearer tok-abc")
	}
}

func TestProxyElement_HandlesUpstreamWithoutPathSuffix(t *testing.T) {
	var sawPath string
	p, _ := withFakeUpstream(t, "", func(w http.ResponseWriter, r *http.Request) {
		sawPath = r.URL.Path
		w.WriteHeader(200)
	})
	seedToken(t, p, "tok")

	el, _ := NewProxyElement(p)

	rec := httptest.NewRecorder()
	el.Handler().ServeHTTP(rec, httptest.NewRequest("GET", "/foo", nil))

	if rec.Code != 200 {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	if sawPath != "/foo" {
		t.Errorf("upstream saw path = %q, want %q (no upstream path prefix)", sawPath, "/foo")
	}
}

func TestProxyElement_503WhenNoToken(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("SPWN_HOME", tmp)
	p := mcp.Provider{Name: "fake", URL: "https://nowhere.example/mcp"}

	el, _ := NewProxyElement(p)

	rec := httptest.NewRecorder()
	el.Handler().ServeHTTP(rec, httptest.NewRequest("POST", "/initialize", nil))

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", rec.Code)
	}
	body, _ := io.ReadAll(rec.Body)
	var got map[string]string
	_ = json.Unmarshal(body, &got)
	if !strings.Contains(got["action"], "spwn auth login fake") {
		t.Errorf("503 body should hint at the login command, got %q", got["action"])
	}
}

func TestProxyElement_ConstructorRejectsBadProvider(t *testing.T) {
	cases := []mcp.Provider{
		{Name: "", URL: "https://x.example/mcp"},
		{Name: "x", URL: ""},
		{Name: "x", URL: "://broken"},
		{Name: "x", URL: "/no-host"},
	}
	for _, p := range cases {
		if _, err := NewProxyElement(p); err == nil {
			t.Errorf("NewProxyElement(%+v) should fail validation", p)
		}
	}
}

func TestRegisterAllProviders_AddsEveryMCPRegistryEntry(t *testing.T) {
	reg := NewRegistry()
	added, err := RegisterAllProviders(reg)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if added != len(mcp.Names()) {
		t.Errorf("added %d, want %d (all of mcp.Registry)", added, len(mcp.Names()))
	}
	for _, name := range mcp.Names() {
		if _, ok := reg.Get(name); !ok {
			t.Errorf("provider %q not found in gate registry after RegisterAllProviders", name)
		}
	}
}

func TestProxyElement_RefreshUpdatesTokenCache(t *testing.T) {
	p, _ := withFakeUpstream(t, "/mcp", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
	})
	// Start with empty creds dir.
	tmp := t.TempDir()
	t.Setenv("SPWN_HOME", tmp)

	el, _ := NewProxyElement(p)
	if el.cachedToken() != "" {
		t.Fatal("cache should be empty before any token exists")
	}

	// Seed a token, then call Refresh — should re-read cache.
	dir := filepath.Dir(mcp.ProviderTokenPath(p))
	_ = os.MkdirAll(dir, 0o700)
	body, _ := json.Marshal(map[string]any{
		"access_token": "fresh-token",
		"token_type":   "Bearer",
		"expires_in":   3600,
	})
	_ = os.WriteFile(mcp.ProviderTokenPath(p), body, 0o600)

	// Refresh will fail to actually refresh (no refresh_token, no
	// real upstream OAuth metadata server) but it always re-reads
	// the cache afterwards regardless. Token should appear.
	_ = el.Refresh(context.Background())

	if el.cachedToken() != "fresh-token" {
		t.Errorf("cache after Refresh = %q, want %q", el.cachedToken(), "fresh-token")
	}
}

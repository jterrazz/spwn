package gate

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// gateWithProviders spins up a CookieSync with a couple of test
// providers wired into a fresh httptest mux. Sets SPWN_HOME to a
// tmpdir so cookie writes land somewhere disposable.
func gateWithProviders(t *testing.T) (*CookieSync, http.Handler) {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("SPWN_HOME", tmp)

	cs := NewCookieSync()
	cs.RegisterProvider(CookieProvider{
		Name:    "x",
		Domains: []string{"x.com", "twitter.com"},
		Cookies: []string{"auth_token", "ct0"},
	})
	cs.RegisterProvider(CookieProvider{
		Name:    "linkedin",
		Domains: []string{"linkedin.com"},
		Cookies: []string{"li_at"},
	})

	mux := http.NewServeMux()
	cs.RegisterRoutes(mux)
	return cs, mux
}

func do(t *testing.T, mux http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reader = bytes.NewReader(b)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestCookies_ProvidersListsRegistry(t *testing.T) {
	_, mux := gateWithProviders(t)
	rec := do(t, mux, "GET", "/sync/providers", nil)
	if rec.Code != 200 {
		t.Fatalf("status = %d", rec.Code)
	}
	var got []CookieProvider
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if len(got) != 2 {
		t.Errorf("expected 2 registered providers, got %d", len(got))
	}
	// Sorted by name → linkedin first, x second.
	if len(got) >= 2 && (got[0].Name != "linkedin" || got[1].Name != "x") {
		t.Errorf("provider order should be sorted by name: got %+v", got)
	}
}

func TestCookies_StatusIsPublic(t *testing.T) {
	_, mux := gateWithProviders(t)
	rec := do(t, mux, "GET", "/sync/status", nil)
	if rec.Code != 200 {
		t.Errorf("status should be public (no auth), got %d", rec.Code)
	}
	var resp struct {
		Providers []struct {
			Name       string `json:"name"`
			HasCookies bool   `json:"has_cookies"`
		} `json:"providers"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if len(resp.Providers) != 2 {
		t.Errorf("status should list both registered providers, got %d", len(resp.Providers))
	}
}

func TestCookies_PushPersistsAndDropsUnknownNames(t *testing.T) {
	cs, mux := gateWithProviders(t)
	rec := do(t, mux, "POST", "/sync/x", map[string]any{
		"cookies": map[string]string{
			"auth_token": "real-x-token",
			"ct0":        "csrf-value",
			"some_other": "should-be-dropped",
			"PII_LEAK":   "should-also-be-dropped",
		},
		"captured": "2026-04-25T18:00:00Z",
	})
	if rec.Code != 204 {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	raw, err := os.ReadFile(CookieFile("x"))
	if err != nil {
		t.Fatalf("read cookies.json: %v", err)
	}
	var got struct {
		Cookies map[string]string `json:"cookies"`
	}
	_ = json.Unmarshal(raw, &got)
	if got.Cookies["auth_token"] != "real-x-token" {
		t.Errorf("auth_token wrong: %v", got.Cookies)
	}
	if _, ok := got.Cookies["some_other"]; ok {
		t.Errorf("non-allowlisted cookie 'some_other' was persisted: %v", got.Cookies)
	}
	if cs.ProviderLastSync("x").IsZero() {
		t.Errorf("ProviderLastSync(x) should be set after a successful push")
	}
}

func TestCookies_PushRejectsUnknownProvider(t *testing.T) {
	_, mux := gateWithProviders(t)
	rec := do(t, mux, "POST", "/sync/bogus", map[string]any{"cookies": map[string]string{"auth_token": "x"}})
	if rec.Code != 404 {
		t.Errorf("unknown provider should be 404, got %d", rec.Code)
	}
}

func TestCookies_PushRejectsEmptyAllowlistedSet(t *testing.T) {
	_, mux := gateWithProviders(t)
	rec := do(t, mux, "POST", "/sync/x", map[string]any{"cookies": map[string]string{"random": "junk"}})
	if rec.Code != 400 {
		t.Errorf("expected 400 for empty allowlisted set, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "no allowlisted") {
		t.Errorf("body should mention allowlist, got %q", rec.Body.String())
	}
}

func TestCookies_PushMethodOnlyPOST(t *testing.T) {
	_, mux := gateWithProviders(t)
	rec := do(t, mux, "GET", "/sync/x", nil)
	if rec.Code != 405 {
		t.Errorf("GET /sync/x should be 405, got %d", rec.Code)
	}
}

func TestCookies_OPTIONSReturns204ForCORS(t *testing.T) {
	_, mux := gateWithProviders(t)
	rec := do(t, mux, "OPTIONS", "/sync/x", nil)
	if rec.Code != 204 {
		t.Errorf("OPTIONS should be 204 for CORS preflight, got %d", rec.Code)
	}
}

func TestCookies_StatusReportsHasCookiesFromDisk(t *testing.T) {
	cs, mux := gateWithProviders(t)

	// Push first, then check status.
	do(t, mux, "POST", "/sync/x", map[string]any{"cookies": map[string]string{"auth_token": "t"}})

	rec := do(t, mux, "GET", "/sync/status", nil)
	var resp struct {
		Providers []struct {
			Name       string `json:"name"`
			HasCookies bool   `json:"has_cookies"`
			LastSync   string `json:"last_sync"`
		} `json:"providers"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	for _, p := range resp.Providers {
		if p.Name == "x" {
			if !p.HasCookies {
				t.Errorf("status should report has_cookies=true for x after push")
			}
			if p.LastSync == "" {
				t.Errorf("status should include last_sync after push")
			}
			return
		}
	}
	t.Errorf("status didn't include x: %+v", resp)
	_ = cs
}

func TestCookies_RegisterProviderIdempotent(t *testing.T) {
	cs := NewCookieSync()
	cs.RegisterProvider(CookieProvider{Name: "x", Cookies: []string{"v1"}})
	cs.RegisterProvider(CookieProvider{Name: "x", Cookies: []string{"v2"}}) // overrides

	got := cs.Providers()
	if len(got) != 1 {
		t.Errorf("expected 1 provider after re-register, got %d", len(got))
	}
	if got[0].Cookies[0] != "v2" {
		t.Errorf("re-register didn't overwrite, got %v", got)
	}
}

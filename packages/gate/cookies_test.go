package gate

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// withGate spins up a CookieSync wired to a fresh httptest mux and
// returns helpers for testing. Sets SPWN_HOME to a tmpdir so cookie
// writes land somewhere disposable.
func withGate(t *testing.T) (*CookieSync, http.Handler, func(method, path string, headers map[string]string, body any) *httptest.ResponseRecorder) {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("SPWN_HOME", tmp)

	cs := NewCookieSync()
	mux := http.NewServeMux()
	cs.RegisterRoutes(mux)

	do := func(method, path string, headers map[string]string, body any) *httptest.ResponseRecorder {
		var reader *bytes.Reader
		if body != nil {
			b, _ := json.Marshal(body)
			reader = bytes.NewReader(b)
		} else {
			reader = bytes.NewReader(nil)
		}
		req := httptest.NewRequest(method, path, reader)
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		return rec
	}
	return cs, mux, do
}

func TestCookies_ProvidersListsRegistry(t *testing.T) {
	_, _, do := withGate(t)
	rec := do("GET", "/sync/providers", nil, nil)
	if rec.Code != 200 {
		t.Fatalf("status = %d", rec.Code)
	}
	var got []CookieProvider
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if len(got) < 2 {
		t.Errorf("expected default registry to have ≥2 entries, got %d", len(got))
	}
	hasX, hasLI := false, false
	for _, p := range got {
		if p.Name == "x" {
			hasX = true
		}
		if p.Name == "linkedin" {
			hasLI = true
		}
	}
	if !hasX || !hasLI {
		t.Errorf("default providers missing x or linkedin: %+v", got)
	}
}

func TestCookies_StatusRequiresSecret(t *testing.T) {
	_, _, do := withGate(t)
	rec := do("GET", "/sync/status", nil, nil)
	if rec.Code != 401 {
		t.Errorf("expected 401 without secret, got %d", rec.Code)
	}
}

func TestCookies_PushRequiresSecret(t *testing.T) {
	_, _, do := withGate(t)
	rec := do("POST", "/sync/x", nil, map[string]any{"cookies": map[string]string{"auth_token": "x"}})
	if rec.Code != 401 {
		t.Errorf("expected 401 without secret, got %d", rec.Code)
	}
}

func TestCookies_PushRejectsUnknownProvider(t *testing.T) {
	_, _, _ = withGate(t)
	secret, _ := GenerateSecret()
	_, _, do := withGate(t) // re-init to pick up secret in fresh tmp
	_ = secret

	// Re-create with a known secret in this tmp.
	tmp := t.TempDir()
	t.Setenv("SPWN_HOME", tmp)
	_ = os.MkdirAll(filepath.Join(tmp, "gate"), 0o700)
	secret, _ = GenerateSecret()

	cs := NewCookieSync()
	mux := http.NewServeMux()
	cs.RegisterRoutes(mux)

	body, _ := json.Marshal(map[string]any{"cookies": map[string]string{"foo": "bar"}})
	req := httptest.NewRequest("POST", "/sync/bogus", bytes.NewReader(body))
	req.Header.Set("X-Spwn-Secret", secret)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 404 {
		t.Errorf("expected 404 for unknown provider, got %d", rec.Code)
	}

	_ = do
}

func TestCookies_PushPersistsAndDropsUnknownNames(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("SPWN_HOME", tmp)
	_ = os.MkdirAll(filepath.Join(tmp, "gate"), 0o700)
	secret, err := GenerateSecret()
	if err != nil {
		t.Fatalf("GenerateSecret: %v", err)
	}

	cs := NewCookieSync()
	mux := http.NewServeMux()
	cs.RegisterRoutes(mux)

	body, _ := json.Marshal(map[string]any{
		"cookies": map[string]string{
			"auth_token":  "real-x-token",
			"ct0":         "csrf-value",
			"some_other":  "should-be-dropped",
			"PII_LEAK":    "should-also-be-dropped",
		},
		"captured": "2026-04-25T18:00:00Z",
	})
	req := httptest.NewRequest("POST", "/sync/x", bytes.NewReader(body))
	req.Header.Set("X-Spwn-Secret", secret)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != 204 {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}

	// Verify on-disk file shape.
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
	if _, ok := got.Cookies["PII_LEAK"]; ok {
		t.Errorf("non-allowlisted cookie 'PII_LEAK' was persisted: %v", got.Cookies)
	}

	// Verify last-sync was recorded.
	if cs.ProviderLastSync("x").IsZero() {
		t.Errorf("ProviderLastSync(x) is zero, expected a recent timestamp")
	}
}

func TestCookies_PushRejectsEmptyAllowlistedSet(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("SPWN_HOME", tmp)
	_ = os.MkdirAll(filepath.Join(tmp, "gate"), 0o700)
	secret, _ := GenerateSecret()

	cs := NewCookieSync()
	mux := http.NewServeMux()
	cs.RegisterRoutes(mux)
	_ = cs

	// Body has cookies but none on the x allowlist.
	body, _ := json.Marshal(map[string]any{"cookies": map[string]string{"random": "junk"}})
	req := httptest.NewRequest("POST", "/sync/x", bytes.NewReader(body))
	req.Header.Set("X-Spwn-Secret", secret)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 400 {
		t.Errorf("expected 400 for empty allowlisted set, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "no allowlisted") {
		t.Errorf("body should mention allowlist, got %q", rec.Body.String())
	}
}

func TestCookies_StatusReportsLastSync(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("SPWN_HOME", tmp)
	_ = os.MkdirAll(filepath.Join(tmp, "gate"), 0o700)
	secret, _ := GenerateSecret()

	cs := NewCookieSync()
	mux := http.NewServeMux()
	cs.RegisterRoutes(mux)

	// Push a sync.
	body, _ := json.Marshal(map[string]any{"cookies": map[string]string{"auth_token": "tok"}})
	req := httptest.NewRequest("POST", "/sync/x", bytes.NewReader(body))
	req.Header.Set("X-Spwn-Secret", secret)
	mux.ServeHTTP(httptest.NewRecorder(), req)

	// Status should now show the last sync.
	req = httptest.NewRequest("GET", "/sync/status", nil)
	req.Header.Set("X-Spwn-Secret", secret)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("status = %d", rec.Code)
	}
	var resp struct {
		Providers []struct {
			Name     string `json:"name"`
			LastSync string `json:"last_sync"`
		} `json:"providers"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	for _, p := range resp.Providers {
		if p.Name == "x" {
			if p.LastSync == "" {
				t.Errorf("status didn't include x last_sync timestamp")
			}
			return
		}
	}
	t.Errorf("status didn't include x at all: %+v", resp)
}

func TestCookies_GenerateSecret_FormatAndPersistence(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("SPWN_HOME", tmp)

	secret, err := GenerateSecret()
	if err != nil {
		t.Fatalf("GenerateSecret: %v", err)
	}
	if !strings.HasPrefix(secret, "SP-") {
		t.Errorf("secret should start with SP-, got %q", secret)
	}
	if !HasSecret() {
		t.Errorf("HasSecret should be true after GenerateSecret")
	}
	raw, _ := os.ReadFile(SecretPath())
	if strings.TrimSpace(string(raw)) != secret {
		t.Errorf("secret on disk doesn't match returned value: disk=%q returned=%q", raw, secret)
	}
}

func TestCookies_PushMethodOnlyPOST(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("SPWN_HOME", tmp)
	_ = os.MkdirAll(filepath.Join(tmp, "gate"), 0o700)
	secret, _ := GenerateSecret()

	cs := NewCookieSync()
	mux := http.NewServeMux()
	cs.RegisterRoutes(mux)
	_ = cs

	req := httptest.NewRequest("GET", "/sync/x", nil)
	req.Header.Set("X-Spwn-Secret", secret)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 405 {
		t.Errorf("GET /sync/x should be 405, got %d", rec.Code)
	}
}

package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Regression: agent names with spaces (like "QA Eng") must be decodable
// from URL paths. The Go mux decodes %20 automatically, but the frontend
// must encode names before putting them in API paths.

func TestAgentNameWithSpaces_GetProfile(t *testing.T) {
	_, mux := newFullTestServer(t)

	// Create an agent whose name has a space - simulate by hitting the list
	// endpoint. The profile endpoint should 404 gracefully (agent doesn't
	// exist) rather than panicking on the space.

	// URL-encoded space: "QA%20Eng"
	w := httptest.NewRecorder()
	r, _ := http.NewRequest("GET", "/api/agents/QA%20Eng", nil)
	mux.ServeHTTP(w, r)

	// Should get 404 (agent doesn't exist in test env), NOT a panic or 500.
	if w.Code != 404 {
		t.Errorf("expected 404 for missing agent with space in name, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAgentNameWithSpaces_GetMind(t *testing.T) {
	_, mux := newFullTestServer(t)

	w := httptest.NewRecorder()
	r, _ := http.NewRequest("GET", "/api/agents/QA%20Eng/mind", nil)
	mux.ServeHTTP(w, r)

	if w.Code != 404 {
		t.Errorf("expected 404 for mind of missing agent, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAgentNameWithSpecialChars_GetProfile(t *testing.T) {
	_, mux := newFullTestServer(t)

	// Names with hyphens, underscores, unicode
	names := []string{"test-agent", "my_agent", "agent%2Fslash"}
	for _, name := range names {
		w := httptest.NewRecorder()
		r, _ := http.NewRequest("GET", "/api/agents/"+name, nil)
		mux.ServeHTTP(w, r)

		// Should be 404 (not found) not 400/500
		if w.Code != 404 {
			t.Errorf("GET /api/agents/%s: expected 404, got %d", name, w.Code)
		}
	}
}

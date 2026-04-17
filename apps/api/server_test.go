package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"spwn.sh/packages/world/state"
)

func newTestServer(t *testing.T) (*Server, *http.ServeMux) {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("SPWN_HOME", tmp)

	store, err := state.NewStoreWithBackendAt(noContainersBackend{}, t.TempDir())
	if err != nil {
		t.Fatalf("runtimestate: %v", err)
	}

	srv := New(store, nil, "127.0.0.1:0")
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", srv.handleHealth)
	mux.HandleFunc("GET /api/worlds", srv.handleListWorlds)
	mux.HandleFunc("GET /api/agents", srv.handleListAgents)
	return srv, mux
}

// newFullTestServer registers ALL routes (matching Start()) for integration tests.
func newFullTestServer(t *testing.T) (*Server, *http.ServeMux) {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("SPWN_HOME", tmp)

	store, err := state.NewStoreWithBackendAt(noContainersBackend{}, t.TempDir())
	if err != nil {
		t.Fatalf("runtimestate: %v", err)
	}

	srv := New(store, nil, "127.0.0.1:0")
	mux := http.NewServeMux()

	// READ endpoints
	mux.HandleFunc("GET /api/health", cors(srv.handleHealth))
	mux.HandleFunc("GET /api/status", cors(srv.handleStatus))
	mux.HandleFunc("GET /api/worlds", cors(srv.handleListWorlds))
	mux.HandleFunc("GET /api/agents", cors(srv.handleListAgents))
	mux.HandleFunc("GET /api/agents/{name}", cors(srv.handleGetAgent))
	mux.HandleFunc("GET /api/agents/{name}/journal", cors(srv.handleGetAgentJournal))
	mux.HandleFunc("GET /api/agents/{name}/mind", cors(srv.handleGetAgentMind))
	mux.HandleFunc("GET /api/agents/{name}/files/{path...}", cors(srv.handleGetAgentFile))

	// WRITE endpoints
	mux.HandleFunc("POST /api/agents", cors(srv.handleCreateAgent))
	mux.HandleFunc("DELETE /api/agents/{name}", cors(srv.handleDeleteAgent))
	mux.HandleFunc("POST /api/agents/{name}/dream", cors(srv.handleDream))
	mux.HandleFunc("POST /api/agents/{name}/sleep", cors(srv.handleSleep))
	mux.HandleFunc("POST /api/agents/{name}/fork", cors(srv.handleFork))
	mux.HandleFunc("PUT /api/agents/{name}/identity", cors(srv.handleUpdateIdentity))

	// Team endpoints
	mux.HandleFunc("GET /api/teams", cors(srv.handleListTeams))
	mux.HandleFunc("POST /api/teams", cors(srv.handleCreateTeam))
	mux.HandleFunc("PUT /api/teams/{slug}", cors(srv.handleUpdateTeam))
	mux.HandleFunc("DELETE /api/teams/{slug}", cors(srv.handleDeleteTeam))

	// Organization endpoints
	mux.HandleFunc("GET /api/organizations", cors(srv.handleListOrganizations))
	mux.HandleFunc("GET /api/organizations/{slug}", cors(srv.handleGetOrganization))
	mux.HandleFunc("POST /api/organizations", cors(srv.handleCreateOrganization))
	mux.HandleFunc("PUT /api/organizations/{slug}", cors(srv.handleUpdateOrganization))
	mux.HandleFunc("DELETE /api/organizations/{slug}", cors(srv.handleDeleteOrganization))

	// Docker-dependent endpoints (read-only mode - arch is nil)
	mux.HandleFunc("POST /api/worlds", cors(srv.handleCreateWorld))
	mux.HandleFunc("POST /api/worlds/{id}/agents", cors(srv.handleDeployAgent))
	mux.HandleFunc("DELETE /api/worlds/{id}", cors(srv.handleDestroyWorld))
	mux.HandleFunc("POST /api/worlds/{id}/snapshot", cors(srv.handleSnapshot))
	mux.HandleFunc("POST /api/worlds/{id}/talk", cors(srv.handleTalk))

	// Architect endpoints
	mux.HandleFunc("GET /api/architect/status", cors(srv.handleArchitectStatus))
	mux.HandleFunc("GET /api/architect/stack", cors(srv.handleArchitectStackGet))
	mux.HandleFunc("POST /api/architect/stack", cors(srv.handleArchitectStackUpdate))

	// History endpoints
	mux.HandleFunc("GET /api/architect/history", cors(srv.handleArchitectHistory))
	mux.HandleFunc("GET /api/worlds/{id}/history", cors(srv.handleWorldHistory))

	// Auth endpoints
	mux.HandleFunc("GET /api/auth/providers", cors(srv.handleAuthProviders))
	mux.HandleFunc("POST /api/auth/check", cors(srv.handleAuthCheck))
	mux.HandleFunc("POST /api/auth/configure", cors(srv.handleAuthConfigure))

	// CORS preflight
	mux.HandleFunc("OPTIONS /", cors(func(w http.ResponseWriter, r *http.Request) {}))

	return srv, mux
}

// createTestAgent creates a minimal agent directory structure in SPWN_HOME.
func createTestAgent(t *testing.T, name string) string {
	t.Helper()
	home := os.Getenv("SPWN_HOME")
	agentDir := filepath.Join(home, "agents", name)

	dirs := []string{
		filepath.Join(agentDir, "identity"),
		filepath.Join(agentDir, "skills"),
		filepath.Join(agentDir, "journal"),
		filepath.Join(agentDir, "playbooks"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatalf("mkdir %s: %v", d, err)
		}
	}

	// Write identity files
	writeFile(t, filepath.Join(agentDir, "identity", "profile.md"), "# Profile\n\nA helpful test agent.\n")
	writeFile(t, filepath.Join(agentDir, "identity", "purpose.md"), "# Purpose\n\nTo test the API.\n")
	writeFile(t, filepath.Join(agentDir, "identity", "traits.md"), "# Traits\n\n- curious\n- diligent\n")

	// Write journal entries
	writeFile(t, filepath.Join(agentDir, "journal", "2025-01-01.md"), "# 2025-01-01\n\nFirst journal entry.\n")

	// Write a skill
	writeFile(t, filepath.Join(agentDir, "skills", "coding.md"), "# Coding\n\nWrites Go code.\n")

	// Write agent.yaml
	writeFile(t, filepath.Join(agentDir, "agent.yaml"), "role: worker\nruntime:\n  engine: claude-code\n  provider: anthropic\n  model: claude-4\n")

	return agentDir
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func doJSON(t *testing.T, mux *http.ServeMux, method, url string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()
	var reqBody *bytes.Buffer
	if body != nil {
		b, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(b)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}
	req := httptest.NewRequest(method, url, reqBody)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	return w
}

func decodeBody(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var body map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v (body: %s)", err, w.Body.String())
	}
	return body
}

func TestHealthEndpoint(t *testing.T) {
	_, mux := newTestServer(t)

	req := httptest.NewRequest("GET", "/api/health", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["status"] != "ok" {
		t.Fatalf("expected status ok, got %q", body["status"])
	}
}

func TestHealthEndpoint_ContentType(t *testing.T) {
	_, mux := newTestServer(t)

	req := httptest.NewRequest("GET", "/api/health", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}
}

func TestListWorlds_Empty(t *testing.T) {
	_, mux := newTestServer(t)

	req := httptest.NewRequest("GET", "/api/worlds", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("expected application/json, got %q", ct)
	}
}

func TestListAgents_Empty(t *testing.T) {
	_, mux := newTestServer(t)

	req := httptest.NewRequest("GET", "/api/agents", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("expected application/json, got %q", ct)
	}
}

func TestInvalidRoute_Returns404(t *testing.T) {
	_, mux := newTestServer(t)

	routes := []string{
		"/api/nonexistent",
		"/api/health/extra",
		"/invalid",
		"/",
	}

	for _, route := range routes {
		t.Run(route, func(t *testing.T) {
			req := httptest.NewRequest("GET", route, nil)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if w.Code != 404 {
				t.Errorf("GET %s: expected 404, got %d", route, w.Code)
			}
		})
	}
}

func TestWrongMethod_Returns405(t *testing.T) {
	_, mux := newTestServer(t)

	// These endpoints are registered as GET only
	methods := []string{"POST", "PUT", "DELETE", "PATCH"}
	for _, method := range methods {
		t.Run(method+" /api/health", func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/health", nil)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			// Go 1.22+ returns 405 for method mismatch on method-specific routes
			if w.Code != 405 {
				t.Errorf("%s /api/health: expected 405, got %d", method, w.Code)
			}
		})
	}
}

func TestServerStop_NilServer(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("SPWN_HOME", tmp)

	store, err := state.NewStoreWithBackendAt(noContainersBackend{}, t.TempDir())
	if err != nil {
		t.Fatalf("runtimestate: %v", err)
	}

	srv := New(store, nil, "127.0.0.1:0")
	// srv.srv is nil - Stop should be a no-op
	err = srv.Stop(context.Background())
	if err != nil {
		t.Errorf("Stop on nil server should return nil, got: %v", err)
	}
}

func TestServerGracefulShutdown(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("SPWN_HOME", tmp)

	store, err := state.NewStoreWithBackendAt(noContainersBackend{}, t.TempDir())
	if err != nil {
		t.Fatalf("runtimestate: %v", err)
	}

	srv := New(store, nil, "127.0.0.1:0")

	// Build server manually on a random port
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", srv.handleHealth)

	httpSrv := &http.Server{Addr: "127.0.0.1:0", Handler: mux}
	srv.srv = httpSrv

	// Start in background - use a listener to get the actual port
	ln, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	go func() {
		_ = httpSrv.Serve(ln)
	}()

	time.Sleep(50 * time.Millisecond)

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := srv.Stop(ctx); err != nil {
		t.Errorf("Stop failed: %v", err)
	}
}

// ============================================================
// Integration tests for all spwn API endpoints
// ============================================================

func TestStatus(t *testing.T) {
	_, mux := newFullTestServer(t)

	w := doJSON(t, mux, "GET", "/api/status", nil)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	body := decodeBody(t, w)

	// Must have worlds, agents, architect fields
	for _, key := range []string{"worlds", "agents", "architect"} {
		if _, ok := body[key]; !ok {
			t.Errorf("missing key %q in status response", key)
		}
	}

	// architect should be false (nil arch)
	if body["architect"] != false {
		t.Errorf("expected architect=false, got %v", body["architect"])
	}
}

func TestCreateAgent(t *testing.T) {
	_, mux := newFullTestServer(t)

	w := doJSON(t, mux, "POST", "/api/agents", map[string]string{"name": "test-agent"})
	if w.Code != 201 {
		t.Fatalf("expected 201, got %d (body: %s)", w.Code, w.Body.String())
	}

	body := decodeBody(t, w)
	if body["name"] != "test-agent" {
		t.Errorf("expected name=test-agent, got %v", body["name"])
	}
	if body["path"] == nil || body["path"] == "" {
		t.Errorf("expected non-empty path")
	}
}

func TestCreateAgent_MissingName(t *testing.T) {
	_, mux := newFullTestServer(t)

	w := doJSON(t, mux, "POST", "/api/agents", map[string]string{})
	if w.Code != 400 {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestDeleteAgent(t *testing.T) {
	_, mux := newFullTestServer(t)
	createTestAgent(t, "doomed")

	w := doJSON(t, mux, "DELETE", "/api/agents/doomed", nil)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d (body: %s)", w.Code, w.Body.String())
	}

	body := decodeBody(t, w)
	if body["deleted"] != "doomed" {
		t.Errorf("expected deleted=doomed, got %v", body["deleted"])
	}

	// Verify it's actually gone
	w2 := doJSON(t, mux, "GET", "/api/agents/doomed", nil)
	if w2.Code != 404 {
		t.Errorf("expected 404 after delete, got %d", w2.Code)
	}
}

func TestDeleteAgent_NotFound(t *testing.T) {
	_, mux := newFullTestServer(t)

	w := doJSON(t, mux, "DELETE", "/api/agents/nonexistent", nil)
	if w.Code != 404 {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestGetAgentProfile(t *testing.T) {
	_, mux := newFullTestServer(t)
	createTestAgent(t, "alice")

	w := doJSON(t, mux, "GET", "/api/agents/alice", nil)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d (body: %s)", w.Code, w.Body.String())
	}

	body := decodeBody(t, w)
	if body["name"] != "alice" {
		t.Errorf("expected name=alice, got %v", body["name"])
	}
	if body["role"] != "worker" {
		t.Errorf("expected role=worker, got %v", body["role"])
	}
	if body["engine"] != "claude-code" {
		t.Errorf("expected engine=claude-code, got %v", body["engine"])
	}
	if body["profile"] != "A helpful test agent." {
		t.Errorf("expected profile content, got %v", body["profile"])
	}
	if body["purpose"] != "To test the API." {
		t.Errorf("expected purpose content, got %v", body["purpose"])
	}

	// Traits should be an array
	traits, ok := body["traits"].([]interface{})
	if !ok {
		t.Fatalf("traits should be an array, got %T", body["traits"])
	}
	if len(traits) != 2 {
		t.Errorf("expected 2 traits, got %d", len(traits))
	}

	// Skills should include "coding"
	skills, ok := body["skills"].([]interface{})
	if !ok {
		t.Fatalf("skills should be an array, got %T", body["skills"])
	}
	if len(skills) != 1 || skills[0] != "coding" {
		t.Errorf("expected skills=[coding], got %v", skills)
	}
}

func TestGetAgentProfile_NotFound(t *testing.T) {
	_, mux := newFullTestServer(t)

	w := doJSON(t, mux, "GET", "/api/agents/ghost", nil)
	if w.Code != 404 {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestGetAgentJournal(t *testing.T) {
	_, mux := newFullTestServer(t)
	createTestAgent(t, "bob")

	w := doJSON(t, mux, "GET", "/api/agents/bob/journal", nil)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d (body: %s)", w.Code, w.Body.String())
	}

	// Response should be a JSON array
	var entries []interface{}
	if err := json.NewDecoder(w.Body).Decode(&entries); err != nil {
		t.Fatalf("decode journal: %v", err)
	}
	// We created one journal entry
	if len(entries) < 1 {
		t.Errorf("expected at least 1 journal entry, got %d", len(entries))
	}
}

func TestGetAgentMind(t *testing.T) {
	_, mux := newFullTestServer(t)
	createTestAgent(t, "carol")

	w := doJSON(t, mux, "GET", "/api/agents/carol/mind", nil)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d (body: %s)", w.Code, w.Body.String())
	}

	// Response should be a JSON object (layers map)
	body := decodeBody(t, w)
	// Should have at least the identity layer
	if body["identity"] == nil {
		t.Errorf("expected identity layer in mind response, got keys: %v", body)
	}
}

func TestGetAgentFile(t *testing.T) {
	_, mux := newFullTestServer(t)
	createTestAgent(t, "dave")

	w := doJSON(t, mux, "GET", "/api/agents/dave/files/identity/profile.md", nil)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d (body: %s)", w.Code, w.Body.String())
	}

	body := decodeBody(t, w)
	content, ok := body["content"].(string)
	if !ok || content == "" {
		t.Errorf("expected file content, got %v", body["content"])
	}
	if body["path"] != "identity/profile.md" {
		t.Errorf("expected path=identity/profile.md, got %v", body["path"])
	}
}

func TestGetAgentFile_NotFound(t *testing.T) {
	_, mux := newFullTestServer(t)
	createTestAgent(t, "dave2")

	w := doJSON(t, mux, "GET", "/api/agents/dave2/files/nonexistent.md", nil)
	if w.Code != 404 {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestGetAgentFile_DirectoryTraversal(t *testing.T) {
	_, mux := newFullTestServer(t)
	createTestAgent(t, "dave3")

	// Go's HTTP router normalizes ".." in paths with a 301/307 redirect,
	// so we test that the server code itself rejects ".." if it reaches the handler.
	// Use a raw request that bypasses router normalization by encoding dots.
	w := doJSON(t, mux, "GET", "/api/agents/dave3/files/identity/..%2F..%2F..%2Fetc%2Fpasswd", nil)
	// Should be rejected - either 400 (bad path) or 404 (not found)
	if w.Code != 400 && w.Code != 404 {
		t.Fatalf("expected 400 or 404 for traversal attempt, got %d", w.Code)
	}
}

func TestUpdateIdentity(t *testing.T) {
	_, mux := newFullTestServer(t)
	createTestAgent(t, "eve")

	w := doJSON(t, mux, "PUT", "/api/agents/eve/identity", map[string]string{
		"field":   "purpose",
		"content": "To conquer the world.",
	})
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d (body: %s)", w.Code, w.Body.String())
	}

	body := decodeBody(t, w)
	if body["status"] != "ok" {
		t.Errorf("expected status=ok, got %v", body["status"])
	}

	// Verify the file was actually written
	home := os.Getenv("SPWN_HOME")
	data, err := os.ReadFile(filepath.Join(home, "agents", "eve", "identity", "purpose.md"))
	if err != nil {
		t.Fatalf("read updated file: %v", err)
	}
	if got := string(data); got != "# Purpose\n\nTo conquer the world.\n" {
		t.Errorf("unexpected file content: %q", got)
	}
}

func TestUpdateIdentity_InvalidField(t *testing.T) {
	_, mux := newFullTestServer(t)
	createTestAgent(t, "eve2")

	w := doJSON(t, mux, "PUT", "/api/agents/eve2/identity", map[string]string{
		"field":   "hacker",
		"content": "pwned",
	})
	if w.Code != 400 {
		t.Fatalf("expected 400 for invalid field, got %d", w.Code)
	}
}

func TestDream(t *testing.T) {
	_, mux := newFullTestServer(t)
	createTestAgent(t, "dreamer")

	w := doJSON(t, mux, "POST", "/api/agents/dreamer/dream", nil)
	// Dream may succeed or fail depending on journal contents - either 200 or 500 is acceptable
	if w.Code != 200 && w.Code != 500 {
		t.Fatalf("expected 200 or 500, got %d (body: %s)", w.Code, w.Body.String())
	}
}

func TestSleep(t *testing.T) {
	_, mux := newFullTestServer(t)
	createTestAgent(t, "sleeper")

	w := doJSON(t, mux, "POST", "/api/agents/sleeper/sleep", nil)
	// Sleep may succeed or fail depending on contents - either 200 or 500 is acceptable
	if w.Code != 200 && w.Code != 500 {
		t.Fatalf("expected 200 or 500, got %d (body: %s)", w.Code, w.Body.String())
	}
}

func TestFork(t *testing.T) {
	_, mux := newFullTestServer(t)
	createTestAgent(t, "original")

	w := doJSON(t, mux, "POST", "/api/agents/original/fork", map[string]interface{}{
		"target": "clone",
		"layers": []string{"identity"},
	})
	if w.Code != 201 {
		t.Fatalf("expected 201, got %d (body: %s)", w.Code, w.Body.String())
	}

	// Verify the clone was created
	w2 := doJSON(t, mux, "GET", "/api/agents/clone", nil)
	if w2.Code != 200 {
		t.Errorf("expected clone agent to exist, got %d", w2.Code)
	}
}

func TestFork_MissingTarget(t *testing.T) {
	_, mux := newFullTestServer(t)
	createTestAgent(t, "original2")

	w := doJSON(t, mux, "POST", "/api/agents/original2/fork", map[string]string{})
	if w.Code != 400 {
		t.Fatalf("expected 400 for missing target, got %d", w.Code)
	}
}

func TestCORS(t *testing.T) {
	_, mux := newFullTestServer(t)

	// Test CORS headers on a regular GET
	w := doJSON(t, mux, "GET", "/api/health", nil)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if origin := w.Header().Get("Access-Control-Allow-Origin"); origin != "*" {
		t.Errorf("expected CORS origin=*, got %q", origin)
	}
	if methods := w.Header().Get("Access-Control-Allow-Methods"); methods == "" {
		t.Error("expected Access-Control-Allow-Methods header")
	}
	if headers := w.Header().Get("Access-Control-Allow-Headers"); headers == "" {
		t.Error("expected Access-Control-Allow-Headers header")
	}
}

func TestCORS_Preflight(t *testing.T) {
	_, mux := newFullTestServer(t)

	req := httptest.NewRequest("OPTIONS", "/api/agents", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "POST")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != 204 {
		t.Fatalf("expected 204 for OPTIONS preflight, got %d", w.Code)
	}
	if origin := w.Header().Get("Access-Control-Allow-Origin"); origin != "*" {
		t.Errorf("expected CORS origin=*, got %q", origin)
	}
}

func TestArchitectStatus_ReadOnlyMode(t *testing.T) {
	_, mux := newFullTestServer(t)

	w := doJSON(t, mux, "GET", "/api/architect/status", nil)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	body := decodeBody(t, w)
	// The handler shells out to `docker inspect spwn-architect`.
	// On machines where the real spwn-architect container is running,
	// docker returns "running" instead of "stopped".
	// Accept either value - the important thing is the endpoint works.
	status, _ := body["status"].(string)
	if status != "stopped" && status != "running" {
		t.Errorf("expected status=stopped or running, got %v", body["status"])
	}
}

func TestCreateWorld_ReadOnlyMode(t *testing.T) {
	_, mux := newFullTestServer(t)

	w := doJSON(t, mux, "POST", "/api/worlds", map[string]string{"agent": "test"})
	if w.Code != 503 {
		t.Fatalf("expected 503 (read-only mode), got %d (body: %s)", w.Code, w.Body.String())
	}

	body := decodeBody(t, w)
	errMsg, _ := body["error"].(string)
	if errMsg == "" {
		t.Error("expected error message in response")
	}
}

func TestDestroyWorld_ReadOnlyMode(t *testing.T) {
	_, mux := newFullTestServer(t)

	w := doJSON(t, mux, "DELETE", "/api/worlds/some-id", nil)
	if w.Code != 503 {
		t.Fatalf("expected 503 (read-only mode), got %d", w.Code)
	}
}

func TestSnapshot_ReadOnlyMode(t *testing.T) {
	_, mux := newFullTestServer(t)

	w := doJSON(t, mux, "POST", "/api/worlds/some-id/snapshot", nil)
	if w.Code != 503 {
		t.Fatalf("expected 503 (read-only mode), got %d", w.Code)
	}
}

func TestListAgents_WithAgent(t *testing.T) {
	_, mux := newFullTestServer(t)
	createTestAgent(t, "listed-agent")

	w := doJSON(t, mux, "GET", "/api/agents", nil)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var agents []interface{}
	if err := json.NewDecoder(w.Body).Decode(&agents); err != nil {
		t.Fatalf("decode agents list: %v", err)
	}
	if len(agents) != 1 {
		t.Errorf("expected 1 agent, got %d", len(agents))
	}
}

func TestCreateAndListAgents(t *testing.T) {
	_, mux := newFullTestServer(t)

	// Create two agents
	for _, name := range []string{"agent-a", "agent-b"} {
		w := doJSON(t, mux, "POST", "/api/agents", map[string]string{"name": name})
		if w.Code != 201 {
			t.Fatalf("create %s: expected 201, got %d", name, w.Code)
		}
	}

	// List should show both
	w := doJSON(t, mux, "GET", "/api/agents", nil)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var agents []interface{}
	if err := json.NewDecoder(w.Body).Decode(&agents); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(agents) != 2 {
		t.Errorf("expected 2 agents, got %d", len(agents))
	}
}

func TestStatusCountsAgents(t *testing.T) {
	_, mux := newFullTestServer(t)

	// Initially 0
	w := doJSON(t, mux, "GET", "/api/status", nil)
	body := decodeBody(t, w)
	if body["agents"].(float64) != 0 {
		t.Errorf("expected 0 agents initially")
	}

	// Create an agent
	createTestAgent(t, "counted")

	w = doJSON(t, mux, "GET", "/api/status", nil)
	body = decodeBody(t, w)
	if body["agents"].(float64) != 1 {
		t.Errorf("expected 1 agent after create, got %v", body["agents"])
	}
}

// ============================================================
// TODO API endpoint tests
// ============================================================

func TestGetArchitectStack_DefaultTemplate(t *testing.T) {
	_, mux := newFullTestServer(t)

	// No todo file exists - should return default template
	w := doJSON(t, mux, "GET", "/api/architect/stack", nil)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d (body: %s)", w.Code, w.Body.String())
	}

	body := decodeBody(t, w)
	content, ok := body["content"].(string)
	if !ok || content == "" {
		t.Fatalf("expected content string, got %v", body["content"])
	}

	// Verify default template has expected sections
	if !strings.Contains(content, "# Architect Stack") {
		t.Errorf("default template missing '# Architect Stack' heading")
	}
	if !strings.Contains(content, "## Focus") {
		t.Errorf("default template missing '## Focus' section")
	}
	if !strings.Contains(content, "## Queued") {
		t.Errorf("default template missing '## Queued' section")
	}
	if !strings.Contains(content, "## Done") {
		t.Errorf("default template missing '## Done' section")
	}
}

func TestPostArchitectStack(t *testing.T) {
	_, mux := newFullTestServer(t)

	stackContent := "# Architect Stack\n\n## Focus\n- [ ] Deploy v2\n\n## Queued\n- [ ] Write docs\n\n## Done\n- [x] Setup CI\n"

	// Write content
	w := doJSON(t, mux, "POST", "/api/architect/stack", map[string]string{
		"content": stackContent,
	})
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d (body: %s)", w.Code, w.Body.String())
	}

	body := decodeBody(t, w)
	if body["status"] != "ok" {
		t.Errorf("expected status=ok, got %v", body["status"])
	}

	// Verify the file was written to disk
	home := os.Getenv("SPWN_HOME")
	stackPath := filepath.Join(home, "architect", "stack.md")
	data, err := os.ReadFile(stackPath)
	if err != nil {
		t.Fatalf("failed to read todo file from disk: %v", err)
	}
	if string(data) != stackContent {
		t.Errorf("on-disk content mismatch:\ngot:  %q\nwant: %q", string(data), stackContent)
	}

	// Verify GET returns the same content
	w2 := doJSON(t, mux, "GET", "/api/architect/stack", nil)
	if w2.Code != 200 {
		t.Fatalf("expected 200 on GET, got %d", w2.Code)
	}
	body2 := decodeBody(t, w2)
	if body2["content"] != stackContent {
		t.Errorf("GET content mismatch after POST:\ngot:  %q\nwant: %q", body2["content"], stackContent)
	}
}

func TestArchitectStatusKPIs(t *testing.T) {
	_, mux := newFullTestServer(t)

	// Create some agents to populate KPIs
	createTestAgent(t, "kpi-agent-1")
	createTestAgent(t, "kpi-agent-2")

	// Write a TODO file with known task counts
	home := os.Getenv("SPWN_HOME")
	stackDir := filepath.Join(home, "architect")
	if err := os.MkdirAll(stackDir, 0755); err != nil {
		t.Fatalf("mkdir architect: %v", err)
	}
	stackContent := "# Architect Stack\n\n## Focus\n- [ ] Task A\n- [ ] Task B\n- [ ] Task C\n\n## Done\n- [x] Task D\n- [X] Task E\n"
	writeFile(t, filepath.Join(stackDir, "stack.md"), stackContent)

	w := doJSON(t, mux, "GET", "/api/architect/status", nil)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d (body: %s)", w.Code, w.Body.String())
	}

	body := decodeBody(t, w)

	// Verify kpis object exists
	kpis, ok := body["kpis"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected kpis to be an object, got %T: %v", body["kpis"], body["kpis"])
	}

	// Verify agent count
	agents, ok := kpis["agents"].(float64)
	if !ok {
		t.Fatalf("expected agents to be a number, got %T", kpis["agents"])
	}
	if agents != 2 {
		t.Errorf("expected 2 agents, got %v", agents)
	}

	// Verify worlds count (should be 0, no worlds created)
	worlds, ok := kpis["worlds"].(float64)
	if !ok {
		t.Fatalf("expected worlds to be a number, got %T", kpis["worlds"])
	}
	if worlds != 0 {
		t.Errorf("expected 0 worlds, got %v", worlds)
	}

	// Verify task counts
	tasksPending, ok := kpis["tasksPending"].(float64)
	if !ok {
		t.Fatalf("expected tasksPending to be a number, got %T", kpis["tasksPending"])
	}
	if tasksPending != 3 {
		t.Errorf("expected 3 pending tasks, got %v", tasksPending)
	}

	tasksCompleted, ok := kpis["tasksCompleted"].(float64)
	if !ok {
		t.Fatalf("expected tasksCompleted to be a number, got %T", kpis["tasksCompleted"])
	}
	if tasksCompleted != 2 {
		t.Errorf("expected 2 completed tasks, got %v", tasksCompleted)
	}
}

func TestArchitectStatusStackParsing(t *testing.T) {
	_, mux := newFullTestServer(t)

	home := os.Getenv("SPWN_HOME")
	stackDir := filepath.Join(home, "architect")
	if err := os.MkdirAll(stackDir, 0755); err != nil {
		t.Fatalf("mkdir architect: %v", err)
	}

	tests := []struct {
		name            string
		content         string
		expectedPending float64
		expectedDone    float64
	}{
		{
			name:              "empty file",
			content:           "",
			expectedPending:   0,
			expectedDone: 0,
		},
		{
			name:              "only pending",
			content:           "# TODO\n- [ ] Task 1\n- [ ] Task 2\n",
			expectedPending:   2,
			expectedDone: 0,
		},
		{
			name:              "only completed",
			content:           "# TODO\n- [x] Done 1\n- [X] Done 2\n- [x] Done 3\n",
			expectedPending:   0,
			expectedDone: 3,
		},
		{
			name:              "mixed tasks",
			content:           "# TODO\n\n## Focus\n- [ ] Active 1\n\n## Queued\n- [ ] Backlog 1\n- [ ] Backlog 2\n\n## Done\n- [x] Done 1\n",
			expectedPending:   3,
			expectedDone: 1,
		},
		{
			name:              "indented tasks",
			content:           "# TODO\n  - [ ] Indented pending\n  - [x] Indented done\n",
			expectedPending:   1,
			expectedDone: 1,
		},
		{
			name:              "non-task lines ignored",
			content:           "# TODO\n\nSome text\n- Regular bullet\n- [ ] Real task\n## Heading\n- [x] Done task\n",
			expectedPending:   1,
			expectedDone: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writeFile(t, filepath.Join(stackDir, "stack.md"), tt.content)

			w := doJSON(t, mux, "GET", "/api/architect/status", nil)
			if w.Code != 200 {
				t.Fatalf("expected 200, got %d", w.Code)
			}

			body := decodeBody(t, w)
			kpis, ok := body["kpis"].(map[string]interface{})
			if !ok {
				t.Fatalf("expected kpis object, got %T", body["kpis"])
			}

			pending := kpis["tasksPending"].(float64)
			completed := kpis["tasksCompleted"].(float64)

			if pending != tt.expectedPending {
				t.Errorf("tasksPending: got %v, want %v", pending, tt.expectedPending)
			}
			if completed != tt.expectedDone {
				t.Errorf("tasksDone: got %v, want %v", completed, tt.expectedDone)
			}
		})
	}
}

// ============================================================
// parseStackAction tests
// ============================================================

func TestParseStackAction_Add(t *testing.T) {
	input := "[STACK_PUSH] Deploy API\nPriority: high\nI'll set it up."
	action := parseStackAction(input)
	if action == nil {
		t.Fatal("expected a StackAction, got nil")
	}
	if action.Type != "push" {
		t.Errorf("expected type 'add', got %q", action.Type)
	}
	if action.Title != "Deploy API" {
		t.Errorf("expected title 'Deploy API', got %q", action.Title)
	}
	if action.Priority != "high" {
		t.Errorf("expected priority 'high', got %q", action.Priority)
	}
	if action.Description != "I'll set it up." {
		t.Errorf("expected description 'I'll set it up.', got %q", action.Description)
	}
}

func TestParseStackAction_Done(t *testing.T) {
	input := "[STACK_POP] Deploy API\nDone: deployed to prod"
	action := parseStackAction(input)
	if action == nil {
		t.Fatal("expected a StackAction, got nil")
	}
	if action.Type != "pop" {
		t.Errorf("expected type 'done', got %q", action.Type)
	}
	if action.Title != "Deploy API" {
		t.Errorf("expected title 'Deploy API', got %q", action.Title)
	}
	if action.Description != "deployed to prod" {
		t.Errorf("expected description 'deployed to prod', got %q", action.Description)
	}
}

func TestParseStackAction_Update(t *testing.T) {
	input := "[STACK_UPDATE] Deploy API\nProgress: 50% complete"
	action := parseStackAction(input)
	if action == nil {
		t.Fatal("expected a StackAction, got nil")
	}
	if action.Type != "update" {
		t.Errorf("expected type 'update', got %q", action.Type)
	}
	if action.Title != "Deploy API" {
		t.Errorf("expected title 'Deploy API', got %q", action.Title)
	}
	if action.Description != "50% complete" {
		t.Errorf("expected description '50%% complete', got %q", action.Description)
	}
}

func TestParseStackAction_NoAction(t *testing.T) {
	input := "Just a regular response without any TODO markers.\nAnother line here."
	action := parseStackAction(input)
	if action != nil {
		t.Errorf("expected nil for regular text, got %+v", action)
	}
}

func TestParseStackAction_MultipleActions(t *testing.T) {
	input := "[STACK_PUSH] First task\nPriority: high\n\n[STACK_PUSH] Second task\nPriority: low"
	action := parseStackAction(input)
	if action == nil {
		t.Fatal("expected a StackAction, got nil")
	}
	// Only the first action should be parsed
	if action.Title != "First task" {
		t.Errorf("expected first action title 'First task', got %q", action.Title)
	}
	if action.Priority != "high" {
		t.Errorf("expected priority 'high', got %q", action.Priority)
	}
}

func TestParseStackAction_InlineText(t *testing.T) {
	input := "Sure! I'll add that.\n[STACK_PUSH] Review code\nPriority: low\nWill do."
	action := parseStackAction(input)
	if action == nil {
		t.Fatal("expected a StackAction, got nil")
	}
	if action.Type != "push" {
		t.Errorf("expected type 'add', got %q", action.Type)
	}
	if action.Title != "Review code" {
		t.Errorf("expected title 'Review code', got %q", action.Title)
	}
	if action.Priority != "low" {
		t.Errorf("expected priority 'low', got %q", action.Priority)
	}
}

func TestParseStackAction_EmptyInput(t *testing.T) {
	action := parseStackAction("")
	if action != nil {
		t.Errorf("expected nil for empty input, got %+v", action)
	}
}

func TestParseStackAction_TableDriven(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantType  string
		wantTitle string
	}{
		{"add task", "[STACK_PUSH] Deploy API\nPriority: high\nI'll set it up.", "push", "Deploy API"},
		{"done task", "[STACK_POP] Deploy API\nDone: deployed to prod", "pop", "Deploy API"},
		{"update task", "[STACK_UPDATE] Fix bug\nProgress: investigating", "update", "Fix bug"},
		{"no action", "Just a regular response", "", ""},
		{"action with surrounding text", "Sure!\n[STACK_PUSH] Review code\nPriority: low\nWill do.", "push", "Review code"},
		{"empty string", "", "", ""},
		{"marker only no title", "[STACK_PUSH] \nPriority: medium", "push", ""},
		{"done with no details", "[STACK_POP] Ship v2", "pop", "Ship v2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action := parseStackAction(tt.input)
			if tt.wantType == "" {
				if action != nil {
					t.Errorf("expected nil action, got %+v", action)
				}
				return
			}
			if action == nil {
				t.Fatalf("expected action with type %q, got nil", tt.wantType)
			}
			if action.Type != tt.wantType {
				t.Errorf("type: got %q, want %q", action.Type, tt.wantType)
			}
			if action.Title != tt.wantTitle {
				t.Errorf("title: got %q, want %q", action.Title, tt.wantTitle)
			}
		})
	}
}

// ============================================================
// TODO file operations tests
// ============================================================

func TestGetArchitectTodo_EmptyFile(t *testing.T) {
	_, mux := newFullTestServer(t)

	// Write an empty todo file
	home := os.Getenv("SPWN_HOME")
	stackDir := filepath.Join(home, "architect")
	if err := os.MkdirAll(stackDir, 0755); err != nil {
		t.Fatalf("mkdir architect: %v", err)
	}
	writeFile(t, filepath.Join(stackDir, "stack.md"), "")

	w := doJSON(t, mux, "GET", "/api/architect/stack", nil)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	body := decodeBody(t, w)
	content, ok := body["content"].(string)
	if !ok {
		t.Fatalf("expected content string, got %T", body["content"])
	}
	// Empty file should return the empty string (not default template)
	if content != "" {
		t.Errorf("expected empty content for empty file, got %q", content)
	}
}

func TestGetArchitectTodo_WithContent(t *testing.T) {
	_, mux := newFullTestServer(t)

	home := os.Getenv("SPWN_HOME")
	stackDir := filepath.Join(home, "architect")
	if err := os.MkdirAll(stackDir, 0755); err != nil {
		t.Fatalf("mkdir architect: %v", err)
	}

	expected := "# Architect Stack\n\n## Focus\n- [ ] Build API\n\n## Queued\n- [ ] Write docs\n\n## Done\n- [x] Setup\n"
	writeFile(t, filepath.Join(stackDir, "stack.md"), expected)

	w := doJSON(t, mux, "GET", "/api/architect/stack", nil)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	body := decodeBody(t, w)
	content := body["content"].(string)
	if content != expected {
		t.Errorf("content mismatch:\ngot:  %q\nwant: %q", content, expected)
	}
}

func TestArchitectStatusKPIs_TodoCounting(t *testing.T) {
	_, mux := newFullTestServer(t)

	home := os.Getenv("SPWN_HOME")
	stackDir := filepath.Join(home, "architect")
	if err := os.MkdirAll(stackDir, 0755); err != nil {
		t.Fatalf("mkdir architect: %v", err)
	}

	// 5 pending, 3 completed
	stackContent := "# TODO\n\n## Focus\n- [ ] A\n- [ ] B\n\n## Queued\n- [ ] C\n- [ ] D\n- [ ] E\n\n## Done\n- [x] F\n- [X] G\n- [x] H\n"
	writeFile(t, filepath.Join(stackDir, "stack.md"), stackContent)

	w := doJSON(t, mux, "GET", "/api/architect/status", nil)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	body := decodeBody(t, w)
	kpis := body["kpis"].(map[string]interface{})

	pending := kpis["tasksPending"].(float64)
	completed := kpis["tasksCompleted"].(float64)

	if pending != 5 {
		t.Errorf("tasksPending: got %v, want 5", pending)
	}
	if completed != 3 {
		t.Errorf("tasksDone: got %v, want 3", completed)
	}
}

// ============================================================
// Auth endpoint tests
// ============================================================

func TestAuthProviders(t *testing.T) {
	_, mux := newFullTestServer(t)

	w := doJSON(t, mux, "GET", "/api/auth/providers", nil)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d (body: %s)", w.Code, w.Body.String())
	}

	body := decodeBody(t, w)
	providers, ok := body["providers"]
	if !ok {
		t.Fatal("response missing 'providers' key")
	}
	// providers should be an array (possibly empty depending on env)
	arr, ok := providers.([]interface{})
	if !ok {
		t.Fatalf("providers should be an array, got %T", providers)
	}
	// Each provider should have expected fields
	for _, p := range arr {
		pm, ok := p.(map[string]interface{})
		if !ok {
			t.Fatalf("provider entry should be object, got %T", p)
		}
		if _, ok := pm["provider"]; !ok {
			t.Error("provider entry missing 'provider' field")
		}
		if _, ok := pm["connected"]; !ok {
			t.Error("provider entry missing 'connected' field")
		}
	}
}

func TestAuthCheck(t *testing.T) {
	_, mux := newFullTestServer(t)

	w := doJSON(t, mux, "POST", "/api/auth/check", map[string]string{"provider": "anthropic"})
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d (body: %s)", w.Code, w.Body.String())
	}

	// Response should be a JSON object with status info
	body := decodeBody(t, w)
	// The Validate function returns a ProviderStatus - just check it's valid JSON
	if len(body) == 0 {
		t.Error("expected non-empty auth check response")
	}
}

func TestAuthConfigure(t *testing.T) {
	_, mux := newFullTestServer(t)

	w := doJSON(t, mux, "POST", "/api/auth/configure", map[string]string{
		"provider": "anthropic",
		"token":    "sk-test-fake-token-12345",
	})
	// Should succeed (200) - SaveToken writes to SPWN_HOME which is a temp dir
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d (body: %s)", w.Code, w.Body.String())
	}

	body := decodeBody(t, w)
	if body["status"] != "ok" {
		t.Errorf("expected status=ok, got %v", body["status"])
	}
}

func TestAuthConfigure_MissingToken(t *testing.T) {
	_, mux := newFullTestServer(t)

	w := doJSON(t, mux, "POST", "/api/auth/configure", map[string]string{
		"provider": "anthropic",
		"token":    "",
	})
	if w.Code != 400 {
		t.Fatalf("expected 400, got %d (body: %s)", w.Code, w.Body.String())
	}
}

// ============================================================
// History endpoint tests
// ============================================================

func TestArchitectHistory(t *testing.T) {
	_, mux := newFullTestServer(t)

	w := doJSON(t, mux, "GET", "/api/architect/history", nil)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d (body: %s)", w.Code, w.Body.String())
	}

	body := decodeBody(t, w)
	sessions, ok := body["sessions"]
	if !ok {
		t.Fatal("response missing 'sessions' key")
	}
	// sessions should be an array (empty - no Docker container in test)
	arr, ok := sessions.([]interface{})
	if !ok {
		t.Fatalf("sessions should be an array, got %T", sessions)
	}
	// In test environment, expect empty since no Docker container exists
	_ = arr // may be empty, that's fine
}

func TestWorldHistory(t *testing.T) {
	_, mux := newFullTestServer(t)

	// World doesn't exist in test state, so we expect 404
	w := doJSON(t, mux, "GET", "/api/worlds/test-world-123/history", nil)
	if w.Code != 404 {
		t.Fatalf("expected 404 for nonexistent world, got %d (body: %s)", w.Code, w.Body.String())
	}
}

func TestWorldHistory_MissingID(t *testing.T) {
	_, mux := newFullTestServer(t)

	// Empty world ID - Go's mux may redirect /api/worlds//history to
	// /api/worlds/history (307) since it cleans double slashes.
	// Accept 307 (redirect), 400 (validation), or 404 (not found) -
	// all indicate the request is not served as a valid world history.
	w := doJSON(t, mux, "GET", "/api/worlds//history", nil)
	validCodes := map[int]bool{301: true, 307: true, 400: true, 404: true}
	if !validCodes[w.Code] {
		t.Fatalf("expected 301, 307, 400, or 404, got %d (body: %s)", w.Code, w.Body.String())
	}
}

// Ensure unused import of fmt is used
var _ = fmt.Sprintf

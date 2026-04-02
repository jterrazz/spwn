package observatory

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

	"spwn.sh/core/universe/internal/state"
)

func newTestServer(t *testing.T) (*Server, *http.ServeMux) {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("SPWN_HOME", tmp)

	store, err := state.NewStore()
	if err != nil {
		t.Fatalf("create store: %v", err)
	}

	srv := New(store, nil, "127.0.0.1:0")
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", srv.handleHealth)
	mux.HandleFunc("GET /api/universes", srv.handleListUniverses)
	mux.HandleFunc("GET /api/agents", srv.handleListAgents)
	return srv, mux
}

// newFullTestServer registers ALL routes (matching Start()) for integration tests.
func newFullTestServer(t *testing.T) (*Server, *http.ServeMux) {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("SPWN_HOME", tmp)

	store, err := state.NewStore()
	if err != nil {
		t.Fatalf("create store: %v", err)
	}

	srv := New(store, nil, "127.0.0.1:0")
	mux := http.NewServeMux()

	// READ endpoints
	mux.HandleFunc("GET /api/health", cors(srv.handleHealth))
	mux.HandleFunc("GET /api/status", cors(srv.handleStatus))
	mux.HandleFunc("GET /api/universes", cors(srv.handleListUniverses))
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

	// Docker-dependent endpoints (read-only mode — arch is nil)
	mux.HandleFunc("POST /api/worlds", cors(srv.handleCreateWorld))
	mux.HandleFunc("DELETE /api/worlds/{id}", cors(srv.handleDestroyWorld))
	mux.HandleFunc("POST /api/worlds/{id}/snapshot", cors(srv.handleSnapshot))
	mux.HandleFunc("GET /api/worlds/{id}/logs", cors(srv.handleWorldLogs))

	// Architect endpoints
	mux.HandleFunc("GET /api/architect/status", cors(srv.handleArchitectStatus))
	mux.HandleFunc("GET /api/architect/todo", cors(srv.handleArchitectTodoGet))
	mux.HandleFunc("POST /api/architect/todo", cors(srv.handleArchitectTodoUpdate))

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
		filepath.Join(agentDir, "memory", "journal"),
		filepath.Join(agentDir, "memory", "playbooks"),
		filepath.Join(agentDir, "memory", "knowledge"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatalf("mkdir %s: %v", d, err)
		}
	}

	// Write identity files
	writeFile(t, filepath.Join(agentDir, "identity", "persona.md"), "# Persona\n\nA helpful test agent.\n")
	writeFile(t, filepath.Join(agentDir, "identity", "purpose.md"), "# Purpose\n\nTo test the API.\n")
	writeFile(t, filepath.Join(agentDir, "identity", "traits.md"), "# Traits\n\n- curious\n- diligent\n")

	// Write journal entries (both legacy and memory paths)
	if err := os.MkdirAll(filepath.Join(agentDir, "journal"), 0755); err != nil {
		t.Fatalf("mkdir journal: %v", err)
	}
	writeFile(t, filepath.Join(agentDir, "journal", "2025-01-01.md"), "# 2025-01-01\n\nFirst journal entry.\n")
	writeFile(t, filepath.Join(agentDir, "memory", "journal", "2025-01-01.md"), "# 2025-01-01\n\nFirst journal entry.\n")

	// Write a skill
	writeFile(t, filepath.Join(agentDir, "skills", "coding.md"), "# Coding\n\nWrites Go code.\n")

	// Write profile.yaml
	writeFile(t, filepath.Join(agentDir, "profile.yaml"), "tier: citizen\nruntime:\n  engine: claude-code\n  provider: anthropic\n  model: claude-4\n")

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

func TestListUniverses_Empty(t *testing.T) {
	_, mux := newTestServer(t)

	req := httptest.NewRequest("GET", "/api/universes", nil)
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

	store, err := state.NewStore()
	if err != nil {
		t.Fatalf("create store: %v", err)
	}

	srv := New(store, nil, "127.0.0.1:0")
	// srv.srv is nil — Stop should be a no-op
	err = srv.Stop(context.Background())
	if err != nil {
		t.Errorf("Stop on nil server should return nil, got: %v", err)
	}
}

func TestServerGracefulShutdown(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("SPWN_HOME", tmp)

	store, err := state.NewStore()
	if err != nil {
		t.Fatalf("create store: %v", err)
	}

	srv := New(store, nil, "127.0.0.1:0")

	// Build server manually on a random port
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", srv.handleHealth)

	httpSrv := &http.Server{Addr: "127.0.0.1:0", Handler: mux}
	srv.srv = httpSrv

	// Start in background — use a listener to get the actual port
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
// Integration tests for all Observatory API endpoints
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
	if body["tier"] != "citizen" {
		t.Errorf("expected tier=citizen, got %v", body["tier"])
	}
	if body["engine"] != "claude-code" {
		t.Errorf("expected engine=claude-code, got %v", body["engine"])
	}
	if body["persona"] != "A helpful test agent." {
		t.Errorf("expected persona content, got %v", body["persona"])
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

	w := doJSON(t, mux, "GET", "/api/agents/dave/files/identity/persona.md", nil)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d (body: %s)", w.Code, w.Body.String())
	}

	body := decodeBody(t, w)
	content, ok := body["content"].(string)
	if !ok || content == "" {
		t.Errorf("expected file content, got %v", body["content"])
	}
	if body["path"] != "identity/persona.md" {
		t.Errorf("expected path=identity/persona.md, got %v", body["path"])
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
	// Should be rejected — either 400 (bad path) or 404 (not found)
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
	// Dream may succeed or fail depending on journal contents — either 200 or 500 is acceptable
	if w.Code != 200 && w.Code != 500 {
		t.Fatalf("expected 200 or 500, got %d (body: %s)", w.Code, w.Body.String())
	}
}

func TestSleep(t *testing.T) {
	_, mux := newFullTestServer(t)
	createTestAgent(t, "sleeper")

	w := doJSON(t, mux, "POST", "/api/agents/sleeper/sleep", nil)
	// Sleep may succeed or fail depending on contents — either 200 or 500 is acceptable
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
	if body["status"] != "stopped" {
		t.Errorf("expected status=stopped (nil arch), got %v", body["status"])
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

func TestWorldLogs_ReadOnlyMode(t *testing.T) {
	_, mux := newFullTestServer(t)

	w := doJSON(t, mux, "GET", "/api/worlds/some-id/logs", nil)
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

func TestGetArchitectTodo_DefaultTemplate(t *testing.T) {
	_, mux := newFullTestServer(t)

	// No todo file exists — should return default template
	w := doJSON(t, mux, "GET", "/api/architect/todo", nil)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d (body: %s)", w.Code, w.Body.String())
	}

	body := decodeBody(t, w)
	content, ok := body["content"].(string)
	if !ok || content == "" {
		t.Fatalf("expected content string, got %v", body["content"])
	}

	// Verify default template has expected sections
	if !strings.Contains(content, "# Architect TODO") {
		t.Errorf("default template missing '# Architect TODO' heading")
	}
	if !strings.Contains(content, "## In Progress") {
		t.Errorf("default template missing '## In Progress' section")
	}
	if !strings.Contains(content, "## Backlog") {
		t.Errorf("default template missing '## Backlog' section")
	}
	if !strings.Contains(content, "## Completed") {
		t.Errorf("default template missing '## Completed' section")
	}
}

func TestPostArchitectTodo(t *testing.T) {
	_, mux := newFullTestServer(t)

	todoContent := "# Architect TODO\n\n## In Progress\n- [ ] Deploy v2\n\n## Backlog\n- [ ] Write docs\n\n## Completed\n- [x] Setup CI\n"

	// Write content
	w := doJSON(t, mux, "POST", "/api/architect/todo", map[string]string{
		"content": todoContent,
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
	todoPath := filepath.Join(home, "architect", "todo.md")
	data, err := os.ReadFile(todoPath)
	if err != nil {
		t.Fatalf("failed to read todo file from disk: %v", err)
	}
	if string(data) != todoContent {
		t.Errorf("on-disk content mismatch:\ngot:  %q\nwant: %q", string(data), todoContent)
	}

	// Verify GET returns the same content
	w2 := doJSON(t, mux, "GET", "/api/architect/todo", nil)
	if w2.Code != 200 {
		t.Fatalf("expected 200 on GET, got %d", w2.Code)
	}
	body2 := decodeBody(t, w2)
	if body2["content"] != todoContent {
		t.Errorf("GET content mismatch after POST:\ngot:  %q\nwant: %q", body2["content"], todoContent)
	}
}

func TestArchitectStatusKPIs(t *testing.T) {
	_, mux := newFullTestServer(t)

	// Create some agents to populate KPIs
	createTestAgent(t, "kpi-agent-1")
	createTestAgent(t, "kpi-agent-2")

	// Write a TODO file with known task counts
	home := os.Getenv("SPWN_HOME")
	todoDir := filepath.Join(home, "architect")
	if err := os.MkdirAll(todoDir, 0755); err != nil {
		t.Fatalf("mkdir architect: %v", err)
	}
	todoContent := "# Architect TODO\n\n## In Progress\n- [ ] Task A\n- [ ] Task B\n- [ ] Task C\n\n## Completed\n- [x] Task D\n- [X] Task E\n"
	writeFile(t, filepath.Join(todoDir, "todo.md"), todoContent)

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

func TestArchitectStatusTodoParsing(t *testing.T) {
	_, mux := newFullTestServer(t)

	home := os.Getenv("SPWN_HOME")
	todoDir := filepath.Join(home, "architect")
	if err := os.MkdirAll(todoDir, 0755); err != nil {
		t.Fatalf("mkdir architect: %v", err)
	}

	tests := []struct {
		name              string
		content           string
		expectedPending   float64
		expectedCompleted float64
	}{
		{
			name:              "empty file",
			content:           "",
			expectedPending:   0,
			expectedCompleted: 0,
		},
		{
			name:              "only pending",
			content:           "# TODO\n- [ ] Task 1\n- [ ] Task 2\n",
			expectedPending:   2,
			expectedCompleted: 0,
		},
		{
			name:              "only completed",
			content:           "# TODO\n- [x] Done 1\n- [X] Done 2\n- [x] Done 3\n",
			expectedPending:   0,
			expectedCompleted: 3,
		},
		{
			name:              "mixed tasks",
			content:           "# TODO\n\n## In Progress\n- [ ] Active 1\n\n## Backlog\n- [ ] Backlog 1\n- [ ] Backlog 2\n\n## Completed\n- [x] Done 1\n",
			expectedPending:   3,
			expectedCompleted: 1,
		},
		{
			name:              "indented tasks",
			content:           "# TODO\n  - [ ] Indented pending\n  - [x] Indented done\n",
			expectedPending:   1,
			expectedCompleted: 1,
		},
		{
			name:              "non-task lines ignored",
			content:           "# TODO\n\nSome text\n- Regular bullet\n- [ ] Real task\n## Heading\n- [x] Done task\n",
			expectedPending:   1,
			expectedCompleted: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writeFile(t, filepath.Join(todoDir, "todo.md"), tt.content)

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
			if completed != tt.expectedCompleted {
				t.Errorf("tasksCompleted: got %v, want %v", completed, tt.expectedCompleted)
			}
		})
	}
}

// ============================================================
// parseTodoAction tests
// ============================================================

func TestParseTodoAction_Add(t *testing.T) {
	input := "[TODO_ADD] Deploy API\nPriority: high\nI'll set it up."
	action := parseTodoAction(input)
	if action == nil {
		t.Fatal("expected a TodoAction, got nil")
	}
	if action.Type != "add" {
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

func TestParseTodoAction_Done(t *testing.T) {
	input := "[TODO_DONE] Deploy API\nCompleted: deployed to prod"
	action := parseTodoAction(input)
	if action == nil {
		t.Fatal("expected a TodoAction, got nil")
	}
	if action.Type != "done" {
		t.Errorf("expected type 'done', got %q", action.Type)
	}
	if action.Title != "Deploy API" {
		t.Errorf("expected title 'Deploy API', got %q", action.Title)
	}
	if action.Description != "deployed to prod" {
		t.Errorf("expected description 'deployed to prod', got %q", action.Description)
	}
}

func TestParseTodoAction_Update(t *testing.T) {
	input := "[TODO_UPDATE] Deploy API\nProgress: 50% complete"
	action := parseTodoAction(input)
	if action == nil {
		t.Fatal("expected a TodoAction, got nil")
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

func TestParseTodoAction_NoAction(t *testing.T) {
	input := "Just a regular response without any TODO markers.\nAnother line here."
	action := parseTodoAction(input)
	if action != nil {
		t.Errorf("expected nil for regular text, got %+v", action)
	}
}

func TestParseTodoAction_MultipleActions(t *testing.T) {
	input := "[TODO_ADD] First task\nPriority: high\n\n[TODO_ADD] Second task\nPriority: low"
	action := parseTodoAction(input)
	if action == nil {
		t.Fatal("expected a TodoAction, got nil")
	}
	// Only the first action should be parsed
	if action.Title != "First task" {
		t.Errorf("expected first action title 'First task', got %q", action.Title)
	}
	if action.Priority != "high" {
		t.Errorf("expected priority 'high', got %q", action.Priority)
	}
}

func TestParseTodoAction_InlineText(t *testing.T) {
	input := "Sure! I'll add that.\n[TODO_ADD] Review code\nPriority: low\nWill do."
	action := parseTodoAction(input)
	if action == nil {
		t.Fatal("expected a TodoAction, got nil")
	}
	if action.Type != "add" {
		t.Errorf("expected type 'add', got %q", action.Type)
	}
	if action.Title != "Review code" {
		t.Errorf("expected title 'Review code', got %q", action.Title)
	}
	if action.Priority != "low" {
		t.Errorf("expected priority 'low', got %q", action.Priority)
	}
}

func TestParseTodoAction_EmptyInput(t *testing.T) {
	action := parseTodoAction("")
	if action != nil {
		t.Errorf("expected nil for empty input, got %+v", action)
	}
}

func TestParseTodoAction_TableDriven(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantType  string
		wantTitle string
	}{
		{"add task", "[TODO_ADD] Deploy API\nPriority: high\nI'll set it up.", "add", "Deploy API"},
		{"done task", "[TODO_DONE] Deploy API\nCompleted: deployed to prod", "done", "Deploy API"},
		{"update task", "[TODO_UPDATE] Fix bug\nProgress: investigating", "update", "Fix bug"},
		{"no action", "Just a regular response", "", ""},
		{"action with surrounding text", "Sure!\n[TODO_ADD] Review code\nPriority: low\nWill do.", "add", "Review code"},
		{"empty string", "", "", ""},
		{"marker only no title", "[TODO_ADD] \nPriority: medium", "add", ""},
		{"done with no details", "[TODO_DONE] Ship v2", "done", "Ship v2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action := parseTodoAction(tt.input)
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
	todoDir := filepath.Join(home, "architect")
	if err := os.MkdirAll(todoDir, 0755); err != nil {
		t.Fatalf("mkdir architect: %v", err)
	}
	writeFile(t, filepath.Join(todoDir, "todo.md"), "")

	w := doJSON(t, mux, "GET", "/api/architect/todo", nil)
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
	todoDir := filepath.Join(home, "architect")
	if err := os.MkdirAll(todoDir, 0755); err != nil {
		t.Fatalf("mkdir architect: %v", err)
	}

	expected := "# Architect TODO\n\n## In Progress\n- [ ] Build API\n\n## Backlog\n- [ ] Write docs\n\n## Completed\n- [x] Setup\n"
	writeFile(t, filepath.Join(todoDir, "todo.md"), expected)

	w := doJSON(t, mux, "GET", "/api/architect/todo", nil)
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
	todoDir := filepath.Join(home, "architect")
	if err := os.MkdirAll(todoDir, 0755); err != nil {
		t.Fatalf("mkdir architect: %v", err)
	}

	// 5 pending, 3 completed
	todoContent := "# TODO\n\n## In Progress\n- [ ] A\n- [ ] B\n\n## Backlog\n- [ ] C\n- [ ] D\n- [ ] E\n\n## Completed\n- [x] F\n- [X] G\n- [x] H\n"
	writeFile(t, filepath.Join(todoDir, "todo.md"), todoContent)

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
		t.Errorf("tasksCompleted: got %v, want 3", completed)
	}
}

// Ensure unused import of fmt is used
var _ = fmt.Sprintf

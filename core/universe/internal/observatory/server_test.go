package observatory

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/jterrazz/spwn/core/universe/internal/state"
)

func TestHealthEndpoint(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("SPWN_HOME", tmp)

	store, err := state.NewStore()
	if err != nil {
		t.Fatalf("create store: %v", err)
	}

	srv := New(store, "127.0.0.1:13901")

	// Build the mux manually so we can test handlers
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", srv.handleHealth)
	mux.HandleFunc("GET /api/universes", srv.handleListUniverses)
	mux.HandleFunc("GET /api/agents", srv.handleListAgents)

	testSrv := &http.Server{Addr: "127.0.0.1:13901", Handler: mux}
	srv.srv = testSrv

	go func() {
		_ = testSrv.ListenAndServe()
	}()

	// Wait for server to be ready
	time.Sleep(100 * time.Millisecond)
	defer testSrv.Shutdown(context.Background())

	// Test /api/health
	resp, err := http.Get("http://127.0.0.1:13901/api/health")
	if err != nil {
		t.Fatalf("GET /api/health failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if body["status"] != "ok" {
		t.Fatalf("expected status ok, got %q", body["status"])
	}

	// Test /api/universes
	resp2, err := http.Get("http://127.0.0.1:13901/api/universes")
	if err != nil {
		t.Fatalf("GET /api/universes failed: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp2.StatusCode)
	}

	if ct := resp2.Header.Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected application/json, got %q", ct)
	}
}

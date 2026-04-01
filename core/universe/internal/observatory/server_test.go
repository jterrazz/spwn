package observatory

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
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

	srv := New(store, "127.0.0.1:0")
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", srv.handleHealth)
	mux.HandleFunc("GET /api/universes", srv.handleListUniverses)
	mux.HandleFunc("GET /api/agents", srv.handleListAgents)
	return srv, mux
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

	srv := New(store, "127.0.0.1:0")
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

	srv := New(store, "127.0.0.1:0")

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

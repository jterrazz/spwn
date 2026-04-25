package gate

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// fakeElement is a minimal Element used to test the registry +
// server routing without touching real upstreams.
type fakeElement struct {
	name     string
	handler  http.Handler
	refreshN int
	refreshE error
}

func (f *fakeElement) Name() string                      { return f.name }
func (f *fakeElement) Handler() http.Handler             { return f.handler }
func (f *fakeElement) Refresh(_ context.Context) error   { f.refreshN++; return f.refreshE }

func TestRegistry_AddRejectsDuplicates(t *testing.T) {
	r := NewRegistry()
	a := &fakeElement{name: "a", handler: http.NotFoundHandler()}
	if err := r.Add(a); err != nil {
		t.Fatalf("first add: %v", err)
	}
	if err := r.Add(a); err == nil {
		t.Fatalf("duplicate add should error")
	}
}

func TestRegistry_AddRejectsEmptyName(t *testing.T) {
	r := NewRegistry()
	if err := r.Add(&fakeElement{name: "", handler: http.NotFoundHandler()}); err == nil {
		t.Errorf("empty name should error")
	}
	if err := r.Add(nil); err == nil {
		t.Errorf("nil element should error")
	}
}

func TestServer_HealthzListsElements(t *testing.T) {
	r := NewRegistry()
	_ = r.Add(&fakeElement{name: "alpha", handler: http.NotFoundHandler()})
	_ = r.Add(&fakeElement{name: "beta", handler: http.NotFoundHandler()})

	s := NewServer(":0", r, nil)
	rec := httptest.NewRecorder()
	s.http.Handler.ServeHTTP(rec, httptest.NewRequest("GET", "/healthz", nil))

	if rec.Code != 200 {
		t.Fatalf("healthz status = %d, want 200", rec.Code)
	}
	var body struct {
		Status   string   `json:"status"`
		Elements []string `json:"elements"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body.Status != "ok" {
		t.Errorf("healthz status field = %q, want ok", body.Status)
	}
	if len(body.Elements) != 2 {
		t.Errorf("healthz lists %d elements, want 2", len(body.Elements))
	}
}

func TestServer_RoutesByElementName(t *testing.T) {
	r := NewRegistry()
	called := ""
	_ = r.Add(&fakeElement{
		name: "notion",
		handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			called = req.URL.Path
			w.WriteHeader(204)
		}),
	})

	s := NewServer(":0", r, nil)
	rec := httptest.NewRecorder()
	s.http.Handler.ServeHTTP(rec, httptest.NewRequest("POST", "/mcp/notion/initialize", strings.NewReader(`{}`)))

	if rec.Code != 204 {
		t.Errorf("status = %d, want 204", rec.Code)
	}
	if called != "/initialize" {
		t.Errorf("element saw path = %q, want %q (path should be stripped of /mcp/<name>)", called, "/initialize")
	}
}

func TestServer_404OnUnknownElement(t *testing.T) {
	s := NewServer(":0", NewRegistry(), nil)
	rec := httptest.NewRecorder()
	s.http.Handler.ServeHTTP(rec, httptest.NewRequest("GET", "/mcp/unknown/x", nil))
	if rec.Code != 404 {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestServer_404OnMissingName(t *testing.T) {
	s := NewServer(":0", NewRegistry(), nil)
	rec := httptest.NewRecorder()
	s.http.Handler.ServeHTTP(rec, httptest.NewRequest("GET", "/mcp/", nil))
	if rec.Code != 404 {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestScheduler_TicksRefreshOnEveryElement(t *testing.T) {
	r := NewRegistry()
	a := &fakeElement{name: "a", handler: http.NotFoundHandler()}
	b := &fakeElement{name: "b", handler: http.NotFoundHandler()}
	_ = r.Add(a)
	_ = r.Add(b)

	s := NewScheduler(r, 30*time.Millisecond, nil)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { s.Run(ctx); close(done) }()

	// Wait for at least one tick after the startup tick.
	time.Sleep(80 * time.Millisecond)
	cancel()
	<-done

	if a.refreshN < 2 || b.refreshN < 2 {
		t.Errorf("expected ≥2 refreshes per element (startup + 1 tick), got a=%d b=%d", a.refreshN, b.refreshN)
	}
}

func TestScheduler_PerElementErrorsDontBreakTheSweep(t *testing.T) {
	r := NewRegistry()
	bad := &fakeElement{name: "bad", handler: http.NotFoundHandler(), refreshE: errors.New("boom")}
	good := &fakeElement{name: "good", handler: http.NotFoundHandler()}
	_ = r.Add(bad)
	_ = r.Add(good)

	s := NewScheduler(r, time.Hour, nil) // long ticker; only startup tick will fire
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { s.Run(ctx); close(done) }()

	time.Sleep(20 * time.Millisecond)
	cancel()
	<-done

	if good.refreshN != 1 {
		t.Errorf("good element wasn't refreshed despite bad element erroring (good.n=%d, bad.n=%d)", good.refreshN, bad.refreshN)
	}
}

func TestServer_RunStopsOnContextCancel(t *testing.T) {
	s := NewServer(":0", NewRegistry(), nil)
	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() { errCh <- s.Run(ctx) }()

	// Give the listener a moment to bind, then cancel.
	time.Sleep(30 * time.Millisecond)
	cancel()

	select {
	case err := <-errCh:
		if err != context.Canceled {
			t.Errorf("Run returned %v, want context.Canceled", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run didn't exit after context cancel")
	}
}

// silence unused import linters when only some tests use them
var _ = io.EOF

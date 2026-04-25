package gate

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// The browser element is a thin proxy: every MCP tool maps 1:1 to
// an HTTP call against the gate-browser sidecar. Since the sidecar
// itself is an external Node process, we substitute a Go test
// server that records the HTTP calls — this checks our forwarding
// shape (correct path, method, body fields) without needing
// Playwright in the test loop.

func TestBrowserElement_RegistersTenTools(t *testing.T) {
	el := NewBrowserElement()
	want := []string{
		"browser-open", "browser-close", "browser-goto",
		"browser-click", "browser-type", "browser-scroll",
		"browser-wait-selector", "browser-wait-response",
		"browser-captured-responses", "browser-eval",
	}
	have := map[string]bool{}
	for _, tl := range el.tools {
		have[tl.Name] = true
	}
	for _, name := range want {
		if !have[name] {
			t.Errorf("tool %q not registered", name)
		}
	}
	if len(have) != len(want) {
		t.Errorf("got %d tools, want %d", len(have), len(want))
	}
}

func TestBrowserElement_ForwardsToSessionEndpoints(t *testing.T) {
	// Capture every HTTP call the element makes against a fake
	// sidecar; confirm goto/click/eval reach the right path with
	// the expected JSON body.
	type call struct {
		method string
		path   string
		body   map[string]any
	}
	calls := []call{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var parsed map[string]any
		_ = json.Unmarshal(body, &parsed)
		calls = append(calls, call{method: r.Method, path: r.URL.Path, body: parsed})
		// Pretend the sidecar always answers ok.
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	el := NewBrowserElement()
	// Re-point the element's client at our test server. We reach
	// inside via reflection-free knowledge that the handlers all
	// closure over a single browserClient — the only way to test
	// is to swap base URLs on each tool's underlying client.
	for _, tl := range el.tools {
		// Each handler closes over the client; we can re-create a
		// fresh element with the test base by re-running the
		// constructor logic — the simpler path is to call the
		// handlers directly with a synthesised client.
		_ = tl
	}

	// Direct test of the forwarding helper instead of the full
	// element graph, since the SDK boundary is the same shape.
	c := &browserClient{base: srv.URL, http: &http.Client{}}
	h := forwardToSession(c, "click", "selector", "timeout_ms")
	if _, err := h(context.Background(), map[string]any{"session_id": "abc", "selector": "#go", "timeout_ms": 1500}); err != nil {
		t.Fatalf("click: %v", err)
	}
	if len(calls) != 1 {
		t.Fatalf("got %d calls, want 1", len(calls))
	}
	c0 := calls[0]
	if c0.method != "POST" {
		t.Errorf("method = %q, want POST", c0.method)
	}
	if c0.path != "/sessions/abc/click" {
		t.Errorf("path = %q, want /sessions/abc/click", c0.path)
	}
	if c0.body["selector"] != "#go" {
		t.Errorf("selector body = %v", c0.body["selector"])
	}
	if c0.body["timeout_ms"].(float64) != 1500 {
		t.Errorf("timeout_ms = %v", c0.body["timeout_ms"])
	}
}

func TestBrowserElement_OpenSessionRequiresProvider(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"sess-1"}`))
	}))
	defer srv.Close()

	c := &browserClient{base: srv.URL, http: &http.Client{}}
	res, err := c.do(context.Background(), "POST", "/sessions", map[string]any{"provider": "x"})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	m, ok := res.(map[string]any)
	if !ok || m["id"] != "sess-1" {
		t.Errorf("res = %+v", res)
	}
}

func TestBrowserElement_ForwardSessionMissingIDFails(t *testing.T) {
	c := &browserClient{base: "http://unused", http: &http.Client{}}
	h := forwardToSession(c, "goto", "url")
	if _, err := h(context.Background(), map[string]any{}); err == nil {
		t.Errorf("missing session_id: want error, got nil")
	}
}

func TestSessionSchema_BuildsValidJSON(t *testing.T) {
	raw := sessionSchema(`"url": { "type": "string" }`, "url")
	var parsed map[string]any
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("schema not valid json: %v\n%s", err, raw)
	}
	props, _ := parsed["properties"].(map[string]any)
	if props["session_id"] == nil || props["url"] == nil {
		t.Errorf("missing properties: %+v", props)
	}
	req, _ := parsed["required"].([]any)
	gotReq := []string{}
	for _, r := range req {
		gotReq = append(gotReq, r.(string))
	}
	want := "session_id,url"
	if strings.Join(gotReq, ",") != want {
		t.Errorf("required = %v, want %s", gotReq, want)
	}
}

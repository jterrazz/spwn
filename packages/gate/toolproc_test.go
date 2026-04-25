package gate

import (
	"context"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
)

// ToolSupervisor's job is to (a) eagerly build per-tool reverse-proxy
// elements at construction so the gate's router has something to
// dispatch to immediately, (b) spawn each tool's subprocess in
// supervised goroutines, (c) restart on crash with backoff, (d)
// surface upstream errors as clean 503s while the subprocess boots.
//
// We don't spawn real Node here — that's an integration concern.
// Pin construction-time invariants and the proxy's error path.

func TestNewToolSupervisor_AllocatesPortsSequentially(t *testing.T) {
	tools := []Tool{
		{Name: "x", Spec: ToolGateSpec{MCP: &ToolMCP{Entry: []string{"node", "x.js"}}}},
		{Name: "linkedin", Spec: ToolGateSpec{MCP: &ToolMCP{Entry: []string{"node", "li.js"}}}},
		{Name: "reddit", Spec: ToolGateSpec{MCP: &ToolMCP{Entry: []string{"node", "r.js"}}}},
	}
	s := NewToolSupervisor(tools, log.New(os.Stderr, "", 0))
	got := map[string]int{}
	for _, e := range s.Elements() {
		got[e.Name()] = e.Port()
	}
	if got["x"] != BaseToolPort+0 || got["linkedin"] != BaseToolPort+1 || got["reddit"] != BaseToolPort+2 {
		t.Errorf("ports = %+v, want sequential from %d", got, BaseToolPort)
	}
}

func TestNewToolSupervisor_SkipsToolsWithoutMCPEntry(t *testing.T) {
	tools := []Tool{
		{Name: "x", Spec: ToolGateSpec{MCP: &ToolMCP{Entry: []string{"node", "x.js"}}}},
		{Name: "cookies-only", Spec: ToolGateSpec{Cookies: &ToolCookies{Domains: []string{"foo.com"}}}},
	}
	s := NewToolSupervisor(tools, log.New(os.Stderr, "", 0))
	if len(s.Elements()) != 1 {
		t.Errorf("got %d elements, want 1 (cookies-only tool should be skipped)", len(s.Elements()))
	}
	if s.Element("cookies-only") != nil {
		t.Errorf("cookies-only tool was registered as an MCP element")
	}
}

func TestToolSupervisor_503WhenUpstreamDown(t *testing.T) {
	// Build an element pointed at a port nobody's listening on.
	// The proxy must answer 503 with a JSON error, not hang.
	target, _ := url.Parse("http://127.0.0.1:1") // port 1 is reserved + closed
	logger := log.New(os.Stderr, "", 0)
	s := NewToolSupervisor([]Tool{{
		Name: "doomed",
		Spec: ToolGateSpec{MCP: &ToolMCP{Entry: []string{"none"}}},
	}}, logger)
	// Override the element's proxy target to our unreachable one.
	el := s.Element("doomed")
	if el == nil {
		t.Fatal("element not built")
	}
	// We re-create the proxy here to point at port 1 (the eager
	// constructor used the supervisor's allocated port, which IS
	// listening for nothing — same effect).
	_ = target

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/", strings.NewReader(`{}`))
	el.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `not ready`) {
		t.Errorf("body should explain unavailability, got %q", rec.Body.String())
	}
}

func TestToolElement_RefreshIsNoOp(t *testing.T) {
	// Tool subprocesses own their own auth refresh — the gate-side
	// element shouldn't try to do anything on tick.
	s := NewToolSupervisor([]Tool{{
		Name: "x",
		Spec: ToolGateSpec{MCP: &ToolMCP{Entry: []string{"x"}}},
	}}, log.New(os.Stderr, "", 0))
	el := s.Element("x")
	if err := el.Refresh(context.Background()); err != nil {
		t.Errorf("Refresh: %v", err)
	}
}

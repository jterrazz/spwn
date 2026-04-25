package gate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// NewBrowserElement returns the gate's generic browser MCP element.
// It thin-wraps the gate-browser sidecar's HTTP API so agents that
// need ad-hoc browsing (Stagehand-style escape hatch) have access
// to navigate / click / type / eval primitives without a dedicated
// catalog tool — useful for one-off scrapes or sites that don't
// justify a full tool yet.
//
// Sessions opened via this element load cookies for the named
// provider (must be a registered cookie-sync provider). To browse
// without cookies, pass provider:"" — the sidecar opens an empty
// context.
//
// Cost trade-off: each MCP call is one HTTP RTT to the sidecar +
// one Playwright op. Cheap for "click button, read result" loops;
// expensive for tight scrolling. For high-throughput scrapes,
// implement a hardcoded catalog tool instead.
func NewBrowserElement() *MCPServer {
	s := NewMCPServer("browser", "Browser (Playwright primitives)", "0.1.0")
	c := &browserClient{base: "http://127.0.0.1:9001", http: &http.Client{Timeout: 60 * time.Second}}

	s.AddTool(&MCPTool{
		Name:        "browser-open",
		Description: "Open a browser session. Pass `provider` to load that provider's session cookies (must be a registered cookie-sync provider, e.g. \"x\"); empty string = no cookies. Returns a session_id to pass to subsequent calls. Sessions auto-close after 5 min idle.",
		InputSchema: json.RawMessage(`{
            "type": "object",
            "properties": {
                "provider": { "type": "string", "description": "Cookie provider name, or empty for no cookies" }
            }
        }`),
		Handler: func(ctx context.Context, args map[string]any) (any, error) {
			provider, _ := args["provider"].(string)
			return c.do(ctx, "POST", "/sessions", map[string]any{"provider": provider})
		},
	})

	s.AddTool(&MCPTool{
		Name:        "browser-close",
		Description: "Close a session and free its browser context. Always call this when done — the auto-reaper is a safety net, not a substitute.",
		InputSchema: sessionSchema(""),
		Handler: func(ctx context.Context, args map[string]any) (any, error) {
			id, ok := args["session_id"].(string)
			if !ok || id == "" {
				return nil, fmt.Errorf("session_id is required")
			}
			return c.do(ctx, "DELETE", "/sessions/"+id, nil)
		},
	})

	s.AddTool(&MCPTool{
		Name:        "browser-goto",
		Description: "Navigate the session to a URL. wait_until = \"domcontentloaded\" (default), \"load\", or \"networkidle\".",
		InputSchema: sessionSchema(`"url": { "type": "string" }, "wait_until": { "type": "string" }`, "url"),
		Handler:     forwardToSession(c, "goto", "url", "wait_until"),
	})

	s.AddTool(&MCPTool{
		Name:        "browser-click",
		Description: "Click a CSS selector. Throws if the selector isn't visible within timeout_ms.",
		InputSchema: sessionSchema(`"selector": { "type": "string" }, "timeout_ms": { "type": "integer" }`, "selector"),
		Handler:     forwardToSession(c, "click", "selector", "timeout_ms"),
	})

	s.AddTool(&MCPTool{
		Name:        "browser-type",
		Description: "Click a selector and type text into it. Useful for forms.",
		InputSchema: sessionSchema(`"selector": { "type": "string" }, "text": { "type": "string" }`, "selector", "text"),
		Handler:     forwardToSession(c, "type", "selector", "text", "timeout_ms"),
	})

	s.AddTool(&MCPTool{
		Name:        "browser-scroll",
		Description: "Scroll the page. delta_y is pixels per scroll (default 4000), count is how many times (default 1), wait_ms is the delay between scrolls.",
		InputSchema: sessionSchema(`"delta_y": { "type": "integer" }, "count": { "type": "integer" }, "wait_ms": { "type": "integer" }`),
		Handler:     forwardToSession(c, "scroll", "delta_y", "count", "wait_ms"),
	})

	s.AddTool(&MCPTool{
		Name:        "browser-wait-selector",
		Description: "Wait for a selector to be visible (or attached). Throws on timeout.",
		InputSchema: sessionSchema(`"selector": { "type": "string" }, "state": { "type": "string", "enum": ["visible", "attached"] }, "timeout_ms": { "type": "integer" }`, "selector"),
		Handler:     forwardToSession(c, "wait-selector", "selector", "state", "timeout_ms"),
	})

	s.AddTool(&MCPTool{
		Name:        "browser-wait-response",
		Description: "Wait for the next HTTP response matching url_pattern (regex). Returns the parsed JSON body if Content-Type is JSON; otherwise the text. By default skips non-JSON responses (set allow_non_json:true to opt out).",
		InputSchema: sessionSchema(`"url_pattern": { "type": "string" }, "method": { "type": "string" }, "timeout_ms": { "type": "integer" }, "allow_non_json": { "type": "boolean" }`, "url_pattern"),
		Handler:     forwardToSession(c, "wait-response", "url_pattern", "method", "timeout_ms", "allow_non_json"),
	})

	s.AddTool(&MCPTool{
		Name:        "browser-captured-responses",
		Description: "Pull every JSON response captured since session start (or since since_ts) matching url_pattern. Use this to harvest GraphQL/XHR bodies after a navigation, without explicitly waiting for each one.",
		InputSchema: sessionSchema(`"url_pattern": { "type": "string" }, "since_ts": { "type": "integer" }`),
		Handler:     forwardToSession(c, "captured-responses", "url_pattern", "since_ts"),
	})

	s.AddTool(&MCPTool{
		Name:        "browser-eval",
		Description: "Run a JS expression in the page context. Returns the value (must be JSON-serializable). Useful for DOM scrapes the other primitives don't cover.",
		InputSchema: sessionSchema(`"script": { "type": "string" }`, "script"),
		Handler:     forwardToSession(c, "eval", "script"),
	})

	return s
}

// sessionSchema builds the inputSchema JSON for a tool that takes
// session_id plus any number of extra fields and required keys.
// Variadic args after the first string are required-key names.
func sessionSchema(extraProps string, required ...string) json.RawMessage {
	props := `"session_id": { "type": "string" }`
	if extraProps != "" {
		props += ", " + extraProps
	}
	req := []string{`"session_id"`}
	for _, r := range required {
		req = append(req, `"`+r+`"`)
	}
	reqJSON := "[" + joinComma(req) + "]"
	return json.RawMessage(`{"type":"object","properties":{` + props + `},"required":` + reqJSON + `}`)
}

func joinComma(in []string) string {
	out := ""
	for i, s := range in {
		if i > 0 {
			out += ","
		}
		out += s
	}
	return out
}

// forwardToSession returns a handler that POSTs to /sessions/:id/<op>
// with whichever fields are present in args. Keeps tool wiring to one
// line per method.
func forwardToSession(c *browserClient, op string, fields ...string) func(context.Context, map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		id, ok := args["session_id"].(string)
		if !ok || id == "" {
			return nil, fmt.Errorf("session_id is required")
		}
		body := make(map[string]any, len(fields))
		for _, f := range fields {
			if v, ok := args[f]; ok && v != nil {
				body[f] = v
			}
		}
		return c.do(ctx, "POST", "/sessions/"+id+"/"+op, body)
	}
}

// browserClient is a thin HTTP client for the in-container sidecar.
// Marshals body to JSON, parses response back to any.
type browserClient struct {
	base string
	http *http.Client
}

func (c *browserClient) do(ctx context.Context, method, path string, body any) (any, error) {
	var rdr io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		rdr = bytes.NewReader(buf)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, rdr)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sidecar %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("sidecar %s %s: %d %s", method, path, resp.StatusCode, bytes.TrimSpace(raw))
	}
	if len(raw) == 0 {
		return map[string]any{"ok": true}, nil
	}
	var parsed any
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return string(raw), nil
	}
	return parsed, nil
}

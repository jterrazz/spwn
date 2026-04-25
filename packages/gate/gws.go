package gate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
)

// gwsCredsPath is where the gate looks for the Google credentials
// file. The user runs `gws auth setup` + `gws auth export` on the
// host, drops the resulting credentials.json into ~/.spwn/gate/google/,
// and the gate's bind mount surfaces it here.
const gwsCredsPath = "/gate/google/credentials.json"

// runGws executes a gws subprocess with credentials wired and returns
// its stdout as a parsed JSON value. Falls back to raw text content
// when the response isn't JSON (gws errors are sometimes plain text).
func runGws(ctx context.Context, args ...string) (any, error) {
	if _, err := os.Stat(gwsCredsPath); err != nil {
		return nil, fmt.Errorf(
			"no Google credentials at %s — run on the host:\n"+
				"  gws auth setup\n"+
				"  gws auth login -s gmail,calendar\n"+
				"  mkdir -p ~/.spwn/gate/google\n"+
				"  gws auth export --unmasked > ~/.spwn/gate/google/credentials.json\n"+
				"then `spwn gate restart`",
			gwsCredsPath,
		)
	}

	cmd := exec.CommandContext(ctx, "gws", args...)
	cmd.Env = append(os.Environ(),
		"GOOGLE_WORKSPACE_CLI_CREDENTIALS_FILE="+gwsCredsPath,
	)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		// gws exit codes carry meaning, but for now just surface
		// stderr — the caller wraps as MCP isError=true content.
		return nil, fmt.Errorf("gws %v: %w (stderr: %s)", args, err, bytes.TrimSpace(stderr.Bytes()))
	}

	out := bytes.TrimSpace(stdout.Bytes())
	if len(out) == 0 {
		return map[string]any{"ok": true}, nil
	}

	// Try to parse as JSON; fall back to text.
	var parsed any
	if err := json.Unmarshal(out, &parsed); err == nil {
		return parsed, nil
	}
	return string(out), nil
}

// NewGmailElement returns a gate element exposing common Gmail tools
// over MCP, backed by the gws CLI. Tools are deliberately a small
// curated set covering the 90% case (search, read, draft, labels) —
// the gws surface is huge but most agents only need a handful.
func NewGmailElement() *MCPServer {
	s := NewMCPServer("gmail", "Gmail (via gws)", "0.1.0")

	s.AddTool(&MCPTool{
		Name:        "search-threads",
		Description: "Search Gmail threads with a Gmail query (e.g. \"label:newsletters newer_than:1d\").",
		InputSchema: json.RawMessage(`{
            "type": "object",
            "properties": {
                "query": { "type": "string", "description": "Gmail search query" },
                "limit": { "type": "integer", "description": "Max threads to return (default 10)" }
            },
            "required": ["query"]
        }`),
		Handler: func(ctx context.Context, args map[string]any) (any, error) {
			q, _ := args["query"].(string)
			limit := 10
			if l, ok := args["limit"].(float64); ok {
				limit = int(l)
			}
			params, _ := json.Marshal(map[string]any{"userId": "me", "q": q, "maxResults": limit})
			return runGws(ctx, "gmail", "users", "threads", "list", "--params", string(params))
		},
	})

	s.AddTool(&MCPTool{
		Name:        "get-thread",
		Description: "Fetch a Gmail thread by id with all its messages.",
		InputSchema: json.RawMessage(`{
            "type": "object",
            "properties": { "id": { "type": "string" } },
            "required": ["id"]
        }`),
		Handler: func(ctx context.Context, args map[string]any) (any, error) {
			id, _ := args["id"].(string)
			params, _ := json.Marshal(map[string]any{"userId": "me", "id": id})
			return runGws(ctx, "gmail", "users", "threads", "get", "--params", string(params))
		},
	})

	s.AddTool(&MCPTool{
		Name:        "list-drafts",
		Description: "List Gmail drafts.",
		InputSchema: json.RawMessage(`{
            "type": "object",
            "properties": { "limit": { "type": "integer" } }
        }`),
		Handler: func(ctx context.Context, args map[string]any) (any, error) {
			limit := 10
			if l, ok := args["limit"].(float64); ok {
				limit = int(l)
			}
			params, _ := json.Marshal(map[string]any{"userId": "me", "maxResults": limit})
			return runGws(ctx, "gmail", "users", "drafts", "list", "--params", string(params))
		},
	})

	s.AddTool(&MCPTool{
		Name:        "list-labels",
		Description: "List Gmail labels (for filtering search-threads queries).",
		Handler: func(ctx context.Context, _ map[string]any) (any, error) {
			return runGws(ctx, "gmail", "users", "labels", "list", "--params", `{"userId":"me"}`)
		},
	})

	return s
}

// NewGcalElement exposes Google Calendar tools via gws. Same pattern
// as Gmail: small curated tool set, gws backend.
func NewGcalElement() *MCPServer {
	s := NewMCPServer("gcal", "Google Calendar (via gws)", "0.1.0")

	s.AddTool(&MCPTool{
		Name:        "list-calendars",
		Description: "List the user's calendars.",
		Handler: func(ctx context.Context, _ map[string]any) (any, error) {
			return runGws(ctx, "calendar", "calendarList", "list")
		},
	})

	s.AddTool(&MCPTool{
		Name:        "list-events",
		Description: "List events on a calendar (default: primary). Use ISO 8601 timestamps.",
		InputSchema: json.RawMessage(`{
            "type": "object",
            "properties": {
                "calendar": { "type": "string", "description": "Calendar id (default \"primary\")" },
                "from":     { "type": "string", "description": "ISO 8601 lower bound" },
                "to":       { "type": "string", "description": "ISO 8601 upper bound" },
                "limit":    { "type": "integer" }
            }
        }`),
		Handler: func(ctx context.Context, args map[string]any) (any, error) {
			cal := "primary"
			if c, ok := args["calendar"].(string); ok && c != "" {
				cal = c
			}
			limit := 20
			if l, ok := args["limit"].(float64); ok {
				limit = int(l)
			}
			p := map[string]any{
				"calendarId":   cal,
				"maxResults":   limit,
				"singleEvents": true,
				"orderBy":      "startTime",
			}
			if v, ok := args["from"].(string); ok && v != "" {
				p["timeMin"] = v
			}
			if v, ok := args["to"].(string); ok && v != "" {
				p["timeMax"] = v
			}
			params, _ := json.Marshal(p)
			return runGws(ctx, "calendar", "events", "list", "--params", string(params))
		},
	})

	s.AddTool(&MCPTool{
		Name:        "create-event",
		Description: "Create a calendar event.",
		InputSchema: json.RawMessage(`{
            "type": "object",
            "properties": {
                "calendar": { "type": "string", "description": "Calendar id (default \"primary\")" },
                "title":    { "type": "string" },
                "start":    { "type": "string", "description": "ISO 8601 start" },
                "end":      { "type": "string", "description": "ISO 8601 end" }
            },
            "required": ["title", "start", "end"]
        }`),
		Handler: func(ctx context.Context, args map[string]any) (any, error) {
			cal := "primary"
			if c, ok := args["calendar"].(string); ok && c != "" {
				cal = c
			}
			title, _ := args["title"].(string)
			start, _ := args["start"].(string)
			end, _ := args["end"].(string)
			body, _ := json.Marshal(map[string]any{
				"summary": title,
				"start":   map[string]string{"dateTime": start},
				"end":     map[string]string{"dateTime": end},
			})
			return runGws(ctx, "calendar", "events", "insert",
				"--params", fmt.Sprintf(`{"calendarId":"%s"}`, cal),
				"--json", string(body))
		},
	})

	return s
}

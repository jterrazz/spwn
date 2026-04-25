package gate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	authgoogle "spwn.sh/packages/auth/google"
)

// runGws executes a gws subprocess with credentials wired and returns
// its stdout as a parsed JSON value. Falls back to raw text content
// when the response isn't JSON (gws errors are sometimes plain text).
//
// Auth is always-fresh: every call asks google.AccessToken for a
// non-expired access token (refreshing on demand) and passes it to
// gws via GOOGLE_WORKSPACE_CLI_TOKEN. This means tokens managed by
// `spwn auth login google` on the host land here without any
// per-call credential file shuffling — google.LoadTokens reads from
// ~/.spwn/credentials/google/tokens.json which is bind-mounted as
// /credentials/google/tokens.json inside the gate container.
func runGws(ctx context.Context, args ...string) (any, error) {
	tok, err := authgoogle.AccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("get google access token: %w", err)
	}
	if tok == "" {
		return nil, fmt.Errorf(
			"no google credentials — run on the host:\n" +
				"  spwn auth login google\n" +
				"the wizard guides you through a one-time GCP project setup,\n" +
				"then opens a browser for the OAuth click-Allow step",
		)
	}

	cmd := exec.CommandContext(ctx, "gws", args...)
	cmd.Env = append(os.Environ(),
		"GOOGLE_WORKSPACE_CLI_TOKEN="+tok,
	)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("gws %v: %w (stderr: %s)", args, err, bytes.TrimSpace(stderr.Bytes()))
	}

	out := bytes.TrimSpace(stdout.Bytes())
	if len(out) == 0 {
		return map[string]any{"ok": true}, nil
	}

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
	s.OnRefresh = func(ctx context.Context) error {
		// Pre-warm google access token alongside the gate's other
		// scheduled refreshes so the first gmail call after a long
		// idle period doesn't pay the OAuth-refresh round-trip.
		_, err := authgoogle.Refresh(ctx)
		return err
	}

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
// as Gmail: small curated tool set, gws backend, OnRefresh pre-warms
// the access token.
func NewGcalElement() *MCPServer {
	s := NewMCPServer("gcal", "Google Calendar (via gws)", "0.1.0")
	s.OnRefresh = func(ctx context.Context) error {
		_, err := authgoogle.Refresh(ctx)
		return err
	}

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

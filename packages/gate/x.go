package gate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
)

// XCookieProvider is the cookie-sync registration for the X element.
// The spwn-cookie-sync browser extension fetches this from the gate's
// /sync/providers endpoint and watches the listed domains for the
// listed cookie names. Defining the registration alongside the
// element keeps the contract close to the code that depends on it.
func XCookieProvider() CookieProvider {
	return CookieProvider{
		Name:    "x",
		Domains: []string{"x.com", "twitter.com"},
		Cookies: []string{"auth_token", "ct0"},
	}
}

// NewXElement exposes a small curated set of X (Twitter) read-only
// tools backed by twscrape, with cookies from the spwn-cookie-sync
// browser extension. Writes (post, reply) deliberately stay out of
// this element — they go through agent-browser in publish.sh after
// the user approves a draft, so the model can never publish without
// a human in the loop.
//
// The /credentials/x/cookies.json file is written by the gate's
// /sync/x endpoint when the extension pushes. The x-fetch helper
// (apps/gate/x-fetch, baked into the gate image) reads from there
// and shells out to twscrape per request.
func NewXElement() *MCPServer {
	s := NewMCPServer("x", "X (read via twscrape, cookies from extension)", "0.1.0")

	s.AddTool(&MCPTool{
		Name:        "fetch-favorites",
		Description: "Fetch the authenticated user's bookmarked tweets (favorites).",
		InputSchema: json.RawMessage(`{
            "type": "object",
            "properties": {
                "limit": { "type": "integer", "description": "Max tweets (default 50)" }
            }
        }`),
		Handler: func(ctx context.Context, args map[string]any) (any, error) {
			limit := intArg(args, "limit", 50)
			return runXFetch(ctx, "fetch-favorites", "--limit", fmt.Sprint(limit))
		},
	})

	s.AddTool(&MCPTool{
		Name:        "fetch-account",
		Description: "Fetch recent tweets from a specific X handle (without the @).",
		InputSchema: json.RawMessage(`{
            "type": "object",
            "properties": {
                "handle": { "type": "string", "description": "X handle, e.g. \"karpathy\"" },
                "limit":  { "type": "integer", "description": "Max tweets (default 50)" }
            },
            "required": ["handle"]
        }`),
		Handler: func(ctx context.Context, args map[string]any) (any, error) {
			handle, _ := args["handle"].(string)
			if handle == "" {
				return nil, fmt.Errorf("handle is required")
			}
			limit := intArg(args, "limit", 50)
			return runXFetch(ctx, "fetch-account", "--handle", handle, "--limit", fmt.Sprint(limit))
		},
	})

	s.AddTool(&MCPTool{
		Name:        "search",
		Description: "Search X for tweets matching a query (latest first).",
		InputSchema: json.RawMessage(`{
            "type": "object",
            "properties": {
                "query": { "type": "string", "description": "X search query (full operators supported)" },
                "limit": { "type": "integer", "description": "Max results (default 50)" }
            },
            "required": ["query"]
        }`),
		Handler: func(ctx context.Context, args map[string]any) (any, error) {
			q, _ := args["query"].(string)
			if q == "" {
				return nil, fmt.Errorf("query is required")
			}
			limit := intArg(args, "limit", 50)
			return runXFetch(ctx, "search", "--query", q, "--limit", fmt.Sprint(limit))
		},
	})

	s.AddTool(&MCPTool{
		Name:        "fetch-thread",
		Description: "Fetch a single tweet by id (with conversation context where available).",
		InputSchema: json.RawMessage(`{
            "type": "object",
            "properties": {
                "tweet_id": { "type": "string", "description": "Numeric tweet id" }
            },
            "required": ["tweet_id"]
        }`),
		Handler: func(ctx context.Context, args map[string]any) (any, error) {
			id, _ := args["tweet_id"].(string)
			if id == "" {
				return nil, fmt.Errorf("tweet_id is required")
			}
			return runXFetch(ctx, "fetch-thread", "--tweet-id", id)
		},
	})

	return s
}

// runXFetch shells out to /usr/local/bin/x-fetch and returns the
// parsed JSON output. Falls back to text if the output isn't JSON.
func runXFetch(ctx context.Context, args ...string) (any, error) {
	cmd := exec.CommandContext(ctx, "x-fetch", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("x-fetch %v: %w (stderr: %s)", args, err, bytes.TrimSpace(stderr.Bytes()))
	}
	out := bytes.TrimSpace(stdout.Bytes())
	if len(out) == 0 {
		return map[string]any{"items": []any{}, "count": 0}, nil
	}
	var parsed any
	if err := json.Unmarshal(out, &parsed); err != nil {
		return string(out), nil
	}
	return parsed, nil
}

// intArg pulls an integer-shaped value out of an args map regardless
// of whether the JSON-unmarshal landed it as float64 or int.
func intArg(args map[string]any, name string, def int) int {
	v, ok := args[name]
	if !ok {
		return def
	}
	switch x := v.(type) {
	case float64:
		return int(x)
	case int:
		return x
	}
	return def
}

package gate

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"sync"
	"time"

	"spwn.sh/packages/auth/mcp"
)

// NotionElement is a reverse proxy from the gate's `/mcp/notion/`
// surface to https://mcp.notion.com/mcp. The world container talks
// MCP to the gate, the gate adds the OAuth Bearer header, and the
// upstream sees an authenticated request. Credentials never reach
// the world container.
type NotionElement struct {
	provider mcp.Provider
	upstream *url.URL
	proxy    *httputil.ReverseProxy

	// Token cache: read once, refreshed by the package's scheduler;
	// reads serialized by mu so a refresh-and-write doesn't tear.
	mu    sync.RWMutex
	token string
}

// NewNotionElement constructs the Notion element. Returns an error
// only if the upstream URL fails to parse — credential absence is
// non-fatal at construction (calls fail at request time with 401,
// surfacing the actionable "spwn auth login notion" hint to the
// world's wrapper).
func NewNotionElement() (*NotionElement, error) {
	p, ok := mcp.Lookup("notion")
	if !ok {
		return nil, fmt.Errorf("notion provider not registered in mcp.Registry")
	}
	upstream, err := url.Parse(p.URL)
	if err != nil {
		return nil, fmt.Errorf("parse notion upstream: %w", err)
	}

	e := &NotionElement{provider: p, upstream: upstream}

	// Standard reverse proxy: rewrite Host + scheme to the upstream,
	// inject Authorization header from the cached token. Director
	// runs per request so token rotation lands without proxy reset.
	e.proxy = &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = upstream.Scheme
			req.URL.Host = upstream.Host
			// Map gate's path-stripped request back onto the upstream
			// MCP path. The upstream serves at /mcp on this host.
			req.URL.Path = "/mcp" + req.URL.Path
			req.Host = upstream.Host

			tok := e.cachedToken()
			if tok != "" {
				req.Header.Set("Authorization", "Bearer "+tok)
			}
		},
		// Increase the flush interval for SSE-style streaming bodies
		// (MCP streamable HTTP can use chunked / event-stream).
		FlushInterval: 100 * time.Millisecond,
		ErrorHandler: func(w http.ResponseWriter, _ *http.Request, err error) {
			http.Error(w, fmt.Sprintf("upstream error: %v", err), http.StatusBadGateway)
		},
	}

	// Best-effort: load the token from disk at startup so the first
	// request doesn't 401 while the scheduler hasn't yet ticked.
	_ = e.loadToken()
	return e, nil
}

// Name implements Element.
func (e *NotionElement) Name() string { return "notion" }

// Handler implements Element.
func (e *NotionElement) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if e.cachedToken() == "" {
			// Make the failure mode actionable rather than a silent
			// 401. The MCP wrapper script surfaces the body verbatim.
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error":  "no notion credentials on the gate host",
				"action": "run `spwn auth login notion` on the host",
			})
			return
		}
		e.proxy.ServeHTTP(w, r)
	})
}

// Refresh implements Element. Delegates to mcp.Refresh, then re-reads
// the token cache on success so subsequent requests use the new one.
func (e *NotionElement) Refresh(ctx context.Context) error {
	_, err := mcp.Refresh(ctx, e.provider, mcp.DefaultRefreshLeeway)
	// Always re-read the file regardless: another process (the user
	// running `spwn auth login notion`) may have rotated tokens.
	_ = e.loadToken()
	return err
}

func (e *NotionElement) cachedToken() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.token
}

// loadToken reads tokens.json from the gate-mounted credentials dir
// and updates the cache. Missing file → empty cache (handled at
// request time as 503).
func (e *NotionElement) loadToken() error {
	raw, err := os.ReadFile(mcp.ProviderTokenPath(e.provider))
	if err != nil {
		e.mu.Lock()
		e.token = ""
		e.mu.Unlock()
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var t struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(raw, &t); err != nil {
		return fmt.Errorf("parse tokens.json: %w", err)
	}
	e.mu.Lock()
	e.token = t.AccessToken
	e.mu.Unlock()
	return nil
}

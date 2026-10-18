package gate

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"spwn.sh/packages/authentication/mcp"
)

// ProxyElement is the generic auth-injecting reverse proxy. Any
// provider in mcp.Registry can become a gate element with three
// lines: parse the upstream URL, construct, register. Notion was
// the first instance; adding linear, atlassian, etc. is now
// trivial.
//
// Behaviour:
//   - Reads the OAuth access token from the spwn cred cache
//     (~/.spwn/credentials/mcp/oauth/<hash>/tokens.json) at startup
//     and on each Refresh tick.
//   - Reverse-proxies HTTP+MCP requests to the upstream URL with the
//     bearer token injected per request.
//   - Returns a 503 with an actionable JSON body when no token is on
//     disk so the world's wrapper can surface a "run spwn auth login"
//     hint instead of a generic 401.
type ProxyElement struct {
	provider mcp.Provider
	upstream *url.URL
	proxy    *httputil.ReverseProxy

	mu    sync.RWMutex
	token string
}

// NewProxyElement constructs a proxy element for p. The provider's
// URL is parsed once at construction so each request only pays an
// O(1) lookup, not a parse.
func NewProxyElement(p mcp.Provider) (*ProxyElement, error) {
	if p.Name == "" || p.URL == "" {
		return nil, fmt.Errorf("invalid provider: name=%q url=%q", p.Name, p.URL)
	}
	upstream, err := url.Parse(p.URL)
	if err != nil {
		return nil, fmt.Errorf("parse upstream URL %q: %w", p.URL, err)
	}
	if upstream.Scheme == "" || upstream.Host == "" {
		return nil, fmt.Errorf("upstream URL must include scheme and host: %q", p.URL)
	}
	upstreamPath := strings.TrimRight(upstream.Path, "/")

	e := &ProxyElement{provider: p, upstream: upstream}
	e.proxy = &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = upstream.Scheme
			req.URL.Host = upstream.Host
			req.URL.Path = upstreamPath + req.URL.Path
			req.Host = upstream.Host

			tok := e.cachedToken()
			if tok != "" {
				req.Header.Set("Authorization", "Bearer "+tok)
			}
		},
		FlushInterval: 100 * time.Millisecond,
		ErrorHandler: func(w http.ResponseWriter, _ *http.Request, err error) {
			http.Error(w, fmt.Sprintf("upstream %s: %v", p.Name, err), http.StatusBadGateway)
		},
	}

	_ = e.loadToken()
	return e, nil
}

// Name implements Element.
func (e *ProxyElement) Name() string { return e.provider.Name }

// Handler implements Element.
func (e *ProxyElement) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if e.cachedToken() == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error":  fmt.Sprintf("no %s credentials on the gate host", e.provider.Name),
				"action": fmt.Sprintf("run `spwn auth login %s` on the host", e.provider.Name),
			})
			return
		}
		e.proxy.ServeHTTP(w, r)
	})
}

// Refresh implements Element. Always re-reads the token cache after
// the refresh attempt so a third-party rotation (the user re-running
// spwn auth login on the host) lands without restarting the gate.
func (e *ProxyElement) Refresh(ctx context.Context) error {
	_, err := mcp.Refresh(ctx, e.provider, mcp.DefaultRefreshLeeway)
	_ = e.loadToken()
	return err
}

func (e *ProxyElement) cachedToken() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.token
}

func (e *ProxyElement) loadToken() error {
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
		return fmt.Errorf("parse %s tokens.json: %w", e.provider.Name, err)
	}
	e.mu.Lock()
	e.token = t.AccessToken
	e.mu.Unlock()
	return nil
}

// RegisterAllProviders walks mcp.Registry and adds a ProxyElement
// for every provider. Returns the count and the first per-provider
// construction error (which is non-fatal: other providers continue
// to register). Idempotent additions are not safe; call once at
// startup.
func RegisterAllProviders(reg *Registry) (int, error) {
	added := 0
	var firstErr error
	for _, name := range mcp.Names() {
		p, ok := mcp.Lookup(name)
		if !ok {
			continue
		}
		el, err := NewProxyElement(p)
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("%s: %w", name, err)
			}
			continue
		}
		if err := reg.Add(el); err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("%s: %w", name, err)
			}
			continue
		}
		added++
	}
	return added, firstErr
}

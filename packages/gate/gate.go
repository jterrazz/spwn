package gate

import (
	"context"
	"fmt"
	"net/http"
)

// DefaultListenAddr is the address the gate listens on inside its
// container. World containers reach it as `host.docker.internal:9000`.
// Override at runtime with --addr or SPWN_GATE_ADDR.
const DefaultListenAddr = ":9000"

// Element is one upstream service exposed through the gate. Each
// element owns a path prefix under /mcp/<name>/* and serves MCP
// requests with credentials injected.
//
// Implementations must be safe for concurrent use — the HTTP server
// invokes Handler concurrently across requests.
type Element interface {
	// Name is the URL-safe identifier the world container's wrapper
	// targets, e.g. `notion`, `gcp`. Must match `^[a-z][a-z0-9-]*$`.
	Name() string

	// Handler serves the element's portion of the MCP surface. The
	// HTTP request URL has been path-stripped — Handler sees paths
	// relative to /mcp/<name>/. Handler must inject auth and forward
	// upstream.
	Handler() http.Handler

	// Refresh is called by the gate's internal scheduler ahead of
	// token expiry. Elements that don't manage tokens return nil.
	Refresh(ctx context.Context) error
}

// Registry holds the live element set. Construction is decoupled from
// HTTP setup so callers can wire elements at startup, test them in
// isolation, and swap implementations without touching the server.
type Registry struct {
	elements map[string]Element
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{elements: map[string]Element{}}
}

// Add registers an element under its Name(). Returns an error on
// duplicate names so the gate fails fast at startup rather than
// shadowing tools at request time.
func (r *Registry) Add(e Element) error {
	if e == nil {
		return fmt.Errorf("nil element")
	}
	name := e.Name()
	if name == "" {
		return fmt.Errorf("element has empty name")
	}
	if _, ok := r.elements[name]; ok {
		return fmt.Errorf("element %q already registered", name)
	}
	r.elements[name] = e
	return nil
}

// Get returns an element by name plus a found flag.
func (r *Registry) Get(name string) (Element, bool) {
	e, ok := r.elements[name]
	return e, ok
}

// Names returns every registered element name. Order is unspecified.
func (r *Registry) Names() []string {
	out := make([]string, 0, len(r.elements))
	for n := range r.elements {
		out = append(out, n)
	}
	return out
}

// Each iterates every element. Iteration order is unspecified.
func (r *Registry) Each(fn func(Element)) {
	for _, e := range r.elements {
		fn(e)
	}
}

package gate

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// Server is the gate's HTTP frontend. Three route namespaces:
//
//   /healthz         operational liveness
//   /mcp/<element>/  the element bridge (proxy or hand-rolled MCP)
//   /sync/*          cookie-sync receive-end for the spwn-cookie-sync
//                    browser extension; nil cookies disables it
type Server struct {
	addr     string
	registry *Registry
	cookies  *CookieSync
	log      *log.Logger
	http     *http.Server
}

// NewServer constructs a server listening on addr (use "" for
// DefaultListenAddr). Pass cookies=nil to disable /sync/* (test
// harness convenience; production gates always wire it).
func NewServer(addr string, reg *Registry, cookies *CookieSync, logger *log.Logger) *Server {
	if addr == "" {
		addr = DefaultListenAddr
	}
	if logger == nil {
		logger = log.Default()
	}
	mux := http.NewServeMux()

	s := &Server{
		addr:     addr,
		registry: reg,
		cookies:  cookies,
		log:      logger,
		http:     &http.Server{Addr: addr, Handler: mux},
	}

	if cookies != nil {
		cookies.RegisterRoutes(mux)
	}

	// /healthz — liveness probe, used by `spwn gate status` and
	// kubernetes-style health checks. Always 200 if the server is
	// answering; element-specific health is per-element.
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":   "ok",
			"elements": reg.Names(),
		})
	})

	// /mcp/<element>/* — the element bridge. Path-strip before
	// handing off so each element sees requests relative to its own
	// root (`/`, `/initialize`, …) instead of `/mcp/<name>/...`.
	mux.HandleFunc("/mcp/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/mcp/")
		slash := strings.IndexByte(path, '/')
		var name, rest string
		if slash < 0 {
			name = path
			rest = ""
		} else {
			name = path[:slash]
			rest = path[slash:]
		}
		if name == "" {
			http.Error(w, "missing element name in /mcp/<name>/...", http.StatusNotFound)
			return
		}
		el, ok := reg.Get(name)
		if !ok {
			http.Error(w, fmt.Sprintf("unknown element %q", name), http.StatusNotFound)
			return
		}
		// Hand off with the path rewritten so the element handler
		// sees URLs relative to its own root.
		r2 := r.Clone(r.Context())
		r2.URL.Path = rest
		el.Handler().ServeHTTP(w, r2)
	})

	return s
}

// Run blocks serving HTTP until ctx is cancelled. Returns the first
// non-nil error from the server.
func (s *Server) Run(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		s.log.Printf("gate listening on %s", s.addr)
		err := s.http.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = s.http.Shutdown(shutdownCtx)
		return ctx.Err()
	}
}

// Addr returns the bound listen address (for tests / status output).
func (s *Server) Addr() string { return s.addr }

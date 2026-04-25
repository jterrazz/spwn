package gate

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os/exec"
	"sync"
	"time"
)

// ToolElement is the gate Element that fronts a catalog tool's
// MCP subprocess. It implements the Element interface (Name +
// Handler + Refresh) so the existing /mcp/<name>/ routing in
// server.go works without modification.
//
// Lifecycle: the supervisor (ToolSupervisor) spawns the subprocess
// at startup and restarts it on crash, blocking until /healthz on
// the assigned port answers 200. Once healthy, requests to
// /mcp/<name>/* are reverse-proxied transparently.
type ToolElement struct {
	tool  Tool
	port  int
	proxy *httputil.ReverseProxy
}

// Name returns the tool slug — used by the gate router to dispatch
// /mcp/<name>/* requests.
func (e *ToolElement) Name() string { return e.tool.Name }

// Handler returns a reverse proxy at 127.0.0.1:<port>. The gate's
// router strips the /mcp/<name>/ prefix before calling us; the
// subprocess sees /mcp/* paths just like any standalone MCP server
// would.
func (e *ToolElement) Handler() http.Handler { return e.proxy }

// Refresh is a no-op for tool elements — token refresh is owned
// by the subprocess (or by the user re-syncing cookies).
func (e *ToolElement) Refresh(_ context.Context) error { return nil }

// Port is the localhost port the tool's subprocess listens on.
// Useful for diagnostics / log lines.
func (e *ToolElement) Port() int { return e.port }

// ToolSupervisor manages the lifecycle of every catalog tool the
// gate has discovered. Each tool gets its own subprocess and its
// own port, allocated sequentially from BasePort.
type ToolSupervisor struct {
	tools    []Tool
	logger   *log.Logger
	basePort int

	mu       sync.Mutex
	elements map[string]*ToolElement
}

const (
	// BaseToolPort is where per-tool subprocesses get assigned.
	// Picked to leave 9000 (gate HTTP) and 9001 (browser sidecar)
	// free, with plenty of headroom.
	BaseToolPort = 9100
)

// NewToolSupervisor prepares a supervisor and eagerly builds one
// ToolElement per tool with an MCP entry, each pointed at the port
// the subprocess will listen on. Elements are registry-ready
// immediately — until the subprocess is healthy, requests get a
// clean 503 from the proxy's ErrorHandler instead of hanging.
//
// Caller flow:
//   tsup := NewToolSupervisor(tools, logger)
//   for _, el := range tsup.Elements() { reg.Add(el) }
//   go tsup.Run(ctx)
func NewToolSupervisor(tools []Tool, logger *log.Logger) *ToolSupervisor {
	s := &ToolSupervisor{
		tools:    tools,
		logger:   logger,
		basePort: BaseToolPort,
		elements: make(map[string]*ToolElement),
	}
	for i, t := range tools {
		if t.Spec.MCP == nil || len(t.Spec.MCP.Entry) == 0 {
			continue
		}
		port := s.basePort + i
		target, _ := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", port))
		el := &ToolElement{
			tool:  t,
			port:  port,
			proxy: httputil.NewSingleHostReverseProxy(target),
		}
		el.proxy.ErrorHandler = func(name string) func(http.ResponseWriter, *http.Request, error) {
			return func(w http.ResponseWriter, _ *http.Request, err error) {
				logger.Printf("tool %q upstream error: %v", name, err)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusServiceUnavailable)
				io.WriteString(w, fmt.Sprintf(`{"error":"tool %q not ready"}`, name))
			}
		}(t.Name)
		s.elements[t.Name] = el
	}
	return s
}

// Element returns the ToolElement for a tool name once it's healthy,
// or nil if not yet ready / not registered.
func (s *ToolSupervisor) Element(name string) *ToolElement {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.elements[name]
}

// Elements returns every ToolElement currently healthy. Used by the
// gate at startup to register them with the routing registry.
func (s *ToolSupervisor) Elements() []*ToolElement {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*ToolElement, 0, len(s.elements))
	for _, e := range s.elements {
		out = append(out, e)
	}
	return out
}

// Run spawns every tool's subprocess and restarts it on crash with
// backoff. Blocks until ctx is cancelled. Each tool runs in its own
// goroutine — one tool crashing doesn't impact the others.
func (s *ToolSupervisor) Run(ctx context.Context) {
	var wg sync.WaitGroup
	for _, el := range s.Elements() {
		wg.Add(1)
		go func(t Tool, port int) {
			defer wg.Done()
			s.superviseOne(ctx, t, port)
		}(el.tool, el.port)
	}
	wg.Wait()
}

func (s *ToolSupervisor) superviseOne(ctx context.Context, t Tool, port int) {
	backoff := time.Second
	for {
		if err := ctx.Err(); err != nil {
			return
		}
		if err := s.spawnOne(ctx, t, port); err != nil && ctx.Err() == nil {
			s.logger.Printf("tool %q exited: %v — restarting in %s", t.Name, err, backoff)
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
			}
			if backoff < 30*time.Second {
				backoff *= 2
			}
			continue
		}
		backoff = time.Second
		if ctx.Err() != nil {
			return
		}
	}
}

func (s *ToolSupervisor) spawnOne(ctx context.Context, t Tool, port int) error {
	entry := t.Spec.MCP.Entry
	cmd := exec.CommandContext(ctx, entry[0], entry[1:]...)
	cmd.Dir = t.Dir
	cmd.Env = append(cmd.Environ(),
		fmt.Sprintf("GATE_TOOL_NAME=%s", t.Name),
		fmt.Sprintf("GATE_TOOL_PORT=%d", port),
		"GATE_BROWSER_URL=http://127.0.0.1:9001",
		"GATE_CREDENTIALS_DIR=/credentials",
	)
	cmd.Stdout = prefixWriter{logger: s.logger, prefix: fmt.Sprintf("[tool:%s] ", t.Name)}
	cmd.Stderr = cmd.Stdout
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start: %w", err)
	}
	s.logger.Printf("tool %q started (pid %d, port %d) — waiting for health", t.Name, cmd.Process.Pid, port)
	if err := waitToolHealthy(ctx, port); err != nil {
		s.logger.Printf("tool %q health check failed: %v", t.Name, err)
	} else {
		s.logger.Printf("tool %q healthy on 127.0.0.1:%d", t.Name, port)
	}
	return cmd.Wait()
}

// waitToolHealthy polls the subprocess's /healthz until it answers
// or the deadline passes. Tools should expose this on their assigned
// port at the same path the sidecar uses.
func waitToolHealthy(ctx context.Context, port int) error {
	deadline := time.Now().Add(15 * time.Second)
	client := &http.Client{Timeout: time.Second}
	addr := fmt.Sprintf("http://127.0.0.1:%d/healthz", port)
	for time.Now().Before(deadline) {
		if err := ctx.Err(); err != nil {
			return err
		}
		resp, err := client.Get(addr)
		if err == nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			if resp.StatusCode == 200 {
				return nil
			}
		}
		time.Sleep(300 * time.Millisecond)
	}
	return fmt.Errorf("tool not healthy within deadline")
}

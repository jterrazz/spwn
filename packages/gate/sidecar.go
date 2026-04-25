package gate

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"sync"
	"time"
)

// SidecarBrowser supervises the gate-browser Node process, the
// Playwright sidecar baked into the gate image at /opt/gate-browser.
// It exposes an HTTP API on 127.0.0.1:9001 (gate-internal — never
// reachable from the host or worlds) that catalog tools call to
// drive cookie-loaded Chromium sessions.
//
// The supervisor restarts the sidecar on crash with a small backoff,
// so a tool that wedges Playwright doesn't take the gate with it.
type SidecarBrowser struct {
	cmd     []string // e.g. ["node", "/opt/gate-browser/index.js"]
	healthz string   // e.g. "http://127.0.0.1:9001/healthz"
	logger  *log.Logger

	mu      sync.Mutex
	proc    *exec.Cmd
	stopped bool
}

// NewSidecarBrowser builds a supervisor for the in-image
// gate-browser. Caller starts/stops it via Run.
func NewSidecarBrowser(logger *log.Logger) *SidecarBrowser {
	return &SidecarBrowser{
		cmd:     []string{"node", "/opt/gate-browser/index.js"},
		healthz: "http://127.0.0.1:9001/healthz",
		logger:  logger,
	}
}

// Run spawns the sidecar and restarts it on exit until the context
// is cancelled. Blocks until shutdown — call from a goroutine.
func (s *SidecarBrowser) Run(ctx context.Context) {
	backoff := time.Second
	for {
		if err := ctx.Err(); err != nil {
			return
		}
		if err := s.runOnce(ctx); err != nil && ctx.Err() == nil {
			s.logger.Printf("sidecar gate-browser exited: %v — restarting in %s", err, backoff)
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
		// Clean exit — reset backoff before re-spawn.
		backoff = time.Second
		if ctx.Err() != nil {
			return
		}
	}
}

func (s *SidecarBrowser) runOnce(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, s.cmd[0], s.cmd[1:]...)
	// Stream sidecar logs into the gate log with a prefix so they're
	// distinguishable from the Go logs in `docker logs spwn-gate`.
	cmd.Stdout = prefixWriter{logger: s.logger, prefix: "[browser] "}
	cmd.Stderr = prefixWriter{logger: s.logger, prefix: "[browser] "}
	s.mu.Lock()
	s.proc = cmd
	s.mu.Unlock()
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start sidecar: %w", err)
	}
	s.logger.Printf("sidecar gate-browser started (pid %d) — waiting for health", cmd.Process.Pid)
	if err := s.waitHealthy(ctx); err != nil {
		s.logger.Printf("sidecar gate-browser health check failed: %v", err)
	} else {
		s.logger.Printf("sidecar gate-browser healthy on 127.0.0.1:9001")
	}
	return cmd.Wait()
}

// waitHealthy polls /healthz until it answers 200 or the context
// dies. Sidecar takes ~1-3s to be ready (Chromium warmup).
func (s *SidecarBrowser) waitHealthy(ctx context.Context) error {
	deadline := time.Now().Add(15 * time.Second)
	client := &http.Client{Timeout: time.Second}
	for time.Now().Before(deadline) {
		if err := ctx.Err(); err != nil {
			return err
		}
		resp, err := client.Get(s.healthz)
		if err == nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			if resp.StatusCode == 200 {
				return nil
			}
		}
		time.Sleep(300 * time.Millisecond)
	}
	return fmt.Errorf("sidecar not healthy within deadline")
}

// prefixWriter is an io.Writer that splits on newline and forwards
// each line to a logger with a prefix. Keeps multi-line sidecar
// output readable in the gate's logs.
type prefixWriter struct {
	logger *log.Logger
	prefix string
}

func (p prefixWriter) Write(b []byte) (int, error) {
	// Naive: treat each Write as one record. Node typically flushes
	// a line at a time; not worth a buffered scanner.
	p.logger.Printf("%s%s", p.prefix, string(b))
	return len(b), nil
}

// Command spwn-gate is the host-side credential broker / element
// bridge daemon. Run as a long-lived Docker container by `spwn gate
// start` (see apps/cli/gate). One-shot use is supported for testing
// and debugging — `spwn-gate run` starts the server in foreground
// until SIGTERM.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"spwn.sh/packages/gate"
)

func main() {
	logger := log.New(os.Stderr, "[gate] ", log.LstdFlags|log.LUTC)

	if len(os.Args) < 2 || os.Args[1] != "run" {
		fmt.Fprintln(os.Stderr, "usage: spwn-gate run")
		os.Exit(2)
	}

	addr := os.Getenv("SPWN_GATE_ADDR")

	reg := gate.NewRegistry()
	if err := registerElements(reg, logger); err != nil {
		logger.Fatalf("register elements: %v", err)
	}

	cookies := gate.NewCookieSync()

	// Load catalog tools installed under ~/.spwn/gate/tools/ (mounted
	// at /gate/tools/ inside the container). Each tool with a `gate:`
	// section in its tool.yaml gets:
	//   - its CookieProvider auto-registered with cookie-sync
	//   - its MCP subprocess spawned + supervised
	//   - its /mcp/<name>/ route auto-wired into the registry once healthy
	tools, err := gate.LoadTools(gate.InContainerToolsDir)
	if err != nil {
		logger.Printf("warning: load catalog tools: %v", err)
	}
	for _, t := range tools {
		if cp := t.CookieProvider(); cp != nil {
			cookies.RegisterProvider(*cp)
			logger.Printf("registered cookie provider %q from catalog tool", cp.Name)
		}
	}

	logger.Printf("cookie-sync ready: %d provider(s) registered", len(cookies.Providers()))

	// Persist the per-provider domain hint to /credentials/<p>/.domains
	// so the gate-browser sidecar knows which hosts to seed cookies on
	// when a session for that provider is opened.
	if err := cookies.WriteDomainHints(); err != nil {
		logger.Printf("warning: write cookie-provider domain hints: %v", err)
	}

	srv := gate.NewServer(addr, reg, cookies, logger)
	sched := gate.NewScheduler(reg, 0, logger)
	sidecar := gate.NewSidecarBrowser(logger)
	tsup := gate.NewToolSupervisor(tools, logger)
	for _, el := range tsup.Elements() {
		if err := reg.Add(el); err != nil {
			logger.Printf("warning: add tool element %q: %v", el.Name(), err)
		} else {
			logger.Printf("registered MCP element %q from catalog tool (port %d)", el.Name(), el.Port())
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Catch SIGTERM/SIGINT for graceful shutdown — when the container
	// stops or a user Ctrl-Cs the foreground process.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		s := <-sigCh
		logger.Printf("signal %s — shutting down", s)
		cancel()
	}()

	// Scheduler + Playwright sidecar + tool subprocesses run
	// alongside the HTTP server; all stop on cancel.
	go sched.Run(ctx)
	go sidecar.Run(ctx)
	go tsup.Run(ctx)

	if err := srv.Run(ctx); err != nil && err != context.Canceled {
		logger.Fatalf("server: %v", err)
	}
	logger.Printf("clean shutdown")
}

// registerElements wires every gate element into the registry.
// Two layers:
//
//   - ProxyElement (auth-injecting reverse proxy) for every spwn-known
//     hosted MCP provider in mcp.Registry — notion today, linear etc.
//     come for free as one-line additions in packages/auth/mcp/provider.go.
//   - MCPServer (hand-rolled HTTP MCP) for backend-CLI elements — gmail
//     and gcal both back onto the gws CLI installed in this image.
func registerElements(reg *gate.Registry, logger *log.Logger) error {
	added, err := gate.RegisterAllProviders(reg)
	if err != nil {
		logger.Printf("warning: register providers: %v", err)
	}

	// gws-backed elements. Construction never fails — credentials are
	// checked per-request and surfaced as actionable errors.
	if err := reg.Add(gate.NewGmailElement()); err != nil {
		logger.Printf("warning: add gmail: %v", err)
	} else {
		added++
	}
	if err := reg.Add(gate.NewGcalElement()); err != nil {
		logger.Printf("warning: add gcal: %v", err)
	} else {
		added++
	}
	// Generic browser primitive — Stagehand-style escape hatch.
	// Agents that need ad-hoc browsing reach this via /mcp/browser;
	// agents that only use catalog tools (spwn:x, etc.) ignore it.
	if err := reg.Add(gate.NewBrowserElement()); err != nil {
		logger.Printf("warning: add browser: %v", err)
	} else {
		added++
	}
	logger.Printf("registered %d element(s): %v", added, reg.Names())
	return nil
}

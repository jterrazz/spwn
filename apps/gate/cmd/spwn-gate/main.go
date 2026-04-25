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
	// Register cookie providers alongside the elements that consume
	// them. Adding a new cookie-using element = define a CookieProvider
	// next to NewXxxElement and register it here.
	cookies.RegisterProvider(gate.XCookieProvider())
	logger.Printf("cookie-sync ready: %d provider(s) registered", len(cookies.Providers()))

	srv := gate.NewServer(addr, reg, cookies, logger)
	sched := gate.NewScheduler(reg, 0, logger)

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

	// Scheduler runs alongside the HTTP server; both stop on cancel.
	go sched.Run(ctx)

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
	// twscrape-backed read-only X scraper. Cookies come from the
	// browser extension via /sync/x → /credentials/x/cookies.json.
	if err := reg.Add(gate.NewXElement()); err != nil {
		logger.Printf("warning: add x: %v", err)
	} else {
		added++
	}

	logger.Printf("registered %d element(s): %v", added, reg.Names())
	return nil
}

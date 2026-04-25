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

	srv := gate.NewServer(addr, reg, logger)
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

// registerElements wires every spwn-known MCP provider into the
// registry as a generic auth-injecting reverse proxy. Adding a new
// provider is now a one-line change in
// packages/auth/mcp/provider.go — the gate picks it up automatically.
func registerElements(reg *gate.Registry, logger *log.Logger) error {
	added, err := gate.RegisterAllProviders(reg)
	if err != nil {
		// Per-provider failures aren't fatal — RegisterAllProviders
		// continues iterating and reports the first error. Log it
		// and let the gate start with whatever did register.
		logger.Printf("warning: register providers: %v", err)
	}
	logger.Printf("registered %d element(s): %v", added, reg.Names())
	return nil
}

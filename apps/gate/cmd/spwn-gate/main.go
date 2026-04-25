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

// registerElements wires every built-in element into the registry.
// As we add more upstream providers (gcp, x, …), each becomes one
// line here.
func registerElements(reg *gate.Registry, logger *log.Logger) error {
	notion, err := gate.NewNotionElement()
	if err != nil {
		return fmt.Errorf("notion: %w", err)
	}
	if err := reg.Add(notion); err != nil {
		return fmt.Errorf("add notion: %w", err)
	}
	logger.Printf("registered element: notion")
	return nil
}

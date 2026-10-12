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

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		s := <-sigCh
		logger.Printf("signal %s — shutting down", s)
		cancel()
	}()

	go sched.Run(ctx)
	go sidecar.Run(ctx)
	go tsup.Run(ctx)

	if err := srv.Run(ctx); err != nil && err != context.Canceled {
		logger.Fatalf("server: %v", err)
	}
	logger.Printf("clean shutdown")
}

func registerElements(reg *gate.Registry, logger *log.Logger) error {
	added, err := gate.RegisterAllProviders(reg)
	if err != nil {
		logger.Printf("warning: register providers: %v", err)
	}

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
	if err := reg.Add(gate.NewBrowserElement()); err != nil {
		logger.Printf("warning: add browser: %v", err)
	} else {
		added++
	}
	logger.Printf("registered %d element(s): %v", added, reg.Names())
	return nil
}

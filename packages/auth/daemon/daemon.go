package daemon

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/kardianos/service"

	"spwn.sh/packages/auth/mcp"
)

// Service identification — keep stable across versions, otherwise the
// OS init system loses track of installed instances on upgrade.
const (
	ServiceName        = "sh.spwn.auth-refresh"
	ServiceDisplayName = "spwn — MCP credential refresh"
	ServiceDescription = "Periodically refreshes OAuth tokens for MCP providers (Notion, …) so long-running agents never see expired credentials."
)

// DefaultInterval is how often the ticker fires. Picked so that with
// a 5-min refresh leeway we never let a 1h-TTL token (Notion) get
// closer than ~10 min to expiry, even if a tick is missed.
const DefaultInterval = 15 * time.Minute

// MinInterval guards against runaway ticks if a caller misconfigures
// the interval to e.g. zero. Anything below this is silently raised.
const MinInterval = 1 * time.Minute

// Refresher is the function the ticker calls each interval. Defaults
// to mcp.RefreshAll; tests swap it for a fake.
type Refresher func(ctx context.Context) (refreshed int, errs []error)

// Logger is a minimal sink so the package doesn't pin a specific
// logging library. Defaults to the std log package.
type Logger interface {
	Printf(format string, args ...any)
}

// Program is the long-running ticker that satisfies
// service.Interface. Start kicks off the goroutine and returns;
// Stop cancels its context and waits for it to drain.
type Program struct {
	Interval  time.Duration
	Refresh   Refresher
	Log       Logger
	StartTick bool // run one tick immediately on Start (default: true via New)

	cancel context.CancelFunc
	done   chan struct{}
}

// Start is called by service.Service.Run on service start. It must
// not block — kick off goroutines and return.
func (p *Program) Start(_ service.Service) error {
	if p.Interval < MinInterval {
		p.Interval = DefaultInterval
	}
	if p.Refresh == nil {
		p.Refresh = defaultRefresher
	}
	if p.Log == nil {
		p.Log = log.Default()
	}
	ctx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel
	p.done = make(chan struct{})
	go p.run(ctx)
	return nil
}

// Stop is called by the OS init system. Cancel the context and wait
// (briefly) for the run loop to drain so we don't leave a refresh
// half-written on shutdown.
func (p *Program) Stop(_ service.Service) error {
	if p.cancel != nil {
		p.cancel()
	}
	if p.done != nil {
		select {
		case <-p.done:
		case <-time.After(10 * time.Second):
			// Hard timeout — return anyway. The goroutine will exit
			// when its in-flight HTTP call finishes (capped at 30s
			// per provider by mcp.RefreshAll).
		}
	}
	return nil
}

func (p *Program) run(ctx context.Context) {
	defer close(p.done)

	if p.StartTick {
		p.tick(ctx)
	}

	t := time.NewTicker(p.Interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			p.tick(ctx)
		}
	}
}

func (p *Program) tick(ctx context.Context) {
	refreshed, errs := p.Refresh(ctx)
	for _, err := range errs {
		p.Log.Printf("warning: %v", err)
	}
	if refreshed > 0 {
		p.Log.Printf("refreshed %d MCP token(s)", refreshed)
	}
}

func defaultRefresher(ctx context.Context) (int, []error) {
	return mcp.RefreshAll(ctx, mcp.DefaultRefreshLeeway)
}

// New constructs a kardianos service.Service for the refresh daemon
// alongside the Program it drives. Pass interval=0 for DefaultInterval.
//
// The service is configured as a USER service (per-user launchd
// LaunchAgent on macOS, ~/.config/systemd/user on Linux). Tokens
// live in $HOME and the refresh runs as the same user that owns
// them — no root, no system-wide install.
func New(interval time.Duration) (service.Service, *Program, error) {
	prg := &Program{
		Interval:  interval,
		StartTick: true,
	}
	cfg := &service.Config{
		Name:        ServiceName,
		DisplayName: ServiceDisplayName,
		Description: ServiceDescription,
		// `spwn auth daemon run` is the entry point the OS init
		// system invokes. The same binary serves both as the daemon
		// (when called from launchd/systemd) and as the management
		// CLI (when called from a terminal).
		Arguments: []string{"auth", "daemon", "run"},
		Option: service.KeyValue{
			"UserService": true,
			// Keep stderr/stdout where the OS init system can capture
			// them — launchd writes to the standard log dirs by
			// default, systemd routes to journalctl.
		},
	}
	s, err := service.New(prg, cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("construct service: %w", err)
	}
	return s, prg, nil
}

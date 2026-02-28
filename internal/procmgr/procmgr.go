package procmgr

import (
	"context"
	"log/slog"
	"math"
	"sync"
	"time"

	"github.com/jterrazz/universe/internal/agent"
	"github.com/jterrazz/universe/internal/backend"
)

// Status represents the current state of the managed process.
type Status string

const (
	StatusStarting Status = "starting"
	StatusRunning  Status = "running"
	StatusStopped  Status = "stopped"
	StatusCrashed  Status = "crashed"
)

// Config controls restart behavior.
type Config struct {
	MaxRestarts   int           // Maximum consecutive restart attempts (0 = unlimited).
	RestartDelay  time.Duration // Base delay between restarts.
	MaxBackoff    time.Duration // Maximum delay after backoff.
	BackoffFactor float64       // Multiplier for exponential backoff.
}

// DefaultConfig returns sensible defaults for the process manager.
func DefaultConfig() Config {
	return Config{
		MaxRestarts:   5,
		RestartDelay:  2 * time.Second,
		MaxBackoff:    30 * time.Second,
		BackoffFactor: 2.0,
	}
}

// Report provides a thread-safe snapshot of the manager state.
type Report struct {
	Status        Status
	Restarts      int
	LastExitCode  int
	LastError     string
	StartedAt     time.Time
	LastRestartAt time.Time
}

// Manager supervises an agent process with automatic restart on crash.
type Manager struct {
	backend    backend.Backend
	universeID string
	opts       *agent.SpawnOptions
	cfg        Config

	mu         sync.Mutex
	status     Status
	restarts   int
	lastResult *agent.SpawnResult
	lastError  string
	startedAt  time.Time
	lastRestart time.Time

	cancel context.CancelFunc
	done   chan struct{}
}

// New creates a process manager.
func New(b backend.Backend, universeID string, opts *agent.SpawnOptions, cfg Config) *Manager {
	if cfg.RestartDelay == 0 {
		cfg.RestartDelay = 2 * time.Second
	}
	if cfg.MaxBackoff == 0 {
		cfg.MaxBackoff = 30 * time.Second
	}
	if cfg.BackoffFactor == 0 {
		cfg.BackoffFactor = 2.0
	}
	return &Manager{
		backend:    b,
		universeID: universeID,
		opts:       opts,
		cfg:        cfg,
		status:     StatusStopped,
		done:       make(chan struct{}),
	}
}

// Start begins the respawn loop in a background goroutine. Non-blocking.
func (m *Manager) Start(ctx context.Context) {
	ctx, m.cancel = context.WithCancel(ctx)
	m.mu.Lock()
	m.startedAt = time.Now()
	m.status = StatusStarting
	m.mu.Unlock()

	go m.loop(ctx)
}

// Stop cancels the manager and waits for the loop to exit.
func (m *Manager) Stop() {
	if m.cancel != nil {
		m.cancel()
	}
	<-m.done
}

// Wait blocks until the manager loop exits.
func (m *Manager) Wait() {
	<-m.done
}

// Health returns a thread-safe snapshot of the current state.
func (m *Manager) Health() Report {
	m.mu.Lock()
	defer m.mu.Unlock()
	return Report{
		Status:        m.status,
		Restarts:      m.restarts,
		LastExitCode:  m.lastExitCode(),
		LastError:     m.lastError,
		StartedAt:     m.startedAt,
		LastRestartAt: m.lastRestart,
	}
}

// LastResult returns the most recent spawn result, or nil.
func (m *Manager) LastResult() *agent.SpawnResult {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastResult
}

func (m *Manager) lastExitCode() int {
	if m.lastResult != nil {
		return m.lastResult.ExitCode
	}
	return -1
}

func (m *Manager) loop(ctx context.Context) {
	defer close(m.done)

	consecutiveFailures := 0

	for {
		m.mu.Lock()
		m.status = StatusRunning
		m.mu.Unlock()

		result, err := agent.Spawn(ctx, m.backend, m.universeID, m.opts)

		m.mu.Lock()
		m.lastResult = result
		if err != nil {
			m.lastError = err.Error()
		}
		m.mu.Unlock()

		// Context canceled → intentional stop.
		if ctx.Err() != nil {
			m.mu.Lock()
			m.status = StatusStopped
			m.mu.Unlock()
			return
		}

		// Clean exit → done.
		if err == nil && result != nil && result.ExitCode == 0 {
			m.mu.Lock()
			m.status = StatusStopped
			m.mu.Unlock()
			return
		}

		// Crash — decide whether to restart.
		consecutiveFailures++
		m.mu.Lock()
		m.status = StatusCrashed
		m.restarts = consecutiveFailures
		m.mu.Unlock()

		if m.cfg.MaxRestarts > 0 && consecutiveFailures >= m.cfg.MaxRestarts {
			slog.Warn("max restarts exceeded", "restarts", consecutiveFailures, "universe_id", m.universeID)
			return
		}

		delay := m.backoffDelay(consecutiveFailures)
		slog.Info("restarting agent", "attempt", consecutiveFailures, "delay", delay, "universe_id", m.universeID)

		select {
		case <-ctx.Done():
			m.mu.Lock()
			m.status = StatusStopped
			m.mu.Unlock()
			return
		case <-time.After(delay):
		}

		m.mu.Lock()
		m.lastRestart = time.Now()
		m.mu.Unlock()
	}
}

func (m *Manager) backoffDelay(failures int) time.Duration {
	delay := float64(m.cfg.RestartDelay) * math.Pow(m.cfg.BackoffFactor, float64(failures-1))
	if delay > float64(m.cfg.MaxBackoff) {
		delay = float64(m.cfg.MaxBackoff)
	}
	return time.Duration(delay)
}

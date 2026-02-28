package procmgr

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/jterrazz/universe/internal/backend"
	"github.com/jterrazz/universe/internal/config"
)

// mockBackend implements backend.Backend for testing.
type mockBackend struct {
	mu       sync.Mutex
	execFunc func(ctx context.Context, id string, cmd []string) (*backend.ExecResult, error)
	calls    int
}

func (m *mockBackend) Create(ctx context.Context, cfg *config.UniverseConfig) (string, error) {
	return "test-id", nil
}
func (m *mockBackend) Start(ctx context.Context, id string) error   { return nil }
func (m *mockBackend) Stop(ctx context.Context, id string) error    { return nil }
func (m *mockBackend) Remove(ctx context.Context, id string) error  { return nil }
func (m *mockBackend) List(ctx context.Context) ([]backend.ContainerInfo, error) {
	return nil, nil
}
func (m *mockBackend) Inspect(ctx context.Context, id string) (*backend.ContainerInfo, error) {
	return nil, nil
}
func (m *mockBackend) Exec(ctx context.Context, id string, cmd []string) (*backend.ExecResult, error) {
	m.mu.Lock()
	m.calls++
	m.mu.Unlock()
	if m.execFunc != nil {
		return m.execFunc(ctx, id, cmd)
	}
	return &backend.ExecResult{ExitCode: 0}, nil
}

func (m *mockBackend) CallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.calls
}

func TestCleanExit_NoRestarts(t *testing.T) {
	b := &mockBackend{
		execFunc: func(ctx context.Context, id string, cmd []string) (*backend.ExecResult, error) {
			return &backend.ExecResult{ExitCode: 0, Stdout: "done"}, nil
		},
	}

	mgr := New(b, "test-universe", nil, DefaultConfig())
	mgr.Start(context.Background())
	mgr.Wait()

	report := mgr.Health()
	if report.Status != StatusStopped {
		t.Errorf("expected stopped, got %s", report.Status)
	}
	if report.Restarts != 0 {
		t.Errorf("expected 0 restarts, got %d", report.Restarts)
	}
	if b.CallCount() != 1 {
		t.Errorf("expected 1 exec call, got %d", b.CallCount())
	}
}

func TestCrashAndRestart(t *testing.T) {
	callCount := 0
	b := &mockBackend{
		execFunc: func(ctx context.Context, id string, cmd []string) (*backend.ExecResult, error) {
			callCount++
			if callCount <= 2 {
				return &backend.ExecResult{ExitCode: 1, Stderr: "crash"}, fmt.Errorf("agent failed")
			}
			return &backend.ExecResult{ExitCode: 0}, nil
		},
	}

	cfg := Config{
		MaxRestarts:   5,
		RestartDelay:  10 * time.Millisecond,
		MaxBackoff:    50 * time.Millisecond,
		BackoffFactor: 1.5,
	}

	mgr := New(b, "test-universe", nil, cfg)
	mgr.Start(context.Background())
	mgr.Wait()

	report := mgr.Health()
	if report.Status != StatusStopped {
		t.Errorf("expected stopped after recovery, got %s", report.Status)
	}
}

func TestMaxRestartsExceeded(t *testing.T) {
	b := &mockBackend{
		execFunc: func(ctx context.Context, id string, cmd []string) (*backend.ExecResult, error) {
			return &backend.ExecResult{ExitCode: 1, Stderr: "crash"}, fmt.Errorf("agent failed")
		},
	}

	cfg := Config{
		MaxRestarts:   3,
		RestartDelay:  10 * time.Millisecond,
		MaxBackoff:    50 * time.Millisecond,
		BackoffFactor: 1.5,
	}

	mgr := New(b, "test-universe", nil, cfg)
	mgr.Start(context.Background())
	mgr.Wait()

	report := mgr.Health()
	if report.Status != StatusCrashed {
		t.Errorf("expected crashed, got %s", report.Status)
	}
	if report.Restarts != 3 {
		t.Errorf("expected 3 restarts, got %d", report.Restarts)
	}
}

func TestStop_CancelsLoop(t *testing.T) {
	b := &mockBackend{
		execFunc: func(ctx context.Context, id string, cmd []string) (*backend.ExecResult, error) {
			// Simulate a long-running process.
			select {
			case <-ctx.Done():
				return &backend.ExecResult{ExitCode: 1}, ctx.Err()
			case <-time.After(10 * time.Second):
				return &backend.ExecResult{ExitCode: 0}, nil
			}
		},
	}

	mgr := New(b, "test-universe", nil, DefaultConfig())
	mgr.Start(context.Background())

	// Give the loop time to start.
	time.Sleep(50 * time.Millisecond)
	mgr.Stop()

	report := mgr.Health()
	if report.Status != StatusStopped {
		t.Errorf("expected stopped after cancel, got %s", report.Status)
	}
}

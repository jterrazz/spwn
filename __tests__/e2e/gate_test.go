//go:build e2e

package e2e

import (
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/jterrazz/universe/internal/config"
	"github.com/jterrazz/universe/internal/gate"

	"github.com/jterrazz/universe/__tests__/e2e/setup"
)

func TestSpawn_GateBridgeInjected(t *testing.T) {
	bridge := config.GateBridge{
		Source:       "mcp/slack",
		As:           "slack-send",
		Capabilities: []string{"send"},
	}

	setup.NewSpawnBuilder(t).
		NoAgent().
		WithGate(bridge).
		Execute().
		ExpectGate(func(g *setup.GateAssertion) {
			g.HasBridge("slack-send")
			g.BridgeIsExecutable("slack-send")
			g.HasSocket()
		})
}

func TestSpawn_MultipleGateBridges(t *testing.T) {
	bridges := []config.GateBridge{
		{Source: "mcp/slack", As: "slack-send", Capabilities: []string{"send"}},
		{Source: "mcp/db", As: "db-query", Capabilities: []string{"read"}},
	}

	setup.NewSpawnBuilder(t).
		NoAgent().
		WithGate(bridges...).
		Execute().
		ExpectGate(func(g *setup.GateAssertion) {
			g.HasBridge("slack-send")
			g.HasBridge("db-query")
			g.BridgeIsExecutable("slack-send")
			g.BridgeIsExecutable("db-query")
		}).
		ExpectContainer(func(c *setup.ContainerAssertion) {
			c.FileContains("/universe/faculties.md", "Gate Bridges")
			c.FileContains("/universe/faculties.md", "slack-send")
			c.FileContains("/universe/faculties.md", "db-query")
		})
}

func TestSpawn_GateBridgeAppearsInFaculties(t *testing.T) {
	bridge := config.GateBridge{
		Source:       "mcp/slack",
		As:           "slack-send",
		Capabilities: []string{"send", "read"},
	}

	setup.NewSpawnBuilder(t).
		NoAgent().
		WithGate(bridge).
		Execute().
		ExpectContainer(func(c *setup.ContainerAssertion) {
			c.FileContains("/universe/faculties.md", "## Gate Bridges")
			c.FileContains("/universe/faculties.md", "`slack-send`")
			c.FileContains("/universe/faculties.md", "mcp/slack")
			c.FileContains("/universe/faculties.md", "send, read")
		})
}

func TestSpawn_NoGateBridges_NoSocket(t *testing.T) {
	setup.NewSpawnBuilder(t).
		NoAgent().
		Execute().
		ExpectGate(func(g *setup.GateAssertion) {
			g.NoSocket()
		})
}

func TestGate_BridgeProxiesThroughSocket(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("unix socket proxying through Docker bind mounts is not supported on macOS Docker Desktop")
	}

	var mu sync.Mutex
	var calls []gate.InvokeRequest

	handler := func(element string, args []string) (gate.InvokeResult, error) {
		mu.Lock()
		calls = append(calls, gate.InvokeRequest{Element: element, Args: args})
		mu.Unlock()
		return gate.InvokeResult{
			ExitCode: 0,
			Stdout:   "hello from " + element,
		}, nil
	}

	bridge := config.GateBridge{
		Source: "mcp/echo",
		As:     "echo-tool",
	}

	chain := setup.NewSpawnBuilder(t).
		NoAgent().
		WithGate(bridge).
		WithInvokeHandler(handler).
		Execute()

	// Execute the wrapper script inside the container
	output := chain.ExecInContainer([]string{"/gate/bin/echo-tool", "arg1", "arg2"})

	mu.Lock()
	defer mu.Unlock()

	if len(calls) == 0 {
		t.Fatal("Expected invoke handler to be called, but it wasn't")
	}
	if calls[0].Element != "echo-tool" {
		t.Fatalf("Expected element %q, got %q", "echo-tool", calls[0].Element)
	}
	if !strings.Contains(output, "hello from echo-tool") {
		t.Fatalf("Expected output to contain %q, got: %s", "hello from echo-tool", output)
	}
}

func TestGate_MultipleCallsRecorded(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("unix socket proxying through Docker bind mounts is not supported on macOS Docker Desktop")
	}

	var mu sync.Mutex
	var calls []gate.InvokeRequest

	handler := func(element string, args []string) (gate.InvokeResult, error) {
		mu.Lock()
		calls = append(calls, gate.InvokeRequest{Element: element, Args: args})
		mu.Unlock()
		return gate.InvokeResult{ExitCode: 0, Stdout: "ok"}, nil
	}

	bridge := config.GateBridge{Source: "mcp/test", As: "test-cmd"}

	chain := setup.NewSpawnBuilder(t).
		NoAgent().
		WithGate(bridge).
		WithInvokeHandler(handler).
		Execute()

	// Make three calls
	chain.ExecInContainer([]string{"/gate/bin/test-cmd", "first"})
	chain.ExecInContainer([]string{"/gate/bin/test-cmd", "second"})
	chain.ExecInContainer([]string{"/gate/bin/test-cmd", "third"})

	mu.Lock()
	defer mu.Unlock()

	if len(calls) != 3 {
		t.Fatalf("Expected 3 calls, got %d", len(calls))
	}
}

func TestGate_UnbridgedToolDoesNotExist(t *testing.T) {
	bridge := config.GateBridge{Source: "mcp/slack", As: "slack-send"}

	setup.NewSpawnBuilder(t).
		NoAgent().
		WithGate(bridge).
		Execute().
		ExpectGate(func(g *setup.GateAssertion) {
			g.HasBridge("slack-send")
			g.NoBridge("db-query")
			g.NoBridge("gh")
		})
}

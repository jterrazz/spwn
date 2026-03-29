//go:build e2e

package e2e

import (
	"strings"
	"sync"
	"testing"

	"github.com/jterrazz/spwn/core/gate"
	"github.com/jterrazz/spwn/core/universe/tests/e2e/setup"
)

func TestSpawn_GateBridgeInjected(t *testing.T) {
	// GIVEN a gate bridge configuration
	bridge := gate.Bridge{
		Source:       "mcp/slack",
		As:           "slack-send",
		Capabilities: []string{"send"},
	}

	// WHEN a universe is spawned with the bridge
	chain := setup.NewSpawnBuilder(t).
		NoAgent().
		WithGate(bridge).
		Execute()

	// THEN the bridge should be installed and executable
	chain.ExpectGate(func(g *setup.GateAssertion) {
		g.HasBridge("slack-send")
		g.BridgeIsExecutable("slack-send")
	})
}

func TestSpawn_MultipleGateBridges(t *testing.T) {
	// GIVEN two gate bridge configurations
	bridges := []gate.Bridge{
		{Source: "mcp/slack", As: "slack-send", Capabilities: []string{"send"}},
		{Source: "mcp/db", As: "db-query", Capabilities: []string{"read"}},
	}

	// WHEN a universe is spawned with both bridges
	chain := setup.NewSpawnBuilder(t).
		NoAgent().
		WithGate(bridges...).
		Execute()

	// THEN both bridges should be installed and executable
	chain.ExpectGate(func(g *setup.GateAssertion) {
		g.HasBridge("slack-send")
		g.HasBridge("db-query")
		g.BridgeIsExecutable("slack-send")
		g.BridgeIsExecutable("db-query")
	})

	// AND the faculties should document both bridges
	chain.ExpectContainer(func(c *setup.ContainerAssertion) {
		c.FileContains("/universe/faculties.md", "Gate Bridges")
		c.FileContains("/universe/faculties.md", "slack-send")
		c.FileContains("/universe/faculties.md", "db-query")
	})
}

func TestSpawn_GateBridgeAppearsInFaculties(t *testing.T) {
	// GIVEN a bridge with multiple capabilities
	bridge := gate.Bridge{
		Source:       "mcp/slack",
		As:           "slack-send",
		Capabilities: []string{"send", "read"},
	}

	// WHEN a universe is spawned with that bridge
	chain := setup.NewSpawnBuilder(t).
		NoAgent().
		WithGate(bridge).
		Execute()

	// THEN the faculties file should document the bridge details
	chain.ExpectContainer(func(c *setup.ContainerAssertion) {
		c.FileContains("/universe/faculties.md", "## Gate Bridges")
		c.FileContains("/universe/faculties.md", "`slack-send`")
		c.FileContains("/universe/faculties.md", "mcp/slack")
		c.FileContains("/universe/faculties.md", "send, read")
	})
}

func TestSpawn_NoGateBridges_NoGateDir(t *testing.T) {
	// GIVEN no gate bridges configured
	// WHEN a universe is spawned
	chain := setup.NewSpawnBuilder(t).
		NoAgent().
		Execute()

	// THEN no /gate directory should exist
	chain.ExpectGate(func(g *setup.GateAssertion) {
		g.NoGateDir()
	})
}

func TestGate_BridgeProxiesThroughSocket(t *testing.T) {
	// GIVEN a gate bridge with an invoke handler that records calls
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

	bridge := gate.Bridge{
		Source: "mcp/echo",
		As:     "echo-tool",
	}

	chain := setup.NewSpawnBuilder(t).
		NoAgent().
		WithGate(bridge).
		WithInvokeHandler(handler).
		Execute()

	// WHEN the bridge is invoked from inside the container
	output := chain.ExecInContainer([]string{"/gate/bin/echo-tool", "arg1", "arg2"})

	// THEN the invoke handler should have been called with the correct element
	mu.Lock()
	defer mu.Unlock()

	if len(calls) == 0 {
		t.Fatal("Expected invoke handler to be called, but it wasn't")
	}
	if calls[0].Element != "echo-tool" {
		t.Fatalf("Expected element %q, got %q", "echo-tool", calls[0].Element)
	}

	// AND the output should contain the handler's response
	if !strings.Contains(output, "hello from echo-tool") {
		t.Fatalf("Expected output to contain %q, got: %s", "hello from echo-tool", output)
	}
}

func TestGate_MultipleCallsRecorded(t *testing.T) {
	// GIVEN a gate bridge with a call-recording handler
	var mu sync.Mutex
	var calls []gate.InvokeRequest

	handler := func(element string, args []string) (gate.InvokeResult, error) {
		mu.Lock()
		calls = append(calls, gate.InvokeRequest{Element: element, Args: args})
		mu.Unlock()
		return gate.InvokeResult{ExitCode: 0, Stdout: "ok"}, nil
	}

	bridge := gate.Bridge{Source: "mcp/test", As: "test-cmd"}

	chain := setup.NewSpawnBuilder(t).
		NoAgent().
		WithGate(bridge).
		WithInvokeHandler(handler).
		Execute()

	// WHEN the bridge is invoked three times
	chain.ExecInContainer([]string{"/gate/bin/test-cmd", "first"})
	chain.ExecInContainer([]string{"/gate/bin/test-cmd", "second"})
	chain.ExecInContainer([]string{"/gate/bin/test-cmd", "third"})

	// THEN the handler should have been called three times
	mu.Lock()
	defer mu.Unlock()

	if len(calls) != 3 {
		t.Fatalf("Expected 3 calls, got %d", len(calls))
	}
}

func TestGate_UnbridgedToolDoesNotExist(t *testing.T) {
	// GIVEN a universe with only one bridge configured
	bridge := gate.Bridge{Source: "mcp/slack", As: "slack-send"}

	chain := setup.NewSpawnBuilder(t).
		NoAgent().
		WithGate(bridge).
		Execute()

	// THEN only the configured bridge should exist, not others
	chain.ExpectGate(func(g *setup.GateAssertion) {
		g.HasBridge("slack-send")
		g.NoBridge("db-query")
		g.NoBridge("gh")
	})
}

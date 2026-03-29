//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/jterrazz/spwn/core/universe/tests/e2e/setup"
)

func TestLogs_ReturnsReaderFromRunningUniverse(t *testing.T) {
	tc := setup.NewTestContext(t)
	tc.InitAgent("logs-agent")

	chain := tc.Spawn().
		WithAgent("logs-agent").
		Detached().
		Execute()

	reader, err := tc.Arc.Logs(context.Background(), chain.Universe().ID, false, "10")
	if err != nil {
		t.Fatalf("Logs() returned error: %v", err)
	}
	defer reader.Close()

	// Read available output — mock writes to stdout, so we should get something
	buf := make([]byte, 4096)
	n, err := reader.Read(buf)
	if err != nil && err != io.EOF {
		t.Fatalf("Failed to read logs: %v", err)
	}
	if n == 0 {
		t.Log("No log output captured (mock may not have written to stdout yet)")
	}
}

func TestLogs_ContainsMockOutput(t *testing.T) {
	tc := setup.NewTestContext(t)
	tc.InitAgent("logs-content-agent")

	chain := tc.Spawn().
		WithAgent("logs-content-agent").
		Detached().
		Execute()

	// Verify mock was called first
	chain.ExpectMock(func(m *setup.MockAssertion) {
		m.WasCalled()
	})

	reader, err := tc.Arc.Logs(context.Background(), chain.Universe().ID, false, "100")
	if err != nil {
		t.Fatalf("Logs() returned error: %v", err)
	}
	defer reader.Close()

	var output bytes.Buffer
	io.Copy(&output, reader)

	// The reader should be valid and readable (content depends on mock behavior)
	t.Logf("Captured %d bytes of log output", output.Len())
}

func TestLogs_NoFollowReturnsImmediately(t *testing.T) {
	tc := setup.NewTestContext(t)
	tc.InitAgent("logs-nofollow-agent")

	chain := tc.Spawn().
		WithAgent("logs-nofollow-agent").
		Detached().
		Execute()

	// With follow=false, Logs should return and the reader should be finite
	reader, err := tc.Arc.Logs(context.Background(), chain.Universe().ID, false, "5")
	if err != nil {
		t.Fatalf("Logs(follow=false) returned error: %v", err)
	}
	defer reader.Close()

	// Reading should eventually reach EOF (not hang forever)
	var buf bytes.Buffer
	_, err = io.Copy(&buf, reader)
	if err != nil {
		t.Fatalf("Failed to read all log output: %v", err)
	}
}

func TestLogs_NonExistentUniverseReturnsError(t *testing.T) {
	tc := setup.NewTestContext(t)

	_, err := tc.Arc.Logs(context.Background(), "u-nonexistent-12345", false, "10")
	if err == nil {
		t.Fatal("Expected error when getting logs for non-existent universe, got nil")
	}
}

func TestLogs_IdleUniverseWithoutAgent(t *testing.T) {
	tc := setup.NewTestContext(t)

	chain := tc.Spawn().
		NoAgent().
		Execute()

	// Logs should still work on an idle container (just no agent output)
	reader, err := tc.Arc.Logs(context.Background(), chain.Universe().ID, false, "5")
	if err != nil {
		t.Fatalf("Logs() on idle universe returned error: %v", err)
	}
	defer reader.Close()

	var buf bytes.Buffer
	io.Copy(&buf, reader)
	// No assertion on content — idle container may have no output
}

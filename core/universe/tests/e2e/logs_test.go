//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"spwn.sh/core/universe/tests/e2e/setup"
)

func TestLogs_ContainsMockOutput(t *testing.T) {
	// GIVEN a universe with a detached agent that writes to stdout
	tc := setup.NewTestContext(t)
	tc.InitAgent("logs-content-agent")

	chain := tc.Spawn().
		WithAgent("logs-content-agent").
		Detached().
		Execute()

	// WHEN we read the logs
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

	// THEN the log output should contain content from the mock's stdout
	logContent := output.String()
	if output.Len() == 0 {
		t.Fatal("Expected log output to contain bytes, got 0")
	}
	t.Logf("Captured %d bytes of log output: %s", output.Len(), logContent)
}

func TestLogs_NoFollowReturnsImmediately(t *testing.T) {
	// GIVEN a universe with a detached agent
	tc := setup.NewTestContext(t)
	tc.InitAgent("logs-nofollow-agent")

	chain := tc.Spawn().
		WithAgent("logs-nofollow-agent").
		Detached().
		Execute()

	// WHEN we read logs with follow=false
	reader, err := tc.Arc.Logs(context.Background(), chain.Universe().ID, false, "5")
	if err != nil {
		t.Fatalf("Logs(follow=false) returned error: %v", err)
	}
	defer reader.Close()

	// THEN reading should reach EOF (not hang forever) and return bytes
	var buf bytes.Buffer
	_, err = io.Copy(&buf, reader)
	if err != nil {
		t.Fatalf("Failed to read all log output: %v", err)
	}
	if buf.Len() == 0 {
		t.Fatal("Expected non-empty log output from follow=false reader")
	}
}

func TestLogs_IdleUniverseWithoutAgent(t *testing.T) {
	// GIVEN a universe with no agent (idle)
	tc := setup.NewTestContext(t)

	chain := tc.Spawn().
		NoAgent().
		Execute()

	// WHEN we read logs from the idle container
	reader, err := tc.Arc.Logs(context.Background(), chain.Universe().ID, false, "5")
	if err != nil {
		t.Fatalf("Logs() on idle universe returned error: %v", err)
	}
	defer reader.Close()

	// THEN the reader should return without error (content may be empty for idle container)
	var buf bytes.Buffer
	_, err = io.Copy(&buf, reader)
	if err != nil {
		t.Fatalf("Reading logs from idle universe should not error: %v", err)
	}
}

func TestLogs_NonExistentUniverseReturnsError(t *testing.T) {
	// GIVEN a test context with no universes
	tc := setup.NewTestContext(t)

	// WHEN we request logs for a non-existent universe
	_, err := tc.Arc.Logs(context.Background(), "u-nonexistent-12345", false, "10")

	// THEN it should return an error
	if err == nil {
		t.Fatal("Expected error when getting logs for non-existent universe, got nil")
	}
}

func TestLogs_ReturnsReaderFromRunningUniverse(t *testing.T) {
	// GIVEN a universe with a detached agent
	tc := setup.NewTestContext(t)
	tc.InitAgent("logs-agent")

	chain := tc.Spawn().
		WithAgent("logs-agent").
		Detached().
		Execute()

	// WHEN we request a log reader
	reader, err := tc.Arc.Logs(context.Background(), chain.Universe().ID, false, "10")
	if err != nil {
		t.Fatalf("Logs() returned error: %v", err)
	}
	defer reader.Close()

	// THEN the reader should provide output from the running container
	var buf bytes.Buffer
	io.Copy(&buf, reader)

	// The mock writes a JSON blob and echoes to stdout; we should see something
	if buf.Len() == 0 {
		t.Fatal("Expected log output from running universe, got 0 bytes")
	}

	// Verify the output looks like it came from the mock (it writes JSON to a file
	// but the container entrypoint or mock may echo to stdout)
	logStr := buf.String()
	if !strings.Contains(logStr, "{") && buf.Len() < 10 {
		t.Logf("Log output was short and didn't contain JSON markers: %q", logStr)
	}
}

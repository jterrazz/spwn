//go:build e2e

package architect

import (
	"context"
	"testing"
	"time"
)

func TestStopDaemon_ReturnsBeforeTheGracePeriod(t *testing.T) {
	// Given - an architect container running on the test image, in its own home and under its own name
	t.Setenv("SPWN_HOME", t.TempDir())
	t.Setenv("SPWN_ARCHITECT_CONTAINER_SUFFIX", "stop-init-test")
	ctx := context.Background()
	if _, err := StartDaemon(ctx, "spwn-test:latest"); err != nil {
		t.Fatalf("start: %v", err)
	}
	t.Cleanup(func() { _ = StopDaemon(ctx) })

	// When - it is stopped
	begin := time.Now()
	if err := StopDaemon(ctx); err != nil {
		t.Fatalf("stop: %v", err)
	}

	// Then - the stop answers a SIGTERM at once instead of waiting Docker's 10 s grace
	if elapsed := time.Since(begin); elapsed > 3*time.Second {
		t.Fatalf("stop took %s, want under 3s", elapsed)
	}
}

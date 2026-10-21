//go:build e2e

package backend

import (
	"context"
	"testing"
	"time"
)

func TestDocker_StopReturnsBeforeTheGracePeriod(t *testing.T) {
	// Given - a started world container whose PID 1 is `sleep infinity`
	ctx := context.Background()
	d, err := NewDocker()
	if err != nil {
		t.Skipf("docker unavailable: %v", err)
	}
	id, err := d.Create(ctx, ContainerConfig{Image: "spwn-test:latest", Name: "spwn-stop-init-test", PidsLimit: 100, NetworkMode: "none"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	t.Cleanup(func() { _ = d.Remove(ctx, id) })
	if err := d.Start(ctx, id); err != nil {
		t.Fatalf("start: %v", err)
	}

	// When - it is stopped
	begin := time.Now()
	if err := d.Stop(ctx, id); err != nil {
		t.Fatalf("stop: %v", err)
	}

	// Then - the stop answers a SIGTERM at once instead of waiting Docker's 10 s grace
	if elapsed := time.Since(begin); elapsed > 3*time.Second {
		t.Fatalf("stop took %s, want under 3s", elapsed)
	}
}

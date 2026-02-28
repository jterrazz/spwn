//go:build integration

package backend

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jterrazz/universe/internal/config"
)

const testImage = "alpine:3.19"

func testBackend(t *testing.T) (*DockerBackend, context.Context) {
	t.Helper()
	b, err := NewDockerBackend()
	if err != nil {
		t.Fatalf("cannot connect to Docker: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)
	return b, ctx
}

func mustCreate(t *testing.T, b *DockerBackend, ctx context.Context, cfg *config.UniverseConfig) string {
	t.Helper()
	id, err := b.Create(ctx, cfg)
	if err != nil {
		t.Fatalf("create container: %v", err)
	}
	t.Cleanup(func() {
		cleanCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		_ = b.Remove(cleanCtx, id)
	})
	return id
}

func TestCreateAndRemove(t *testing.T) {
	b, ctx := testBackend(t)
	cfg := &config.UniverseConfig{Image: testImage}

	id, err := b.Create(ctx, cfg)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if len(id) < 8 {
		t.Fatalf("expected valid UUID, got %q", id)
	}

	info, err := b.Inspect(ctx, id)
	if err != nil {
		t.Fatalf("Inspect after Create: %v", err)
	}
	if info.ID != id {
		t.Errorf("Inspect ID = %q, want %q", info.ID, id)
	}

	if err := b.Remove(ctx, id); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	_, err = b.Inspect(ctx, id)
	if err == nil {
		t.Fatal("Inspect after Remove should error")
	}
}

func TestStartAndStop(t *testing.T) {
	b, ctx := testBackend(t)
	id := mustCreate(t, b, ctx, &config.UniverseConfig{Image: testImage})

	if err := b.Start(ctx, id); err != nil {
		t.Fatalf("Start: %v", err)
	}

	info, err := b.Inspect(ctx, id)
	if err != nil {
		t.Fatalf("Inspect after Start: %v", err)
	}
	if info.Status != "running" {
		t.Errorf("status after Start = %q, want running", info.Status)
	}

	if err := b.Stop(ctx, id); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	info, err = b.Inspect(ctx, id)
	if err != nil {
		t.Fatalf("Inspect after Stop: %v", err)
	}
	if info.Status != "exited" {
		t.Errorf("status after Stop = %q, want exited", info.Status)
	}
}

func TestExec(t *testing.T) {
	b, ctx := testBackend(t)
	id := mustCreate(t, b, ctx, &config.UniverseConfig{Image: testImage})

	if err := b.Start(ctx, id); err != nil {
		t.Fatalf("Start: %v", err)
	}

	result, err := b.Exec(ctx, id, []string{"echo", "hello"})
	if err != nil {
		t.Fatalf("Exec: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("exit code = %d, want 0", result.ExitCode)
	}
	if got := strings.TrimSpace(result.Stdout); got != "hello" {
		t.Errorf("stdout = %q, want %q", got, "hello")
	}
	if result.Stderr != "" {
		t.Errorf("stderr = %q, want empty", result.Stderr)
	}
}

func TestExecFailure(t *testing.T) {
	b, ctx := testBackend(t)
	id := mustCreate(t, b, ctx, &config.UniverseConfig{Image: testImage})

	if err := b.Start(ctx, id); err != nil {
		t.Fatalf("Start: %v", err)
	}

	result, err := b.Exec(ctx, id, []string{"sh", "-c", "exit 42"})
	if err != nil {
		t.Fatalf("Exec: %v", err)
	}
	if result.ExitCode != 42 {
		t.Errorf("exit code = %d, want 42", result.ExitCode)
	}
}

func TestLabels(t *testing.T) {
	b, ctx := testBackend(t)
	id := mustCreate(t, b, ctx, &config.UniverseConfig{
		Image: testImage,
		Mind:  "test-mind",
	})

	info, err := b.Inspect(ctx, id)
	if err != nil {
		t.Fatalf("Inspect: %v", err)
	}
	if info.Mind != "test-mind" {
		t.Errorf("mind label = %q, want %q", info.Mind, "test-mind")
	}
}

func TestMountMind(t *testing.T) {
	b, ctx := testBackend(t)
	id := mustCreate(t, b, ctx, &config.UniverseConfig{
		Image: testImage,
		Mind:  "test-mind-mount",
	})

	if err := b.Start(ctx, id); err != nil {
		t.Fatalf("Start: %v", err)
	}

	result, err := b.Exec(ctx, id, []string{"ls", "/mind"})
	if err != nil {
		t.Fatalf("Exec ls /mind: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("/mind not accessible, exit code = %d, stderr = %s", result.ExitCode, result.Stderr)
	}
}

func TestMountWorkspace(t *testing.T) {
	b, ctx := testBackend(t)
	tmpDir := t.TempDir()
	id := mustCreate(t, b, ctx, &config.UniverseConfig{
		Image:     testImage,
		Workspace: tmpDir,
	})

	if err := b.Start(ctx, id); err != nil {
		t.Fatalf("Start: %v", err)
	}

	result, err := b.Exec(ctx, id, []string{"ls", "/workspace"})
	if err != nil {
		t.Fatalf("Exec ls /workspace: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("/workspace not accessible, exit code = %d, stderr = %s", result.ExitCode, result.Stderr)
	}
}

func TestList(t *testing.T) {
	b, ctx := testBackend(t)
	id1 := mustCreate(t, b, ctx, &config.UniverseConfig{Image: testImage})
	id2 := mustCreate(t, b, ctx, &config.UniverseConfig{Image: testImage})

	containers, err := b.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	found := map[string]bool{}
	for _, c := range containers {
		found[c.ID] = true
	}
	if !found[id1] {
		t.Errorf("List missing container %s", id1)
	}
	if !found[id2] {
		t.Errorf("List missing container %s", id2)
	}
}

func TestRemoveForce(t *testing.T) {
	b, ctx := testBackend(t)
	cfg := &config.UniverseConfig{Image: testImage}

	id, err := b.Create(ctx, cfg)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := b.Start(ctx, id); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Remove without stopping — should force remove.
	if err := b.Remove(ctx, id); err != nil {
		t.Fatalf("Remove (force): %v", err)
	}

	_, err = b.Inspect(ctx, id)
	if err == nil {
		t.Fatal("container should be gone after force remove")
	}
}

func TestCreateInvalidImage(t *testing.T) {
	b, ctx := testBackend(t)
	cfg := &config.UniverseConfig{Image: "nonexistent-image-that-does-not-exist:99.99"}

	_, err := b.Create(ctx, cfg)
	if err == nil {
		t.Fatal("Create with invalid image should error")
	}
}

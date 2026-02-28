//go:build integration

package e2e

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jterrazz/universe/internal/architect"
	"github.com/jterrazz/universe/internal/backend"
	"github.com/jterrazz/universe/internal/config"
)

func testArchitect(t *testing.T) (*architect.Architect, *backend.DockerBackend, context.Context) {
	t.Helper()
	b, ctx := TestBackend(t)
	a := architect.NewWithBackend(b)
	return a, b, ctx
}

// mustCreateUniverse creates a universe and registers cleanup via the backend.
func mustCreateUniverse(t *testing.T, a *architect.Architect, b backend.Backend, ctx context.Context, cfg *config.UniverseConfig) *config.Universe {
	t.Helper()
	u, err := a.Create(ctx, cfg)
	if err != nil {
		t.Fatalf("Architect.Create: %v", err)
	}
	t.Cleanup(func() {
		cleanCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		_ = b.Remove(cleanCtx, u.ID)
	})
	return u
}

func TestArchitectCreateAndDestroy(t *testing.T) {
	a, _, ctx := testArchitect(t)

	cfg := &config.UniverseConfig{Image: TestImage}
	u, err := a.Create(ctx, cfg)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if len(u.ID) < 8 {
		t.Fatalf("expected valid UUID, got %q", u.ID)
	}
	if u.Image != TestImage {
		t.Errorf("image = %q, want %q", u.Image, TestImage)
	}
	if u.Status != config.StatusCreated {
		t.Errorf("status = %q, want %q", u.Status, config.StatusCreated)
	}

	if err := a.Destroy(ctx, u.ID); err != nil {
		t.Fatalf("Destroy: %v", err)
	}
}

func TestArchitectCreateWritesPhysicsMD(t *testing.T) {
	a, b, ctx := testArchitect(t)
	cfg := &config.UniverseConfig{Image: TestImage}
	u := mustCreateUniverse(t, a, b, ctx, cfg)

	// Start container to read physics.md.
	if err := b.Start(ctx, u.ID); err != nil {
		t.Fatalf("Start: %v", err)
	}

	result, err := b.Exec(ctx, u.ID, []string{"cat", "/universe/physics.md"})
	if err != nil {
		t.Fatalf("Exec cat: %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("cat /universe/physics.md failed: exit %d, stderr: %s", result.ExitCode, result.Stderr)
	}
	if !strings.Contains(result.Stdout, TestImage) {
		t.Errorf("physics.md should contain image name %q, got:\n%s", TestImage, result.Stdout)
	}
}

func TestArchitectCreateWithMind(t *testing.T) {
	a, b, ctx := testArchitect(t)
	cfg := &config.UniverseConfig{
		Image: TestImage,
		Mind:  "test-e2e-mind",
	}
	u := mustCreateUniverse(t, a, b, ctx, cfg)

	if u.Mind != "test-e2e-mind" {
		t.Errorf("mind = %q, want %q", u.Mind, "test-e2e-mind")
	}

	// Start and verify /mind is accessible.
	if err := b.Start(ctx, u.ID); err != nil {
		t.Fatalf("Start: %v", err)
	}
	result, err := b.Exec(ctx, u.ID, []string{"ls", "/mind"})
	if err != nil {
		t.Fatalf("Exec ls /mind: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("/mind not accessible: exit %d, stderr: %s", result.ExitCode, result.Stderr)
	}
}

func TestArchitectCreateWithWorkspace(t *testing.T) {
	a, b, ctx := testArchitect(t)
	tmpDir := t.TempDir()
	cfg := &config.UniverseConfig{
		Image:     TestImage,
		Workspace: tmpDir,
	}
	u := mustCreateUniverse(t, a, b, ctx, cfg)

	// Start and verify /workspace is accessible.
	if err := b.Start(ctx, u.ID); err != nil {
		t.Fatalf("Start: %v", err)
	}
	result, err := b.Exec(ctx, u.ID, []string{"ls", "/workspace"})
	if err != nil {
		t.Fatalf("Exec ls /workspace: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("/workspace not accessible: exit %d, stderr: %s", result.ExitCode, result.Stderr)
	}
	_ = u
}

func TestArchitectList(t *testing.T) {
	a, b, ctx := testArchitect(t)
	u1 := mustCreateUniverse(t, a, b, ctx, &config.UniverseConfig{Image: TestImage})
	u2 := mustCreateUniverse(t, a, b, ctx, &config.UniverseConfig{Image: TestImage})

	universes, err := a.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	found := map[string]bool{}
	for _, u := range universes {
		found[u.ID] = true
	}
	if !found[u1.ID] {
		t.Errorf("List missing universe %s", u1.ID)
	}
	if !found[u2.ID] {
		t.Errorf("List missing universe %s", u2.ID)
	}
}

func TestArchitectInspect(t *testing.T) {
	a, b, ctx := testArchitect(t)
	created := mustCreateUniverse(t, a, b, ctx, &config.UniverseConfig{Image: TestImage})

	inspected, err := a.Inspect(ctx, created.ID)
	if err != nil {
		t.Fatalf("Inspect: %v", err)
	}
	if inspected.ID != created.ID {
		t.Errorf("Inspect ID = %q, want %q", inspected.ID, created.ID)
	}
	if inspected.Image != TestImage {
		t.Errorf("Inspect image = %q, want %q", inspected.Image, TestImage)
	}
}

func TestArchitectInspectNotFound(t *testing.T) {
	a, _, ctx := testArchitect(t)

	_, err := a.Inspect(ctx, "00000000-0000-0000-0000-000000000000")
	if err == nil {
		t.Fatal("Inspect with bogus ID should error")
	}
}

func TestArchitectDestroyNotFound(t *testing.T) {
	a, _, ctx := testArchitect(t)

	err := a.Destroy(ctx, "00000000-0000-0000-0000-000000000000")
	if err == nil {
		t.Fatal("Destroy with bogus ID should error")
	}
}

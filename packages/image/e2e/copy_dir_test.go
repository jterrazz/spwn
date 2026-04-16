//go:build e2e

package e2e

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"spwn.sh/packages/image/internal/imagetest"
)

// TestCopyDirTo_RoundTrip exercises the CopyDirTo / CopyDirFrom contract
// end-to-end against a real Docker container. This is the backbone of
// the new agent architecture: spawn uses CopyDirTo to seed /agents/<name>/
// from the host, and graceful down uses CopyDirFrom to pull allowlisted
// memory dirs back out. Both directions must survive files, nested
// subdirectories, and the tar-stream round trip.
func TestCopyDirTo_RoundTrip(t *testing.T) {
	s := imagetest.SpinUp(t, newRegistry(t), "@spwn/unix")
	be := s.Backend()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	// ── Arrange: a nested host tree resembling a real agent home ──
	host := t.TempDir()
	mustWrite(t, filepath.Join(host, "identity", "profile.md"), "I am neo.")
	mustWrite(t, filepath.Join(host, "knowledge", "world.md"), "The matrix has you.")
	mustWrite(t, filepath.Join(host, "playbooks", "greet.md"), "Hello, world.")

	// ── Act: copy the tree into the container at /agents/neo ──
	if err := be.CopyDirTo(ctx, s.ContainerID, "/agents/neo", host); err != nil {
		t.Fatalf("CopyDirTo: %v", err)
	}

	// ── Assert: every file is visible inside the container ──
	for relPath, want := range map[string]string{
		"/agents/neo/identity/profile.md": "I am neo.",
		"/agents/neo/knowledge/world.md":  "The matrix has you.",
		"/agents/neo/playbooks/greet.md":  "Hello, world.",
	} {
		if !s.FileExists(relPath) {
			t.Errorf("expected %s to exist in container after CopyDirTo", relPath)
			continue
		}
		got := strings.TrimSpace(s.ReadFile(relPath))
		if got != want {
			t.Errorf("%s content = %q, want %q", relPath, got, want)
		}
	}

	// ── Act: mutate one file inside the container to simulate a
	//        mid-session memory write, then copy the subdirectory out ──
	if _, code := s.Exec("echo 'Memory persists.' > /agents/neo/knowledge/learned.md"); code != 0 {
		t.Fatalf("container write failed")
	}

	hostOut := t.TempDir()
	if err := be.CopyDirFrom(ctx, s.ContainerID, "/agents/neo/knowledge", hostOut); err != nil {
		t.Fatalf("CopyDirFrom: %v", err)
	}

	// ── Assert: both the original file and the new one round-trip
	//        back to the host. The tar reader strips the source
	//        basename so files land directly under hostOut. ──
	for rel, want := range map[string]string{
		"world.md":   "The matrix has you.",
		"learned.md": "Memory persists.",
	} {
		b, err := os.ReadFile(filepath.Join(hostOut, rel))
		if err != nil {
			t.Errorf("read %s after CopyDirFrom: %v", rel, err)
			continue
		}
		if strings.TrimSpace(string(b)) != want {
			t.Errorf("%s content = %q, want %q", rel, strings.TrimSpace(string(b)), want)
		}
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

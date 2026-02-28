//go:build integration

package e2e

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/jterrazz/universe/internal/backend"
	"github.com/jterrazz/universe/internal/config"
)

const TestImage = "alpine:3.19"

// TestBackend returns a DockerBackend and context with a 60s timeout.
// Fatals if Docker is unreachable.
func TestBackend(t *testing.T) (*backend.DockerBackend, context.Context) {
	t.Helper()
	b, err := backend.NewDockerBackend()
	if err != nil {
		t.Fatalf("cannot connect to Docker: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)
	return b, ctx
}

// MustCreate creates a container and registers t.Cleanup for removal.
func MustCreate(t *testing.T, b backend.Backend, ctx context.Context, cfg *config.UniverseConfig) string {
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

// MustCreateAndStart creates and starts a container, registering cleanup.
func MustCreateAndStart(t *testing.T, b backend.Backend, ctx context.Context, cfg *config.UniverseConfig) string {
	t.Helper()
	id := MustCreate(t, b, ctx, cfg)
	if err := b.Start(ctx, id); err != nil {
		t.Fatalf("start container: %v", err)
	}
	return id
}

// CLIResult holds the output of a CLI command execution.
type CLIResult struct {
	Stdout string
	Stderr string
	Err    error
}

// RunCLI executes the compiled universe binary with the given args.
func RunCLI(t *testing.T, args ...string) CLIResult {
	t.Helper()
	cmd := exec.Command(binaryPath, args...)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return CLIResult{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
		Err:    err,
	}
}

// MustRunCLI executes the compiled binary and fatals on error.
func MustRunCLI(t *testing.T, args ...string) CLIResult {
	t.Helper()
	r := RunCLI(t, args...)
	if r.Err != nil {
		t.Fatalf("CLI %v failed: %v\nstdout: %s\nstderr: %s", args, r.Err, r.Stdout, r.Stderr)
	}
	return r
}

var uuidRe = regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)

// ExtractUniverseID parses a UUID from CLI output like "Universe created: <uuid>".
func ExtractUniverseID(t *testing.T, output string) string {
	t.Helper()
	match := uuidRe.FindString(output)
	if match == "" {
		t.Fatalf("no UUID found in output: %s", output)
	}
	return match
}

// CLICleanup registers a t.Cleanup that runs `universe destroy <id>`.
func CLICleanup(t *testing.T, id string) {
	t.Helper()
	t.Cleanup(func() {
		cmd := exec.Command(binaryPath, "destroy", id)
		_ = cmd.Run()
	})
}

// AssertContains fails if s does not contain substr.
func AssertContains(t *testing.T, s, substr string) {
	t.Helper()
	if !strings.Contains(s, substr) {
		t.Errorf("expected output to contain %q, got:\n%s", substr, s)
	}
}

// AssertNotContains fails if s contains substr.
func AssertNotContains(t *testing.T, s, substr string) {
	t.Helper()
	if strings.Contains(s, substr) {
		t.Errorf("expected output NOT to contain %q, got:\n%s", substr, s)
	}
}

// sweepTestContainers removes all containers with the universe.id label and test image.
func sweepTestContainers() {
	out, err := exec.Command("docker", "ps", "-a",
		"--filter", "label=universe.id",
		"--filter", fmt.Sprintf("ancestor=%s", TestImage),
		"-q").Output()
	if err != nil || len(out) == 0 {
		return
	}
	ids := strings.Fields(string(out))
	if len(ids) > 0 {
		args := append([]string{"rm", "-f"}, ids...)
		_ = exec.Command("docker", args...).Run()
	}
}

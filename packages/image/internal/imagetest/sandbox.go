package imagetest

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	ib "spwn.sh/packages/image"
	"spwn.sh/packages/image/backend"
	"spwn.sh/packages/image/base"
)

// Sandbox is a running container built from specific tools, used for E2E testing.
type Sandbox struct {
	ContainerID string
	ImageTag    string
	Tools       []string
	backend     backend.Backend
	t           *testing.T
}

// SpinUp builds an image with the given registry and tools, then starts a container.
// The container is cleaned up automatically via t.Cleanup.
func SpinUp(t *testing.T, reg *ib.Registry, tools ...string) *Sandbox {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	t.Cleanup(cancel)

	docker, err := backend.NewDocker()
	if err != nil {
		t.Fatalf("docker backend: %v", err)
	}

	builder := ib.New(reg, docker)
	tag := fmt.Sprintf("spwn-test:%d", time.Now().UnixNano())

	result, err := builder.Build(ctx, ib.BuildRequest{
		BaseDockerfile: base.WorldDockerfile,
		Tools:          tools,
		Tag:            tag,
		SkipVerify:     true,
		LogWriter:      testWriter{t},
	})
	if err != nil {
		t.Fatalf("build image: %v", err)
	}

	containerName := fmt.Sprintf("spwn-e2e-%d", time.Now().UnixNano())
	containerID, err := docker.Create(ctx, backend.ContainerConfig{
		Image: result.Tag,
		Name:  containerName,
	})
	if err != nil {
		t.Fatalf("create container: %v", err)
	}

	if err := docker.Start(ctx, containerID); err != nil {
		docker.Remove(ctx, containerID)
		t.Fatalf("start container: %v", err)
	}

	s := &Sandbox{
		ContainerID: containerID,
		ImageTag:    result.Tag,
		Tools:       result.Tools,
		backend:     docker,
		t:           t,
	}

	t.Cleanup(func() { s.Teardown() })
	return s
}

// Exec runs a command inside the container and returns stdout and exit code.
func (s *Sandbox) Exec(cmd string) (string, int) {
	s.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	output, err := s.backend.ExecOutput(ctx, s.ContainerID, []string{"sh", "-c", cmd})
	if err != nil {
		return output, 1
	}
	return output, 0
}

// FileExists checks if a path exists inside the container.
func (s *Sandbox) FileExists(path string) bool {
	_, exitCode := s.Exec(fmt.Sprintf("test -e %s", path))
	return exitCode == 0
}

// ReadFile reads a file from inside the container.
func (s *Sandbox) ReadFile(path string) string {
	output, _ := s.Exec(fmt.Sprintf("cat %s", path))
	return output
}

// Backend exposes the underlying Backend for tests that need to
// exercise transport methods (CopyDirTo, CopyDirFrom, CopyTo) against
// the running container directly. Most test helpers should prefer the
// higher-level Sandbox methods above.
func (s *Sandbox) Backend() backend.Backend { return s.backend }

// Teardown stops and removes the container and image.
func (s *Sandbox) Teardown() {
	ctx := context.Background()
	_ = s.backend.Stop(ctx, s.ContainerID)
	_ = s.backend.Remove(ctx, s.ContainerID)
	_ = s.backend.ImageRemove(ctx, s.ImageTag)
}

// AssertBinaryExists checks that a binary is available in the container.
func AssertBinaryExists(t *testing.T, s *Sandbox, binary string) {
	t.Helper()
	_, exitCode := s.Exec(fmt.Sprintf("command -v %s", binary))
	if exitCode != 0 {
		t.Errorf("binary %q not found in container", binary)
	}
}

// AssertBinaryVersion runs <binary> <flag> and checks output contains the expected substring.
func AssertBinaryVersion(t *testing.T, s *Sandbox, binary, flag, contains string) {
	t.Helper()
	output, exitCode := s.Exec(fmt.Sprintf("%s %s 2>&1", binary, flag))
	if exitCode != 0 {
		t.Errorf("%s %s failed (exit %d): %s", binary, flag, exitCode, output)
		return
	}
	if !strings.Contains(output, contains) {
		t.Errorf("%s %s output %q does not contain %q", binary, flag, output, contains)
	}
}

// AssertSkillInstalled checks that a tool's SKILL.md exists in the container.
func AssertSkillInstalled(t *testing.T, s *Sandbox, toolName string) {
	t.Helper()
	name := strings.TrimPrefix(toolName, "@")
	path := fmt.Sprintf("/world/skills/%s/SKILL.md", name)
	if !s.FileExists(path) {
		t.Errorf("skill not found at %s", path)
	}
}

// AssertSkillContains checks that a tool's SKILL.md contains a substring.
func AssertSkillContains(t *testing.T, s *Sandbox, toolName, substring string) {
	t.Helper()
	name := strings.TrimPrefix(toolName, "@")
	content := s.ReadFile(fmt.Sprintf("/world/skills/%s/SKILL.md", name))
	if !strings.Contains(content, substring) {
		t.Errorf("skill for %s does not contain %q", toolName, substring)
	}
}

// AssertFileExists checks that a file exists in the container.
func AssertFileExists(t *testing.T, s *Sandbox, path string) {
	t.Helper()
	if !s.FileExists(path) {
		t.Errorf("file %s not found", path)
	}
}

// AssertFileContains checks that a file in the container contains a substring.
func AssertFileContains(t *testing.T, s *Sandbox, path, substring string) {
	t.Helper()
	content := s.ReadFile(path)
	if !strings.Contains(content, substring) {
		t.Errorf("file %s does not contain %q", path, substring)
	}
}

type testWriter struct{ t *testing.T }

func (w testWriter) Write(p []byte) (int, error) {
	w.t.Log(strings.TrimSpace(string(p)))
	return len(p), nil
}

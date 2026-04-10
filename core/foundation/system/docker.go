// Package system provides host environment checks shared by the CLI doctor
// command and the observatory API. Kept dependency-free (uses exec only) so
// it can live inside the foundation module.
package system

import (
	"context"
	"errors"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// DockerStatus describes the health of the local Docker daemon.
type DockerStatus struct {
	// Installed is true if a `docker` binary is on PATH.
	Installed bool `json:"installed"`
	// Running is true if `docker info` returned successfully.
	Running bool `json:"running"`
	// Version is the server version reported by Docker (empty if unknown).
	Version string `json:"version,omitempty"`
	// Error is a human-readable description of why Docker is not usable.
	Error string `json:"error,omitempty"`
	// Hint is an actionable next step the user can take.
	Hint string `json:"hint,omitempty"`
	// Platform is the OS we are running on (used to tailor the hint).
	Platform string `json:"platform"`
}

// CheckDocker probes the local Docker daemon and returns a DockerStatus.
// Always returns a value — never nil — so callers can render the result
// directly without nil checks.
func CheckDocker(ctx context.Context) DockerStatus {
	st := DockerStatus{Platform: runtime.GOOS}

	if _, err := exec.LookPath("docker"); err != nil {
		st.Error = "docker CLI not found on PATH"
		st.Hint = installDockerHint(runtime.GOOS)
		return st
	}
	st.Installed = true

	cctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	out, err := exec.CommandContext(cctx, "docker", "info", "--format", "{{.ServerVersion}}").Output()
	if err != nil {
		st.Error = daemonDownMessage(err)
		st.Hint = startDockerHint(runtime.GOOS)
		return st
	}

	st.Running = true
	st.Version = strings.TrimSpace(string(out))
	return st
}

// OK reports whether Docker is fully usable.
func (s DockerStatus) OK() bool { return s.Installed && s.Running }

// Summary returns a one-line human description for terminal output.
func (s DockerStatus) Summary() string {
	switch {
	case !s.Installed:
		return "not installed"
	case !s.Running:
		return "not running"
	case s.Version != "":
		return "running (v" + s.Version + ")"
	default:
		return "running"
	}
}

func daemonDownMessage(err error) string {
	var ee *exec.ExitError
	if errors.As(err, &ee) && len(ee.Stderr) > 0 {
		// Trim noisy lines, keep the first useful one.
		for _, line := range strings.Split(string(ee.Stderr), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "Client:") {
				continue
			}
			return line
		}
	}
	return "docker daemon is not reachable"
}

func startDockerHint(os string) string {
	switch os {
	case "darwin":
		return "Open Docker Desktop, then retry."
	case "linux":
		return "Start the daemon: sudo systemctl start docker"
	case "windows":
		return "Open Docker Desktop, then retry."
	}
	return "Start your Docker daemon and retry."
}

func installDockerHint(os string) string {
	switch os {
	case "darwin":
		return "Install Docker Desktop: https://www.docker.com/products/docker-desktop/"
	case "linux":
		return "Install Docker Engine: https://docs.docker.com/engine/install/"
	case "windows":
		return "Install Docker Desktop: https://www.docker.com/products/docker-desktop/"
	}
	return "Install Docker: https://docs.docker.com/get-docker/"
}

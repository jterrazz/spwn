// Package system provides host environment checks shared by the CLI doctor
// command and the observatory API. Kept dependency-free (uses exec only) so
// it can live inside the foundation module.
package system

import (
	"context"
	"errors"
	"os"
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
	// Socket is the unix socket path that successfully answered, when known.
	// Empty when probing failed or when the daemon was reached over a
	// non-unix endpoint.
	Socket string `json:"socket,omitempty"`
}

// CheckDocker probes the local Docker daemon and returns a DockerStatus.
// Always returns a value — never nil — so callers can render the result
// directly without nil checks.
//
// The probe tries the user's default `docker` configuration first (which
// honors DOCKER_HOST and the active docker context). If that fails, it
// falls through a list of well-known socket paths used by OrbStack,
// Colima, Docker Desktop and the system default. This is what makes the
// retry button "just work" after a daemon restart even when the spwn app
// was launched from Finder/Dock without the user's terminal env (which
// is the common cause of "OrbStack restarted but spwn can't see it").
func CheckDocker(ctx context.Context) DockerStatus {
	st := DockerStatus{Platform: runtime.GOOS}

	// Locate the docker binary. We can't rely on exec.LookPath alone:
	// when the spwn desktop app is launched from Finder/Dock the PATH
	// it inherits is the bare system PATH (no /usr/local/bin, no
	// /opt/homebrew/bin, no ~/.orbstack/bin), so user-only Docker
	// installs would falsely report "not installed". findDockerBinary
	// falls back to known install locations.
	dockerBin := findDockerBinary()
	if dockerBin == "" {
		st.Error = "docker CLI not found on PATH"
		st.Hint = installDockerHint(runtime.GOOS)
		return st
	}
	st.Installed = true

	// 1. Default attempt — honors whatever DOCKER_HOST / docker context
	// the spawning environment provides.
	if res := tryDockerInfo(ctx, dockerBin, "", 4*time.Second); res.ok {
		st.Running = true
		st.Version = res.version
		return st
	}

	// 2. Fallback: probe known socket paths sequentially. Whichever one
	// answers wins. Each attempt has a tight timeout so the worst-case
	// total stays under the API client's 10s window.
	for _, sock := range candidateSockets() {
		res := tryDockerInfo(ctx, dockerBin, sock, 1500*time.Millisecond)
		if !res.ok {
			continue
		}
		st.Running = true
		st.Version = res.version
		st.Socket = strings.TrimPrefix(sock, "unix://")
		return st
	}

	// Nothing worked — surface the original failure (from the default
	// attempt) so the user sees an honest error rather than the last
	// fallback's "no such file" noise.
	if res := tryDockerInfo(ctx, dockerBin, "", 1500*time.Millisecond); res.err != nil {
		st.Error = daemonDownMessage(res.err)
	} else {
		st.Error = "docker daemon is not reachable"
	}
	st.Hint = startDockerHint(runtime.GOOS)
	return st
}

type dockerProbeResult struct {
	version string
	err     error
	ok      bool
}

// tryDockerInfo runs `docker info` (optionally pinned to a specific
// DOCKER_HOST) and returns the parsed result. dockerBin is the absolute
// path returned by findDockerBinary; when dockerHost is empty, the
// inherited environment is used unchanged.
func tryDockerInfo(ctx context.Context, dockerBin, dockerHost string, timeout time.Duration) dockerProbeResult {
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(cctx, dockerBin, "info", "--format", "{{.ServerVersion}}")
	if dockerHost != "" {
		cmd.Env = append(os.Environ(), "DOCKER_HOST="+dockerHost)
	}
	out, err := cmd.Output()
	if err != nil {
		return dockerProbeResult{err: err}
	}
	return dockerProbeResult{version: strings.TrimSpace(string(out)), ok: true}
}

// findDockerBinary returns an absolute path to a usable `docker` binary,
// or "" if none can be found. It tries the inherited PATH first, then
// falls back to well-known install locations used by user-only Docker
// installs (OrbStack, Colima, brew, Docker Desktop). This is what makes
// the desktop app work when launched from Finder/Dock instead of a
// shell — Finder gives processes the bare system PATH which excludes
// every common Docker install location.
func findDockerBinary() string {
	if path, err := exec.LookPath("docker"); err == nil {
		return path
	}

	candidates := []string{}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		candidates = append(candidates,
			home+"/.orbstack/bin/docker",
			home+"/.docker/bin/docker",
			home+"/.rd/bin/docker",
		)
	}
	candidates = append(candidates,
		"/Applications/OrbStack.app/Contents/MacOS/xbin/docker",
		"/Applications/Docker.app/Contents/Resources/bin/docker",
		"/Applications/Rancher Desktop.app/Contents/Resources/resources/darwin/bin/docker",
		"/opt/homebrew/bin/docker", // Apple-Silicon Homebrew
		"/usr/local/bin/docker",    // Intel Homebrew / Docker Desktop default
		"/opt/local/bin/docker",    // MacPorts
	)
	for _, p := range candidates {
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			return p
		}
	}
	return ""
}

// candidateSockets returns the list of well-known docker socket paths
// to probe when the default `docker info` attempt has failed. Order
// matters — we put per-user sockets first so we never accidentally
// connect to a system-wide daemon the user wasn't using.
func candidateSockets() []string {
	home, err := os.UserHomeDir()
	candidates := []string{}
	if err == nil && home != "" {
		candidates = append(candidates,
			"unix://"+home+"/.orbstack/run/docker.sock",         // OrbStack
			"unix://"+home+"/.colima/default/docker.sock",       // Colima default profile
			"unix://"+home+"/.docker/run/docker.sock",           // Docker Desktop (macOS)
			"unix://"+home+"/.rd/docker.sock",                   // Rancher Desktop
		)
	}
	candidates = append(candidates, "unix:///var/run/docker.sock")
	return candidates
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
		return "Open Docker Desktop or OrbStack, then retry."
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

// Package probe contains environment probes that talk to the Docker
// daemon directly via the official engine SDK. This is the canonical
// place to ask "is Docker reachable, and if so where" — both the CLI
// doctor command and the observatory API consume this package so the
// answer is always identical.
//
// Why the SDK and not `docker info`?
//
//   - The SDK speaks the engine HTTP API natively, so we don't depend on
//     a `docker` CLI binary being on PATH. Users with user-only installs
//     of OrbStack, Colima, Rancher Desktop, Lima or rootless Podman all
//     "just work" without LSEnvironment hacks in the Tauri app bundle.
//   - The SDK negotiates the API version automatically, so we never see
//     "client newer than server" errors against an older engine.
//   - We get a real ping + ServerVersion call instead of parsing CLI
//     output, which means errors are typed (network, perm, version
//     mismatch, …) rather than scraped strings.
//
// The probe is best-effort: it tries the user's default configuration
// first (DOCKER_HOST, ~/.docker/config.json, contexts) and only falls
// through to the well-known per-user socket paths if that fails. The
// fallback list covers every common runtime.
package probe

import (
	"context"
	"errors"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/docker/docker/client"
)

// DockerStatus describes the health of the local Docker daemon.
type DockerStatus struct {
	// Installed is true once we have proven the daemon is reachable.
	// We deliberately do not separate "binary present" from "daemon up"
	// — spwn talks to the daemon directly, so binary presence is
	// irrelevant. Field kept for JSON backwards compatibility with
	// older clients of the API.
	Installed bool `json:"installed"`
	// Running is true if the daemon answered a ping.
	Running bool `json:"running"`
	// Version is the engine version reported by /version.
	Version string `json:"version,omitempty"`
	// APIVersion is the negotiated engine API version.
	APIVersion string `json:"apiVersion,omitempty"`
	// Host is the URL the SDK ended up talking to (e.g. unix:///…/docker.sock).
	Host string `json:"host,omitempty"`
	// Socket is the unix socket path that answered, when applicable.
	// Empty for TCP/named-pipe daemons.
	Socket string `json:"socket,omitempty"`
	// Runtime is a friendly label inferred from the host path:
	// "OrbStack", "Colima", "Docker Desktop", "Rancher Desktop",
	// "Lima", "Podman", or "Docker" when unknown.
	Runtime string `json:"runtime,omitempty"`
	// Error is a human-readable description of why Docker is not usable.
	Error string `json:"error,omitempty"`
	// Hint is an actionable next step the user can take.
	Hint string `json:"hint,omitempty"`
	// Platform is the OS we are running on (used to tailor the hint).
	Platform string `json:"platform"`
}

// OK reports whether Docker is fully usable.
func (s DockerStatus) OK() bool { return s.Running }

// Summary returns a one-line human description for terminal output.
func (s DockerStatus) Summary() string {
	if !s.Running {
		return "not running"
	}
	parts := []string{}
	if s.Runtime != "" {
		parts = append(parts, s.Runtime)
	}
	if s.Version != "" {
		parts = append(parts, "v"+s.Version)
	}
	if len(parts) == 0 {
		return "running"
	}
	return "running (" + strings.Join(parts, " ") + ")"
}

// CheckDocker probes the local Docker daemon and returns a DockerStatus.
// Always returns a value — never nil — so callers can render the result
// directly without nil checks. The returned status is the result of the
// FIRST host that answered; subsequent hosts are not contacted.
func CheckDocker(ctx context.Context) DockerStatus {
	st := DockerStatus{Platform: runtime.GOOS}

	// 1. Default attempt — honors DOCKER_HOST, ~/.docker/contexts, etc.
	if res := tryHost(ctx, "", 4*time.Second); res.ok {
		return res.apply(st)
	}

	// 2. Probe well-known per-user sockets sequentially. Each attempt is
	// short so the worst-case total stays well under the 10s API window.
	var lastErr error
	for _, host := range candidateHosts() {
		// Cheap pre-check for unix sockets: skip the SDK round-trip if
		// the file does not exist or is not a socket.
		if path, ok := unixPath(host); ok {
			info, err := os.Stat(path)
			if err != nil || info.Mode()&os.ModeSocket == 0 {
				continue
			}
		}
		res := tryHost(ctx, host, 1500*time.Millisecond)
		if res.ok {
			return res.apply(st)
		}
		if res.err != nil {
			lastErr = res.err
		}
	}

	// Nothing answered. Surface a single, honest error message —
	// re-using the daemon error from the default attempt is more useful
	// than the last fallback's generic "no such file" noise.
	if defRes := tryHost(ctx, "", 1500*time.Millisecond); defRes.err != nil {
		st.Error = humanizeDaemonError(defRes.err)
	} else if lastErr != nil {
		st.Error = humanizeDaemonError(lastErr)
	} else {
		st.Error = "docker daemon is not reachable"
	}
	st.Hint = startDockerHint(runtime.GOOS)
	return st
}

// ── Internals ─────────────────────────────────────────────────────────

type probeResult struct {
	ok         bool
	err        error
	version    string
	apiVersion string
	host       string
}

// apply copies the successful probe result into a DockerStatus base.
func (r probeResult) apply(base DockerStatus) DockerStatus {
	base.Installed = true
	base.Running = true
	base.Version = r.version
	base.APIVersion = r.apiVersion
	base.Host = r.host
	if path, ok := unixPath(r.host); ok {
		base.Socket = path
	}
	base.Runtime = inferRuntime(r.host)
	return base
}

// tryHost creates a one-shot Docker SDK client pointed at the given
// host (empty = inherit from environment) and pings it.
func tryHost(ctx context.Context, host string, timeout time.Duration) probeResult {
	opts := []client.Opt{client.WithAPIVersionNegotiation()}
	if host == "" {
		opts = append(opts, client.FromEnv)
	} else {
		opts = append(opts, client.WithHost(host))
	}
	cli, err := client.NewClientWithOpts(opts...)
	if err != nil {
		return probeResult{err: err}
	}
	defer cli.Close()

	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ping, err := cli.Ping(cctx)
	if err != nil {
		return probeResult{err: err, host: cli.DaemonHost()}
	}

	// ServerVersion is non-fatal — Ping is the source of truth, but
	// version is nice to have. Use the ping context so we don't double
	// the timeout budget.
	var version string
	if v, err := cli.ServerVersion(cctx); err == nil {
		version = v.Version
	}

	return probeResult{
		ok:         true,
		version:    version,
		apiVersion: ping.APIVersion,
		host:       cli.DaemonHost(),
	}
}

// candidateHosts returns the well-known per-user socket paths for every
// common Docker runtime, plus the system default. Order matters — we
// put per-user sockets first so we never accidentally connect to a
// system-wide daemon the user wasn't using.
func candidateHosts() []string {
	hosts := []string{}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		hosts = append(hosts,
			"unix://"+home+"/.orbstack/run/docker.sock",         // OrbStack
			"unix://"+home+"/.colima/default/docker.sock",       // Colima default profile
			"unix://"+home+"/.docker/run/docker.sock",           // Docker Desktop (per-user)
			"unix://"+home+"/.rd/docker.sock",                   // Rancher Desktop
			"unix://"+home+"/.lima/default/sock/docker.sock",    // Lima
		)
	}
	hosts = append(hosts,
		"unix:///var/run/docker.sock", // System Docker / Linux default
	)

	// Linux rootless installs use $XDG_RUNTIME_DIR/docker.sock
	// (typically /run/user/<uid>/docker.sock).
	if runtime.GOOS == "linux" {
		if xdg := os.Getenv("XDG_RUNTIME_DIR"); xdg != "" {
			hosts = append(hosts, "unix://"+xdg+"/docker.sock")
			hosts = append(hosts, "unix://"+xdg+"/podman/podman.sock")
		}
	}
	return hosts
}

// unixPath returns the filesystem path of a unix:// host, and a flag
// indicating whether the input was a unix host at all.
func unixPath(host string) (string, bool) {
	if !strings.HasPrefix(host, "unix://") {
		return "", false
	}
	return strings.TrimPrefix(host, "unix://"), true
}

// inferRuntime guesses a friendly runtime name from the answering
// daemon's socket path. Best-effort: returns "Docker" when no marker
// is recognised, and "" when host is empty.
func inferRuntime(host string) string {
	if host == "" {
		return ""
	}
	low := strings.ToLower(host)
	switch {
	case strings.Contains(low, "orbstack"):
		return "OrbStack"
	case strings.Contains(low, "colima"):
		return "Colima"
	case strings.Contains(low, ".rd/"):
		return "Rancher Desktop"
	case strings.Contains(low, "lima"):
		return "Lima"
	case strings.Contains(low, "podman"):
		return "Podman"
	case strings.Contains(low, "/.docker/"):
		return "Docker Desktop"
	}
	return "Docker"
}

// humanizeDaemonError turns the SDK's wrapped error into something a
// non-engineer can act on. The SDK wraps net.OpError + syscall errno;
// we recognise the most common shapes.
func humanizeDaemonError(err error) string {
	if err == nil {
		return "docker daemon is not reachable"
	}
	msg := err.Error()
	low := strings.ToLower(msg)
	switch {
	case strings.Contains(low, "no such file or directory"),
		strings.Contains(low, "connect: connection refused"):
		return "no docker daemon listening on the configured socket"
	case strings.Contains(low, "permission denied"):
		return "permission denied talking to the docker socket"
	case strings.Contains(low, "context deadline exceeded"),
		errors.Is(err, context.DeadlineExceeded):
		return "docker daemon did not respond in time"
	}
	// Fall back to the last meaningful line of the error.
	for _, line := range strings.Split(msg, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
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


package gate

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// Container/image identity. Stable across upgrades so docker
// recognises restarts vs new installs.
const (
	ImageName     = "spwn-gate:latest"
	ContainerName = "spwn-gate"
	HostPort      = "9000"
)

// EnsureRunning brings the gate to a "running" state idempotently:
//   - if the image is missing, build it from source
//   - if the container is missing, create + start it
//   - if it exists but stopped, start it
//   - if it's already running, no-op
//
// Safe to call from spwn up's hot path on every spawn — running case
// is a single docker inspect.
func EnsureRunning(ctx context.Context, w io.Writer) error {
	return ensureRunning(ctx, w, false)
}

// EnsureRunningRebuild is like EnsureRunning but also forces an
// image rebuild. Use after a binary upgrade, otherwise the existing
// container keeps running the previous gate binary indefinitely.
func EnsureRunningRebuild(ctx context.Context, w io.Writer) error {
	return ensureRunning(ctx, w, true)
}

func ensureRunning(ctx context.Context, w io.Writer, forceRebuild bool) error {
	if w == nil {
		w = io.Discard
	}

	if forceRebuild {
		// Stop+rm so the new image takes effect, then drop the image
		// so build runs unconditionally. Errors are non-fatal: missing
		// container/image is the success state we're heading toward.
		_ = Stop(ctx)
		_ = removeImage(ctx)
	}

	if running, err := isRunning(ctx); err != nil {
		return fmt.Errorf("check container: %w", err)
	} else if running {
		return nil
	}

	// Container exists but stopped → start it.
	if exists, err := containerExists(ctx); err != nil {
		return fmt.Errorf("check container: %w", err)
	} else if exists {
		fmt.Fprintln(w, "starting existing spwn-gate container")
		if err := dockerCmd(ctx, "start", ContainerName); err != nil {
			return err
		}
		return nil
	}

	// Container doesn't exist. Ensure image, then run.
	if has, err := imageExists(ctx); err != nil {
		return fmt.Errorf("check image: %w", err)
	} else if !has {
		fmt.Fprintln(w, "building spwn-gate image (one-time)…")
		if err := buildImage(ctx, w); err != nil {
			return fmt.Errorf("build image: %w", err)
		}
	}

	fmt.Fprintln(w, "creating + starting spwn-gate container")
	return runContainer(ctx)
}

func removeImage(ctx context.Context) error {
	return exec.CommandContext(ctx, "docker", "image", "rm", "-f", ImageName).Run()
}

// Stop stops + removes the gate container. Image stays on disk so the
// next start is fast.
func Stop(ctx context.Context) error {
	if exists, err := containerExists(ctx); err != nil {
		return err
	} else if !exists {
		return nil
	}
	if running, _ := isRunning(ctx); running {
		if err := dockerCmd(ctx, "stop", ContainerName); err != nil {
			return err
		}
	}
	return dockerCmd(ctx, "rm", ContainerName)
}

// Status returns the docker container state ("running", "stopped",
// "missing"), useful for `spwn gate status`.
func Status(ctx context.Context) (string, error) {
	exists, err := containerExists(ctx)
	if err != nil {
		return "", err
	}
	if !exists {
		return "missing", nil
	}
	if r, err := isRunning(ctx); err != nil {
		return "", err
	} else if r {
		return "running", nil
	}
	return "stopped", nil
}

// LogsCmd returns the prepared docker logs command so callers can
// stream output to the user's terminal.
func LogsCmd(ctx context.Context, follow bool, tail int) *exec.Cmd {
	args := []string{"logs"}
	if tail > 0 {
		args = append(args, "--tail", fmt.Sprintf("%d", tail))
	}
	if follow {
		args = append(args, "-f")
	}
	args = append(args, ContainerName)
	return exec.CommandContext(ctx, "docker", args...)
}

// --- internals ---

func runContainer(ctx context.Context) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	credsDir := filepath.Join(home, ".spwn", "credentials")
	gateDir := filepath.Join(home, ".spwn", "gate")
	spwnHome := filepath.Join(home, ".spwn")
	_ = os.MkdirAll(credsDir, 0o700)
	_ = os.MkdirAll(gateDir, 0o700)

	args := []string{
		"run", "-d",
		"--name", ContainerName,
		"--restart", "unless-stopped",
		"-p", "127.0.0.1:" + HostPort + ":" + HostPort,
		// Gate is the credential BROKER — owns the tokens, refreshes
		// them on its own schedule, and rotates them in-place via
		// atomic-replace (write tmp + rename). Worlds get the same
		// path as :ro; only the gate gets rw.
		"-v", credsDir + ":/credentials",
		"-v", gateDir + ":/gate",
		// Match the in-container default cred path; the gate reads
		// `mcp.ProviderTokenPath` which expands to /credentials/mcp/...
		"-e", "SPWN_HOME=/",
	}

	// Orchestration broker mounts. Optional — if any of these aren't
	// resolvable, the gate still starts (cookie sync + element
	// brokering work without them); the spwn:dispatch tool inside
	// just fails fast with a clear error when invoked.
	//
	// Three things a dispatch tool needs to call `spwn agent talk`:
	//   1. The spwn binary on PATH (linux ELF, cross-compiled on host)
	//   2. /var/run/docker.sock so spwn can `docker exec` into worlds
	//   3. ~/.spwn/state.json + activity.jsonl so spwn finds the live
	//      world IDs bound to the project on the host
	if spwnBin, ok := resolveLinuxSpwnBinary(); ok {
		args = append(args,
			"-v", spwnBin+":/usr/local/bin/spwn:ro",
		)
	}
	if sockPath, gid, ok := resolveDockerSocket(); ok {
		args = append(args,
			"-v", sockPath+":/var/run/docker.sock",
			"--group-add", gid,
			// OrbStack + Docker Desktop both expose the bind-mounted
			// socket as root-owned inside the container regardless of
			// the host file's GID. Add group 0 (root) so the gate's
			// non-root spwn user can read the socket. On a true Linux
			// host this is a no-op; on macOS it's the only thing that
			// makes `docker ps` work from inside the gate.
			"--group-add", "0",
		)
	}
	// Mount host ~/.spwn into the gate user's home so spwn-the-binary
	// finds its state. The gate runs as user `spwn` (see Dockerfile)
	// whose home is /home/spwn — that's where spwn looks for ~/.spwn.
	args = append(args,
		"-v", spwnHome+":/home/spwn/.spwn",
	)
	// Mirror-mount the user's home directory at the same path inside
	// the gate (read-only). The spwn:dispatch tool needs to chdir to
	// a project workspace before invoking `spwn agent talk`, and it
	// reads the workspace host path from the target world container's
	// labels (sh.spwn.world.workspaces). Mirroring the path keeps the
	// label data directly usable — no host→container translation.
	//
	// Trust note: the gate is already a privileged broker (owns OAuth
	// tokens, talks to Docker daemon). Read-only access to the user's
	// home extends nothing meaningful in the threat model.
	args = append(args,
		"-v", home+":"+home+":ro",
	)

	args = append(args, ImageName)
	return dockerCmd(ctx, args...)
}

// resolveLinuxSpwnBinary returns the absolute host path of a Linux
// spwn binary suitable for bind-mounting into the gate. The gate
// container is linux/<host-arch>; the host's macOS Mach-O binary
// would yield "Exec format error" inside the container.
//
// Order of resolution:
//  1. SPWN_BINARY env var — must point to a Linux ELF (test seam).
//  2. Cached cross-compile at ~/.spwn/cache/spwn-linux. If missing
//     OR older than 24h, rebuild via the same go-build path the
//     gate-binary build uses. Falls back to stale cache on rebuild
//     failure (a stale binary beats no binary).
//
// Returns ok=false on any unrecoverable failure. The gate still
// starts; only the spwn:dispatch tool inside loses functionality.
func resolveLinuxSpwnBinary() (string, bool) {
	if env := os.Getenv("SPWN_BINARY"); env != "" {
		if st, err := os.Stat(env); err == nil && !st.IsDir() {
			return env, true
		}
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", false
	}
	cacheDir := filepath.Join(home, ".spwn", "cache")
	cached := filepath.Join(cacheDir, "spwn-linux")

	if st, err := os.Stat(cached); err == nil && !st.IsDir() {
		if time.Since(st.ModTime()) < 24*time.Hour {
			return cached, true
		}
	}

	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return "", false
	}
	root, err := findSourceRoot()
	if err != nil {
		// Fall back to stale cache if we can't locate the source tree
		// (binary distribution case — no source on disk).
		if st, err2 := os.Stat(cached); err2 == nil && !st.IsDir() {
			return cached, true
		}
		return "", false
	}

	tmp, err := os.CreateTemp("", "spwn-linux-*")
	if err != nil {
		return "", false
	}
	tmp.Close()
	defer os.Remove(tmp.Name())

	build := exec.Command("go", "build",
		"-trimpath", "-ldflags=-s -w",
		"-o", tmp.Name(),
		"./apps/cli/cmd/spwn",
	)
	build.Dir = root
	build.Env = append(os.Environ(),
		"GOOS=linux",
		"GOARCH="+runtime.GOARCH,
		"CGO_ENABLED=0",
	)
	if err := build.Run(); err != nil {
		// Fall back to stale cache.
		if st, err2 := os.Stat(cached); err2 == nil && !st.IsDir() {
			return cached, true
		}
		return "", false
	}
	if err := copyFile(tmp.Name(), cached); err != nil {
		return "", false
	}
	_ = os.Chmod(cached, 0o755)
	return cached, true
}

// resolveDockerSocket finds the host docker.sock path and the GID
// it's owned by. The gate's `spwn` user needs that GID added via
// --group-add to actually read the bind-mounted socket.
//
// macOS rabbit hole: /var/run/docker.sock is typically a symlink to
// the actual socket which lives under the user's ~/.docker (Docker
// Desktop), ~/.orbstack (OrbStack), or ~/.colima (Colima). We try
// DOCKER_HOST first (authoritative), then the well-known symlinks,
// then the runtime-specific direct paths.
//
// Returns the path to bind-mount + the GID as a string. ok=false if
// no socket found or GID couldn't be resolved.
func resolveDockerSocket() (string, string, bool) {
	candidates := []string{}
	if h := os.Getenv("DOCKER_HOST"); strings.HasPrefix(h, "unix://") {
		candidates = append(candidates, strings.TrimPrefix(h, "unix://"))
	}
	candidates = append(candidates, "/var/run/docker.sock")
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates,
			filepath.Join(home, ".orbstack", "run", "docker.sock"),
			filepath.Join(home, ".docker", "run", "docker.sock"),
			filepath.Join(home, ".colima", "default", "docker.sock"),
		)
	}
	for _, p := range candidates {
		st, err := os.Stat(p)
		if err != nil || st.IsDir() {
			continue
		}
		sys := st.Sys()
		if sys == nil {
			continue
		}
		if stat, ok := sys.(*syscall.Stat_t); ok {
			return p, strconv.FormatUint(uint64(stat.Gid), 10), true
		}
	}
	return "", "", false
}

func isRunning(ctx context.Context) (bool, error) {
	// `docker container inspect` (vs the bare `docker inspect`) is
	// scoped to containers only — without `container`, docker matches
	// images by the same name first, which yields a false positive
	// because the image and container share the `spwn-gate` name.
	out, err := exec.CommandContext(ctx, "docker", "container", "inspect", "-f", "{{.State.Running}}", ContainerName).Output()
	if err != nil {
		return false, nil
	}
	return strings.TrimSpace(string(out)) == "true", nil
}

func containerExists(ctx context.Context) (bool, error) {
	err := exec.CommandContext(ctx, "docker", "container", "inspect", ContainerName).Run()
	if err == nil {
		return true, nil
	}
	if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
		return false, nil
	}
	return false, err
}

func imageExists(ctx context.Context) (bool, error) {
	err := exec.CommandContext(ctx, "docker", "image", "inspect", ImageName).Run()
	if err == nil {
		return true, nil
	}
	if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
		return false, nil
	}
	return false, err
}

func buildImage(ctx context.Context, w io.Writer) error {
	root, err := findSourceRoot()
	if err != nil {
		return err
	}

	// Cross-compile the gate binary for linux/<host-arch> (Docker
	// Desktop runs containers matching the host arch on macOS).
	// Building inside Docker fails because go.work references every
	// module in the workspace, which a docker build context can't
	// see piecewise — the architect image solves this the same way.
	tmpBin, err := os.CreateTemp("", "spwn-gate-linux-*")
	if err != nil {
		return fmt.Errorf("create temp binary: %w", err)
	}
	tmpBin.Close()
	defer os.Remove(tmpBin.Name())

	build := exec.CommandContext(ctx, "go", "build",
		"-trimpath", "-ldflags=-s -w",
		"-o", tmpBin.Name(),
		"./apps/gate/cmd/spwn-gate",
	)
	build.Dir = root
	build.Env = append(os.Environ(),
		"GOOS=linux",
		"GOARCH="+runtime.GOARCH,
		"CGO_ENABLED=0",
	)
	build.Stdout = w
	build.Stderr = w
	if err := build.Run(); err != nil {
		return fmt.Errorf("cross-compile spwn-gate: %w", err)
	}

	// Stage the binary into a temp build context next to a copy of
	// the Dockerfile so docker build sees just `./spwn-gate` + Dockerfile.
	stage, err := os.MkdirTemp("", "spwn-gate-build-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stage)

	if err := copyFile(tmpBin.Name(), filepath.Join(stage, "spwn-gate")); err != nil {
		return fmt.Errorf("stage binary: %w", err)
	}
	if err := copyFile(filepath.Join(root, "apps", "gate", "Dockerfile"), filepath.Join(stage, "Dockerfile")); err != nil {
		return fmt.Errorf("stage Dockerfile: %w", err)
	}
	// gate-browser is the Playwright sidecar — a Node HTTP service
	// that catalog tools call to drive a cookie-loaded Chromium.
	// Stage the whole dir.
	if err := copyDir(filepath.Join(root, "apps", "gate", "browser"), filepath.Join(stage, "browser")); err != nil {
		return fmt.Errorf("stage browser sidecar: %w", err)
	}
	// @spwn/gate-tool SDK — required by every catalog tool. Baked
	// into /usr/lib/node_modules/@spwn/gate-tool/ inside the image.
	if err := copyDir(filepath.Join(root, "apps", "gate", "sdk"), filepath.Join(stage, "sdk")); err != nil {
		return fmt.Errorf("stage @spwn/gate-tool SDK: %w", err)
	}

	cmd := exec.CommandContext(ctx, "docker", "build",
		"--platform", "linux/"+runtime.GOARCH,
		"-t", ImageName,
		stage,
	)
	cmd.Stdout = w
	cmd.Stderr = w
	return cmd.Run()
}

func copyFile(src, dst string) error {
	in, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, in, 0o755)
}

// copyDir recursively copies src to dst (file mode 0644, dirs 0755).
// Used by the gate build to stage the Node sidecar tree.
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}

func dockerCmd(ctx context.Context, args ...string) error {
	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker %s: %w (%s)", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

// findSourceRoot locates the spwn workspace (contains go.work and
// apps/gate/Dockerfile). Mirrors the equivalent in
// packages/architect/build.go — kept inline because architect is in
// a sibling layer, importing it would create a cross-module dep.
func findSourceRoot() (string, error) {
	if bi, ok := debug.ReadBuildInfo(); ok {
		for _, s := range bi.Settings {
			if s.Key == "vcs.directory" && s.Value != "" && isSpwnRoot(s.Value) {
				return s.Value, nil
			}
		}
	}
	if exe, err := os.Executable(); err == nil {
		if r := findRootUpward(filepath.Dir(exe)); r != "" {
			return r, nil
		}
	}
	if cwd, err := os.Getwd(); err == nil {
		if r := findRootUpward(cwd); r != "" {
			return r, nil
		}
	}
	return "", fmt.Errorf("cannot find spwn source tree (looked from cwd, exe dir, vcs.directory)")
}

func findRootUpward(dir string) string {
	for {
		if isSpwnRoot(dir) {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func isSpwnRoot(dir string) bool {
	_, err1 := os.Stat(filepath.Join(dir, "go.work"))
	_, err2 := os.Stat(filepath.Join(dir, "apps", "gate", "Dockerfile"))
	return err1 == nil && err2 == nil
}

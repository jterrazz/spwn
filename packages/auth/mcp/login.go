package mcp

import (
	"context"
	_ "embed"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
)

// HelperImage is the tag of the one-shot OAuth helper container.
// Built lazily on first Login from the embedded dockerfile below.
const HelperImage = "spwn-mcp-auth:latest"

//go:embed dockerfile.mcp-auth
var helperDockerfile []byte

// Login runs the OAuth dance for p in a helper container, persisting
// the resulting tokens to the host cache. Output (the "open this
// URL" hint mcp2cli prints) is streamed to w so the user sees it
// live. Blocks until either (a) the user clicks Allow and the
// callback completes, or (b) the helper exits with an error.
//
// The flow:
//  1. Ensure ~/.spwn/credentials/mcp exists.
//  2. Ensure spwn-mcp-auth:latest exists, building it if not.
//  3. Pick a free TCP port for the OAuth callback so concurrent logins
//     don't collide.
//  4. docker run --rm -p 127.0.0.1:<port>:<port> -v <cache>:/root/.cache/mcp2cli
//     spwn-mcp-auth:latest --mcp <p.URL> --oauth
//     --oauth-redirect-uri http://127.0.0.1:<port>/callback --list
//
// --list is the cheapest mcp2cli call that triggers the OAuth init —
// it returns once the server has handed back a tools list, which
// only happens after the token exchange.
func Login(ctx context.Context, p Provider, w io.Writer) error {
	if w == nil {
		w = io.Discard
	}

	cache := CacheDir()
	if err := os.MkdirAll(cache, 0o700); err != nil {
		return fmt.Errorf("create cache dir: %w", err)
	}

	if err := ensureHelperImage(ctx, w); err != nil {
		return err
	}

	port, err := freeTCPPort()
	if err != nil {
		return fmt.Errorf("pick callback port: %w", err)
	}

	portStr := strconv.Itoa(port)
	redirect := "http://127.0.0.1:" + portStr + "/callback"
	publish := "127.0.0.1:" + portStr + ":" + portStr
	mount := cache + ":/root/.cache/mcp2cli"

	args := []string{
		"run", "--rm",
		"-p", publish,
		"-v", mount,
		HelperImage,
		"--mcp", p.URL,
		"--oauth",
		"--oauth-redirect-uri", redirect,
		"--list",
	}
	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Stdout = w
	cmd.Stderr = w
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("mcp oauth login failed: %w", err)
	}
	if !IsAuthenticated(p) {
		// mcp2cli exited 0 but no tokens.json landed — likely a bind-
		// mount or path mismatch. Surface this loudly rather than
		// silently claiming success.
		return fmt.Errorf("login completed but no tokens persisted at %s", ProviderTokenPath(p))
	}
	return nil
}

// ensureHelperImage builds spwn-mcp-auth:latest if Docker doesn't
// already have it. Streams build output to w so the user sees what's
// happening during the (one-time, slow-ish) initial pull of
// python:3.12-slim.
func ensureHelperImage(ctx context.Context, w io.Writer) error {
	check := exec.CommandContext(ctx, "docker", "image", "inspect", HelperImage)
	check.Stdout = io.Discard
	check.Stderr = io.Discard
	if err := check.Run(); err == nil {
		return nil
	}
	fmt.Fprintf(w, "Building %s (one-time setup)...\n", HelperImage)

	tmp, err := os.MkdirTemp("", "spwn-mcp-auth-")
	if err != nil {
		return fmt.Errorf("temp dir: %w", err)
	}
	defer os.RemoveAll(tmp)
	if err := os.WriteFile(filepath.Join(tmp, "Dockerfile"), helperDockerfile, 0o644); err != nil {
		return fmt.Errorf("write Dockerfile: %w", err)
	}

	build := exec.CommandContext(ctx, "docker", "build", "-t", HelperImage, tmp)
	build.Stdout = w
	build.Stderr = w
	if err := build.Run(); err != nil {
		return fmt.Errorf("build helper image: %w", err)
	}
	return nil
}

// freeTCPPort asks the kernel for an unused TCP port on loopback by
// binding :0 and reading back the assigned port. Race window between
// close + docker bind is negligible in practice (worst case the user
// retries), and avoids a hard-coded port that conflicts with the
// user's other local services.
func freeTCPPort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

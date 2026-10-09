package gh

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// ErrGhNotInstalled is surfaced when neither host gh CLI is found
// nor (future) an in-container gh helper. Callers should print the
// installation hint rather than re-wrap this.
var ErrGhNotInstalled = errors.New("gh CLI not installed on host")

// ErrHostNotLoggedIn means gh exists on the host but `gh auth
// token` failed — user hasn't run `gh auth login` yet.
var ErrHostNotLoggedIn = errors.New("host gh CLI is not logged in")

// Login imports an existing host gh authentication into the spwn
// cache. Concretely: extract the token via `gh auth token`,
// resolve the username via `gh api user --jq .login`, and write a
// plaintext hosts.yml under CacheDir so the token survives without
// the keychain (which the container can't reach).
//
// Output (status messages, errors) is streamed to w.
//
// Future: if `gh` isn't on the host, fall back to running
// `gh auth login --web` inside a helper container. For v1 we
// require host gh — most users have it; we emit a clear hint
// otherwise.
func Login(ctx context.Context, w io.Writer) error {
	if w == nil {
		w = io.Discard
	}

	ghBin, err := exec.LookPath("gh")
	if err != nil {
		return ErrGhNotInstalled
	}

	token, err := readHostToken(ctx, ghBin)
	if err != nil {
		return err
	}
	user, err := readHostUser(ctx, ghBin)
	if err != nil {
		// Username failure isn't fatal — gh works without it. Log
		// And continue with an empty user line.
		fmt.Fprintf(w, "warning: could not resolve username (%v); continuing\n", err)
		user = ""
	}

	if err := os.MkdirAll(CacheDir(), 0o700); err != nil {
		return fmt.Errorf("create cache dir: %w", err)
	}
	if err := writeHostsYAML(token, user); err != nil {
		return fmt.Errorf("write hosts.yml: %w", err)
	}
	fmt.Fprintf(w, "Imported gh credentials for %s\n", user)
	return nil
}

func readHostToken(ctx context.Context, ghBin string) (string, error) {
	cmd := exec.CommandContext(ctx, ghBin, "auth", "token")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrHostNotLoggedIn, err)
	}
	tok := strings.TrimSpace(string(out))
	if tok == "" {
		return "", ErrHostNotLoggedIn
	}
	return tok, nil
}

func readHostUser(ctx context.Context, ghBin string) (string, error) {
	cmd := exec.CommandContext(ctx, ghBin, "api", "user", "--jq", ".login")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// writeHostsYAML emits the plaintext-token form of gh's hosts.yml.
// The shape matches what gh writes when it can't reach a keyring:
//
//	github.com:
//	    oauth_token: <token>
//	    user: <login>
//	    git_protocol: https
//
// We don't try to preserve the host's git_protocol (often ssh) —
// inside a container we never use SSH for git over the GitHub MCP,
// and https avoids needing the host's SSH key.
func writeHostsYAML(token, user string) error {
	var b strings.Builder
	b.WriteString("github.com:\n")
	b.WriteString("    oauth_token: " + token + "\n")
	if user != "" {
		b.WriteString("    user: " + user + "\n")
	}
	b.WriteString("    git_protocol: https\n")
	return os.WriteFile(HostsPath(), []byte(b.String()), 0o600)
}

package auth

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"spwn.sh/packages/auth/mcp"
	"spwn.sh/packages/platform"
)

// SyncCredentials resolves all credentials and writes them to the credentials
// directory (~/.spwn/credentials/). This directory is bind-mounted into every
// container at /credentials/, so updates are visible immediately.
//
// Safe to call repeatedly - uses atomic writes.
func SyncCredentials() error {
	dir := platform.CredentialsDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create credentials dir: %w", err)
	}

	// Resolve all credentials from env, keychain, cached files
	creds := ResolveAll()

	// Skip disabled providers (user clicked "Reset" in settings)
	for p, cred := range creds {
		if IsProviderDisabled(p) {
			cred.Type = CredTypeNone
			cred.Token = ""
		}
	}

	// Write .env file (atomic)
	if err := writeEnvFile(dir, creds); err != nil {
		return fmt.Errorf("write .env: %w", err)
	}

	// Sync runtime-specific files (e.g., codex auth.json)
	if err := syncRuntimeFiles(dir); err != nil {
		return fmt.Errorf("sync runtime files: %w", err)
	}

	// Refresh expiring MCP OAuth tokens (Notion etc) so containers
	// spawned with the cred bind mount get fresh tokens. Per-provider
	// failures are logged but never block sync — a broken refresh
	// shouldn't take down `spwn up` or `spwn agent talk`.
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	if _, errs := mcp.RefreshAll(ctx, mcp.DefaultRefreshLeeway); len(errs) > 0 {
		for _, err := range errs {
			log.Printf("warning: mcp token refresh: %v", err)
		}
	}

	// Write sync timestamp
	_ = os.WriteFile(filepath.Join(dir, ".last-sync"), []byte(time.Now().Format(time.RFC3339)), 0600)

	return nil
}

// writeEnvFile writes all resolved credentials as KEY=VALUE lines.
// Uses atomic write: .env.tmp → rename → .env.
//
// Empty-result protection: if the resolver returned no usable
// credentials (every entry is CredTypeNone / empty Token) AND the
// existing .env on disk already has content, we LEAVE the file
// alone and log a warning. Rationale: the resolver can transiently
// return empty when keychain access fails in a non-interactive
// process context (launchd-spawned daemons, `make`-spawned
// subprocesses, gate-side spwn invocations inside the container
// where keychain isn't reachable at all). Overwriting good creds
// with the empty result corrupts every subsequent dispatch
// (containers' prelaunch script copies the now-empty
// /credentials/anthropic/.credentials.json over the in-container
// one, breaking auth for every worker that next runs).
//
// We're optimising for "don't break working creds"; the cost is
// that a USER-DRIVEN auth removal (logout) won't propagate until
// they explicitly clear the creds dir. That's an explicit user
// action anyway and `spwn auth logout` already handles it directly.
func writeEnvFile(dir string, creds map[Provider]*Credential) error {
	var lines []string

	for _, cred := range creds {
		if cred.Type == CredTypeNone || cred.Token == "" {
			continue
		}
		// Write the primary env var
		lines = append(lines, fmt.Sprintf("export %s=%q", cred.EnvVar, cred.Token))
	}

	envPath := filepath.Join(dir, ".env")

	// Empty-result protection — see function doc.
	if len(lines) == 0 {
		if st, err := os.Stat(envPath); err == nil && st.Size() > 0 {
			log.Printf("warning: SyncCredentials resolved no credentials but %s already has content; leaving it alone (likely a transient keychain probe failure)", envPath)
			return nil
		}
		// No existing content either — write the empty file so
		// callers see "synced, just no creds" rather than "missing".
	}

	// Sort for deterministic output
	sort.Strings(lines)

	content := strings.Join(lines, "\n")
	if len(lines) > 0 {
		content += "\n"
	}

	// Atomic write
	tmpPath := filepath.Join(dir, ".env.tmp")

	if err := os.WriteFile(tmpPath, []byte(content), 0600); err != nil {
		return err
	}
	return os.Rename(tmpPath, envPath)
}

// syncRuntimeFiles copies runtime-specific credential files into the
// credentials directory. Handled today:
//   - Codex:  ~/.codex/auth.json          → credentials/openai/auth.json
//   - Claude: ~/.claude/.credentials.json → credentials/anthropic/.credentials.json
//
// These mirror what the user produces by running `codex login` /
// `claude login` on the host. The bind into world containers is the
// per-tier-zero auth bind (see spawn.go), so once written here the
// next spawn picks them up without an extra step.
func syncRuntimeFiles(dir string) error {
	home, _ := os.UserHomeDir()

	// Codex auth.json
	codexAuth := filepath.Join(home, ".codex", "auth.json")
	if _, err := os.Stat(codexAuth); err == nil {
		destDir := filepath.Join(dir, "openai")
		if err := os.MkdirAll(destDir, 0700); err != nil {
			return err
		}
		if err := copyFile(codexAuth, filepath.Join(destDir, "auth.json")); err != nil {
			return fmt.Errorf("copy codex auth: %w", err)
		}
	}

	// Claude .credentials.json — present on Linux and as a macOS
	// fallback. detectAnthropic also reads from ~/.claude directly,
	// but the sync is what makes the credential available INSIDE
	// worker containers (which see /credentials, not the host home).
	claudeCreds := filepath.Join(home, ".claude", ".credentials.json")
	if _, err := os.Stat(claudeCreds); err == nil {
		destDir := filepath.Join(dir, "anthropic")
		if err := os.MkdirAll(destDir, 0700); err != nil {
			return err
		}
		if err := copyFile(claudeCreds, filepath.Join(destDir, ".credentials.json")); err != nil {
			return fmt.Errorf("copy claude credentials: %w", err)
		}
	}

	return nil
}

// copyFile copies src to dst with 0600 permissions.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

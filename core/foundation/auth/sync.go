package auth

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"spwn.sh/core/foundation"
)

// SyncCredentials resolves all credentials and writes them to the credentials
// directory (~/.spwn/credentials/). This directory is bind-mounted into every
// container at /credentials/, so updates are visible immediately.
//
// Safe to call repeatedly — uses atomic writes.
func SyncCredentials() error {
	dir := foundation.CredentialsDir()
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

	// Write sync timestamp
	_ = os.WriteFile(filepath.Join(dir, ".last-sync"), []byte(time.Now().Format(time.RFC3339)), 0600)

	return nil
}

// writeEnvFile writes all resolved credentials as KEY=VALUE lines.
// Uses atomic write: .env.tmp → rename → .env
func writeEnvFile(dir string, creds map[Provider]*Credential) error {
	var lines []string

	for _, cred := range creds {
		if cred.Type == CredTypeNone || cred.Token == "" {
			continue
		}
		// Write the primary env var
		lines = append(lines, fmt.Sprintf("export %s=%q", cred.EnvVar, cred.Token))
	}

	// Sort for deterministic output
	sort.Strings(lines)

	content := strings.Join(lines, "\n")
	if len(lines) > 0 {
		content += "\n"
	}

	// Atomic write
	tmpPath := filepath.Join(dir, ".env.tmp")
	envPath := filepath.Join(dir, ".env")

	if err := os.WriteFile(tmpPath, []byte(content), 0600); err != nil {
		return err
	}
	return os.Rename(tmpPath, envPath)
}

// syncRuntimeFiles copies runtime-specific credential files into the
// credentials directory. Currently handles:
// - Codex: ~/.codex/auth.json → credentials/openai/auth.json
func syncRuntimeFiles(dir string) error {
	// Codex auth.json
	home, _ := os.UserHomeDir()
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

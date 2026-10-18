package authentication

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

	"spwn.sh/packages/authentication/mcp"
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

	creds := ResolveAll()

	for p, cred := range creds {
		if IsProviderDisabled(p) {
			cred.Type = CredTypeNone
			cred.Token = ""
		}
	}

	if err := writeEnvFile(dir, creds); err != nil {
		return fmt.Errorf("write .env: %w", err)
	}

	if err := syncRuntimeFiles(dir); err != nil {
		return fmt.Errorf("sync runtime files: %w", err)
	}

	// invariant: Refresh expiring MCP OAuth tokens (Notion etc) so containers
	// spawned with the cred bind mount get fresh tokens. Per-provider
	// failures are logged but never block sync: a broken refresh
	// shouldn't take down `spwn up` or `spwn agent talk`.
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	if _, errs := mcp.RefreshAll(ctx, mcp.DefaultRefreshLeeway); len(errs) > 0 {
		for _, err := range errs {
			log.Printf("warning: mcp token refresh: %v", err)
		}
	}

	_ = os.WriteFile(filepath.Join(dir, ".last-sync"), []byte(time.Now().Format(time.RFC3339)), 0600)

	return nil
}

func writeEnvFile(dir string, creds map[Provider]*Credential) error {
	var lines []string

	for _, cred := range creds {
		if cred.Type == CredTypeNone || cred.Token == "" {
			continue
		}
		lines = append(lines, fmt.Sprintf("export %s=%q", cred.EnvVar, cred.Token))
	}

	envPath := filepath.Join(dir, ".env")

	if len(lines) == 0 {
		if st, err := os.Stat(envPath); err == nil && st.Size() > 0 {
			log.Printf("warning: SyncCredentials resolved no credentials but %s already has content; leaving it alone (likely a transient keychain probe failure)", envPath)
			return nil
		}
	}

	sort.Strings(lines)

	content := strings.Join(lines, "\n")
	if len(lines) > 0 {
		content += "\n"
	}

	tmpPath := filepath.Join(dir, ".env.tmp")

	if err := os.WriteFile(tmpPath, []byte(content), 0600); err != nil {
		return err
	}
	return os.Rename(tmpPath, envPath)
}

func syncRuntimeFiles(dir string) error {
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

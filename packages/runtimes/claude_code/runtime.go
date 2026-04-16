package claude_code

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strings"

	rt "spwn.sh/packages/world/runtime"
)

// Runtime is the claude-code spawn-time adapter — implements the
// world/runtime.Runtime interface (BuildCommand, credential sync,
// prelaunch shell, default config files). Distinct from Tool (the
// image.Tool for image builds) which lives in claude_code.go.
var Runtime = &runtimeAdapter{}

type runtimeAdapter struct{}

func init() { rt.Register(Runtime) }

// Name returns the runtime identifier.
func (c *runtimeAdapter) Name() string { return "claude-code" }

// BuildCommand constructs the claude CLI command with all flags.
func (c *runtimeAdapter) BuildCommand(cfg rt.SpawnConfig) []string {
	cmd := []string{"claude", "--dangerously-skip-permissions"}

	// NPC mode: no named agent, just print
	if cfg.AgentName == "" {
		if cfg.Prompt != "" {
			cmd = append(cmd, "-p", cfg.Prompt, "--print")
		}
		return cmd
	}

	// Worker/Manager/Chief: session management via --resume
	// If a SessionID is provided (from a previous response), resume that session.
	// Otherwise start a fresh session. The session_id is returned in the JSON
	// response and should be captured by the caller for subsequent messages.
	if cfg.SessionID != "" {
		cmd = append(cmd, "--resume", cfg.SessionID)
	}

	if cfg.Prompt != "" {
		cmd = append(cmd, "-p", cfg.Prompt)
	}

	return cmd
}

// SupportsSession returns true if the runtime can resume sessions.
func (c *runtimeAdapter) SupportsSession() bool { return true }
func (c *runtimeAdapter) Available() bool       { return true }

// ── Container-side setup ─────────────────────────────────────────

// DefaultConfigFiles pre-dismisses Claude Code's first-run UI.
// Without these, every invocation of `claude` inside a fresh agent
// home walks the user through onboarding, trust dialogs, and
// permission prompts - painful when the goal is "open an
// interactive session" and the user never sees a clean terminal.
//
// The files are written at spawn time into /agents/<name>/, which
// is the actual HOME the runtime runs under (not /home/spwn).
// Previous attempts baked these into the base image at build time
// and lost to the HOME override.
func (c *runtimeAdapter) DefaultConfigFiles(agentHome string) map[string][]byte {
	// Trust the agent's own home + the workspaces mount root so
	// Claude Code doesn't prompt on first access. We can't
	// enumerate the resolved workspace names here without plumbing
	// them through; trusting /workspaces (the root every bind mount
	// lands under) covers any child path the agent will cd into.
	claudeJSON := map[string]any{
		"hasCompletedOnboarding": true,
		"projects": map[string]any{
			agentHome: map[string]any{
				"hasTrustDialogAccepted": true,
			},
			"/workspaces": map[string]any{
				"hasTrustDialogAccepted": true,
			},
		},
	}
	claudeJSONBytes, _ := json.Marshal(claudeJSON)

	settingsJSON := map[string]any{
		"skipDangerousModePermissionPrompt": true,
	}
	settingsBytes, _ := json.Marshal(settingsJSON)

	return map[string][]byte{
		".claude.json":          claudeJSONBytes,
		".claude/settings.json": settingsBytes,
	}
}

// ── Host-side credential sync ────────────────────────────────────

// SyncHostCredentials copies the host's Claude Code OAuth token into
// credsDir so the containerised runtime can read it via the
// /credentials bind mount. Resolution order mirrors the opencode
// claude-auth plugin, which is the best-documented solution in the
// wild (https://github.com/griffinmartin/opencode-claude-auth):
//
//  1. ~/.claude/.credentials.json on the host (works on all
//     platforms; the only file Claude Code itself reads on Linux
//     and the file-based fallback on macOS when the Keychain is
//     inaccessible).
//  2. macOS Keychain service "Claude Code-credentials" via
//     `security find-generic-password -s ... -w` - the primary
//     store on macOS for OAuth subscription users.
//
// CLAUDE_CODE_OAUTH_TOKEN, ANTHROPIC_API_KEY, and ANTHROPIC_AUTH_TOKEN
// are already flowed through the env-var path by packages/auth, so
// this method only handles the file-based sources that path cannot
// reach.
//
// A missing credential source is not an error: the env-var path may
// still supply working auth. Return an error only for real I/O or
// command failures.
func (c *runtimeAdapter) SyncHostCredentials(credsDir string) error {
	dstDir := filepath.Join(credsDir, "anthropic")
	dst := filepath.Join(dstDir, ".credentials.json")

	// Source 1: host credentials file.
	if b, ok := readHostCredentialsFile(); ok {
		return writeCredsFile(dstDir, dst, b)
	}

	// Source 2: macOS Keychain (silent no-op on other platforms).
	if goruntime.GOOS == "darwin" {
		if b, ok := extractFromMacOSKeychain(); ok {
			return writeCredsFile(dstDir, dst, b)
		}
	}

	// No file/Keychain source available. Clear any stale file
	// from a previous sync so we never mislead the container into
	// thinking it has working creds.
	_ = os.Remove(dst)
	return nil
}

// PrelaunchShell returns the shell snippet that wraps every runtime
// exec. It sources env vars from /credentials/.env (set by
// packages/auth.SyncCredentials), then copies the credential file
// Claude Code expects into the agent's HOME.
//
// Runs as the agent user with /credentials bind-mounted read-only.
// Every line guards with test-before-act so missing sources never
// break the launch - the container may still have working auth via
// env vars sourced from /credentials/.env alone.
func (c *runtimeAdapter) PrelaunchShell() string {
	return strings.Join([]string{
		"source /credentials/.env 2>/dev/null",
		// Claude subscription OAuth: point ~/.claude/.credentials.json
		// at the host-synced file. Use a copy rather than a symlink
		// so Claude Code's in-place token refresh doesn't mutate the
		// bind-mounted source.
		`if [ -f /credentials/anthropic/.credentials.json ]; then ` +
			`mkdir -p "$HOME/.claude" && ` +
			`cp /credentials/anthropic/.credentials.json "$HOME/.claude/.credentials.json" && ` +
			`chmod 600 "$HOME/.claude/.credentials.json"; fi`,
	}, "; ")
}

// ── internal helpers ─────────────────────────────────────────────

// readHostCredentialsFile returns the content of
// ~/.claude/.credentials.json when it exists and is non-empty.
func readHostCredentialsFile() ([]byte, bool) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, false
	}
	path := filepath.Join(home, ".claude", ".credentials.json")
	b, err := os.ReadFile(path)
	if err != nil || len(b) == 0 {
		return nil, false
	}
	return b, true
}

// extractFromMacOSKeychain pulls the Claude Code-credentials entry
// out of the macOS Keychain via the `security` CLI. Returns ok=false
// on any failure (missing entry, locked Keychain, non-macOS).
func extractFromMacOSKeychain() ([]byte, bool) {
	cmd := exec.Command("security", "find-generic-password",
		"-s", "Claude Code-credentials", "-w")
	out, err := cmd.Output()
	if err != nil {
		return nil, false
	}
	out = []byte(strings.TrimSpace(string(out)))
	if len(out) == 0 {
		return nil, false
	}
	return out, true
}

// writeCredsFile atomically writes b to dst with mode 0600 (Claude
// Code requires tight perms on .credentials.json). Creates dir if
// missing.
func writeCredsFile(dir, dst string, b []byte) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	tmp := dst + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return fmt.Errorf("write %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, dst); err != nil {
		return fmt.Errorf("rename %s: %w", dst, err)
	}
	return nil
}

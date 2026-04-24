package codex

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"spwn.sh/packages/runtimes"
)

// Spawner is the codex spawn-time adapter. Implements the full
// `runtimes.Spawner` interface so `spwn agent talk` on a codex-backed
// world routes through the same dispatch as claude-code.
//
// The OpenAI Codex CLI (`@openai/codex` npm) has a different shape
// from Claude Code: interactive is the bare `codex` command, non-
// interactive is `codex exec [--json] [--thread <id>] "<prompt>"`.
// One-shot sessions return a streaming JSONL transcript of
// thread.*/item.* events culminating in a final `turn.completed`
// envelope that carries the assistant's text.
var Spawner = &spawner{}

type spawner struct{}

// Name returns the runtime identifier.
func (*spawner) Name() string { return "codex" }

// BuildCommand constructs the codex CLI argv. The exec subcommand
// runs a single prompt and exits, which is what `spwn agent talk`
// with a message expects. When no prompt is provided (interactive
// session), the argv is just `codex` — the caller supplies a TTY.
//
// SessionID threads resumption through codex's `--thread <id>` flag
// (codex calls its session identifier "thread"; extractSessionID in
// the CLI already bridges the terminology so callers don't need to
// know the runtime's internal vocabulary).
func (*spawner) BuildCommand(cfg runtimes.SpawnConfig) []string {
	// `--dangerously-bypass-approvals-and-sandbox` is accepted by
	// Both the top-level `codex` (interactive REPL) and the `codex
	// Exec` subcommand. Codex's default bwrap sandbox can't nest
	// Inside our worker container ("No permissions to create a new
	// Namespace"); the container IS the sandbox, so nested sandboxing
	// Just blocks tool use without adding safety.
	//
	// `--skip-git-repo-check` is ONLY accepted by the `exec`
	// Subcommand. In interactive mode we rely on the trust-seed
	// Written by PrelaunchShell (~/.codex/config.toml with
	// `[projects."$HOME"] trust_level = "trusted"`) plus a `git init`
	// Of /agents/<name> in that same prelaunch — both together
	// Satisfy codex's "trusted directory" check without needing the
	// Exec-only flag.
	const sandboxBypass = "--dangerously-bypass-approvals-and-sandbox"

	// Interactive mode: no prompt. Covers both the anonymous REPL
	// (no AgentName) and the architect's detached "just start the
	// Agent in its container" spawn (AgentName set, no prompt) — both
	// Want the same bare `codex` REPL with ONLY the flags the
	// Top-level command accepts. Passing `--skip-git-repo-check`
	// Here used to produce "error: unexpected argument
	// '--skip-git-repo-check'" from codex ≥ 0.122.
	if cfg.Prompt == "" {
		return []string{"codex", sandboxBypass}
	}

	// Non-interactive exec path: a prompt was supplied. The exec
	// Subcommand DOES accept --skip-git-repo-check, keep it there as
	// Belt-and-suspenders alongside the trust seed. Session resume
	// Uses the `resume <id>` subcommand rather than a flag — codex
	// ≥ 0.122 dropped `--thread` in favour of the positional
	// Subcommand form. `--json` is baked in here (not in
	// OneShotFlags) because codex's exec subcommand parses PROMPT as
	// The first non-flag positional; any flag appended AFTER the
	// Prompt is swallowed or errors out.
	cmd := []string{"codex", "exec"}
	if cfg.SessionID != "" {
		cmd = append(cmd, "resume")
	}
	cmd = append(cmd, "--skip-git-repo-check", sandboxBypass, "--json")
	if cfg.SessionID != "" {
		cmd = append(cmd, cfg.SessionID)
	}
	cmd = append(cmd, cfg.Prompt)
	return cmd
}

// SupportsSession reports that codex's `--thread <id>` resume path
// is wired, so `spwn agent talk` can pass SessionID across turns
// and conversations survive container restarts.
func (*spawner) SupportsSession() bool { return true }

// Available gates the runtime behind feature-complete checks. Codex
// ships today as an install target + non-interactive runtime.
func (*spawner) Available() bool { return true }

// DefaultConfigFiles returns the files codex wants materialised into
// the agent's HOME at spawn time. The per-agent .codex/config.toml
// (profile + hooks feature flag) is already emitted by the transpile
// renderer (see GenerateAgentConfigTOML) and the trust entry is
// seeded by PrelaunchShell, so nothing else needs seeding here.
func (*spawner) DefaultConfigFiles(agentHome string) map[string][]byte { return nil }

// SyncHostCredentials is a no-op: codex's OAuth file lives at
// ~/.codex/auth.json on the host and is already picked up by
// packages/auth's provider resolver, which writes it to
// /credentials/openai/auth.json. No runtime-specific host sync is
// needed beyond that.
func (*spawner) SyncHostCredentials(credsDir string) error { return nil }

// PrelaunchShell returns the container-side shell fragment that
// wires /credentials/openai/auth.json into the location codex looks
// up on startup (~/.codex/auth.json). Runs as the agent user with
// /credentials bind-mounted read-only; guards with test-before-act so
// the launch never fails when OpenAI creds aren't configured.
//
// Intentionally omits `source /credentials/.env` — that belongs to the
// outer prelaunch composition, not this adapter. Callers that need
// env sourcing chain it themselves.
func (*spawner) PrelaunchShell() string {
	// Three concerns, chained so any one can quietly no-op:
	//
	//   1. OAuth symlink — bring /credentials/openai/auth.json into
	//      $HOME/.codex/ so codex reads the shared credential.
	//   2. Project trust seed — codex ignores `<cwd>/.codex/config.toml`
	//      unless the directory is listed in `~/.codex/config.toml`
	//      under `[projects."<cwd>"] trust_level = "trusted"`. Append
	//      that block (idempotently via a grep guard) so the renderer-
	//      emitted project config actually takes effect.
	//   3. The `$HOME` env var at this point is the agent's home
	//      (/agents/<name>) — the exact directory codex's cwd points at
	//      and the exact key the trust table needs.
	return `mkdir -p $HOME/.codex; ` +
		`[ -f /credentials/openai/auth.json ] && ` +
		`ln -sf /credentials/openai/auth.json $HOME/.codex/auth.json 2>/dev/null; ` +
		`if ! grep -q "projects.\"$HOME\"" $HOME/.codex/config.toml 2>/dev/null; then ` +
		`printf '\n[projects."%s"]\ntrust_level = "trusted"\n' "$HOME" >> $HOME/.codex/config.toml; ` +
		`fi; ` +
		// Codex's "trusted directory" check in interactive mode
		// Additionally requires the cwd to sit inside a git repo.
		// The exec subcommand has a --skip-git-repo-check flag; the
		// Interactive REPL does not. An empty `git init` here makes
		// /agents/<name> a valid git repo without changing any
		// Workspace or user files — 100% idempotent, silent when a
		// .git already exists. Paired with the trust_level=trusted
		// Entry above, codex interactive starts clean.
		`git init -q "$HOME" 2>/dev/null || true`
}

// OneShotFlags is a no-op for codex. The `--json` output flag MUST
// sit before the positional PROMPT argument, which means it belongs
// in BuildCommand (where the prompt lives) rather than appended after
// the fact. Kept on the interface so other runtimes can still layer
// mode-specific flags post-hoc.
//
// `stream-json` and the default both map to `--json` — codex has no
// separate "final envelope" format, the stream IS the contract. The
// talk path either scans the JSONL in-flight (stream mode) or joins
// everything and asks ParseOneShotResult to pull out the final text.
func (*spawner) OneShotFlags(base []string, outputFormat string) []string {
	return base
}

// ParseOneShotResult walks codex's JSONL exec output from last-line
// back and returns the assistant's final text + thread identifier.
//
// Codex exec emits a sequence of events along the lines of:
//   {"type":"thread.started","thread_id":"th_abc"}
//   {"type":"turn.started",...}
//   {"type":"item.completed","item":{"item_type":"agent_message","text":"hello"}}
//   {"type":"turn.completed","usage":{...}}
//
// The last agent_message item is the assistant's final reply; the
// thread_id on thread.started persists across future invocations as
// the session id.
//
// Returns a non-nil error if no JSONL events could be parsed at all —
// callers fall back to printing raw output and scanning for session
// id via extractSessionID as a safety net.
func (*spawner) ParseOneShotResult(raw []byte) (string, string, error) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return "", "", fmt.Errorf("parse codex output: empty")
	}

	var text, threadID string
	var sawAny bool
	for _, line := range bytes.Split(raw, []byte("\n")) {
		trimmed := bytes.TrimSpace(line)
		if len(trimmed) == 0 || trimmed[0] != '{' {
			continue
		}
		var event struct {
			Type     string          `json:"type"`
			ThreadID string          `json:"thread_id"`
			Item     json.RawMessage `json:"item"`
		}
		if err := json.Unmarshal(trimmed, &event); err != nil {
			continue
		}
		sawAny = true
		if event.ThreadID != "" && threadID == "" {
			threadID = event.ThreadID
		}
		if event.Type == "item.completed" && len(event.Item) > 0 {
			// Codex ≥0.122 renamed the item discriminator from
			// `item_type` to `type`. Read both so the parser works
			// across the transition; the newer key wins when both
			// are present on a single envelope.
			var item struct {
				Type     string `json:"type"`
				ItemType string `json:"item_type"`
				Text     string `json:"text"`
			}
			if err := json.Unmarshal(event.Item, &item); err == nil {
				kind := item.Type
				if kind == "" {
					kind = item.ItemType
				}
				if kind == "agent_message" && strings.TrimSpace(item.Text) != "" {
					// Take the LAST agent_message — multi-turn exec
					// can have several; the last is the final reply.
					text = item.Text
				}
			}
		}
	}
	if !sawAny {
		return "", "", fmt.Errorf("parse codex output: no JSON events found")
	}
	return text, threadID, nil
}

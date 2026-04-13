# Codex

Codex is OpenAI's agent runtime — a CLI that executes tasks using GPT models.

## Usage
```bash
codex "your task here"                      # Interactive mode
codex exec "your task here"                 # Non-interactive mode
codex exec "task" --full-auto               # Full auto with sandboxed writes
codex exec "task" --model gpt-5.4           # Specify model
codex exec resume --last                    # Resume last session
```

## Configuration
Codex config lives at `~/.codex/config.toml` inside the container.
Auth tokens are forwarded from the host automatically.

## Sandbox Modes
- `read-only` — can read files, cannot write
- `workspace-write` — can write to workspace directory
- `danger-full-access` — no restrictions (used inside spwn worlds)

## Environment
- Auth is handled via OAuth tokens (subscription-based, e.g. ChatGPT Plus)
- Tokens are mounted from the host at `~/.codex/auth.json`

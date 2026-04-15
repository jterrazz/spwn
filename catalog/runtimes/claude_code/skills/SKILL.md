# Claude Code

Claude Code is your AI agent runtime - the thinking engine that processes tasks.

## Configuration
Claude Code is pre-configured with:
- Onboarding completed
- Workspace trust granted for /workspaces and the agent home
- Dangerous mode permissions skipped

## Usage
```bash
claude "your task here"              # Run a task
claude --continue                    # Continue last session
claude --session-id <id> "task"      # Resume specific session
```

## Environment
- `ANTHROPIC_API_KEY` - API key for Claude (if using API auth)
- `CLAUDE_CODE_OAUTH_TOKEN` - OAuth token (if using subscription auth)

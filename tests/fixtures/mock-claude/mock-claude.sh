#!/bin/bash
# Mock Claude Code CLI for E2E testing
# Records what it observed, writes to workspace, and exits

OUTPUT="/tmp/claude-mock.json"
EXIT_CODE=0
SLEEP=0
SESSION_ID=""
RESUME="false"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --exit-code) EXIT_CODE="$2"; shift 2 ;;
    --sleep) SLEEP="$2"; shift 2 ;;
    --session-id) SESSION_ID="$2"; shift 2 ;;
    --resume) RESUME="true"; shift ;;
    *) shift ;;
  esac
done

# A mounted /agents dir containing at least one agent home means the mind is visible.
MIND_EXISTS=false
if [ -d /agents ] && [ -n "$(ls -A /agents 2>/dev/null)" ]; then
  MIND_EXISTS=true
fi

# /workspaces is the workspace root when any workspace is mounted.
# The container may use /workspaces/default (unnamed) or /workspaces/<name>.
WORKSPACE_EXISTS=false
if [ -d /workspaces ] && [ -n "$(ls -A /workspaces 2>/dev/null)" ]; then
  WORKSPACE_EXISTS=true
fi

# A CLAUDE.md file under /agents/<name>/ proves the renderer ran and
# inlined the world context. Under the new claude-code renderer
# there's no separate /world/physics.md or /world/faculties.md.
CLAUDE_EXISTS=false
FIRST_CLAUDE=""
if [ -d /agents ]; then
  for agent_dir in /agents/*/; do
    if [ -f "${agent_dir}CLAUDE.md" ]; then
      CLAUDE_EXISTS=true
      FIRST_CLAUDE="${agent_dir}CLAUDE.md"
      break
    fi
  done
fi

cat > "$OUTPUT" <<RECORD
{
  "mind_exists": $MIND_EXISTS,
  "claude_md_exists": $CLAUDE_EXISTS,
  "workspace_exists": $WORKSPACE_EXISTS,
  "claude_md_content": $(cat "$FIRST_CLAUDE" 2>/dev/null | head -40 | python3 -c 'import sys,json; print(json.dumps(sys.stdin.read()))' 2>/dev/null || echo '""'),
  "session_id": "$SESSION_ID",
  "resume": $RESUME,
  "pid": $$,
  "exit_code": $EXIT_CODE
}
RECORD

# Write to the first workspace (if any) to prove the agent can DO something.
if [ -d /workspaces ]; then
  FIRST_WS=$(ls /workspaces 2>/dev/null | head -1)
  if [ -n "$FIRST_WS" ] && [ -d "/workspaces/$FIRST_WS" ]; then
    echo "mock-claude was here" > "/workspaces/$FIRST_WS/mock-output.txt" 2>/dev/null || true
  fi
fi

[ "$SLEEP" -gt 0 ] && sleep "$SLEEP"
exit "$EXIT_CODE"

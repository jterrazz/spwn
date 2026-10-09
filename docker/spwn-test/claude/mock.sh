#!/bin/bash

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

MIND_EXISTS=false
if [ -d /agents ] && [ -n "$(ls -A /agents 2>/dev/null)" ]; then
  MIND_EXISTS=true
fi

WORKSPACE_EXISTS=false
if [ -d /workspaces ] && [ -n "$(ls -A /workspaces 2>/dev/null)" ]; then
  WORKSPACE_EXISTS=true
fi

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

if [ -d /workspaces ]; then
  FIRST_WS=$(ls /workspaces 2>/dev/null | head -1)
  if [ -n "$FIRST_WS" ] && [ -d "/workspaces/$FIRST_WS" ]; then
    echo "mock-claude was here" > "/workspaces/$FIRST_WS/mock-output.txt" 2>/dev/null || true
  fi
fi

[ "$SLEEP" -gt 0 ] && sleep "$SLEEP"
exit "$EXIT_CODE"

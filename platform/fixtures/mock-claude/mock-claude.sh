#!/bin/bash
# Mock Claude Code CLI for E2E testing
# Records what it observed and exits

OUTPUT="/tmp/claude-mock.json"
EXIT_CODE=0
SLEEP=0
SESSION_ID=""
RESUME="false"

# Parse flags (accepts and ignores real Claude flags)
while [[ $# -gt 0 ]]; do
  case "$1" in
    --exit-code) EXIT_CODE="$2"; shift 2 ;;
    --sleep) SLEEP="$2"; shift 2 ;;
    --session-id) SESSION_ID="$2"; shift 2 ;;
    --resume) RESUME="true"; shift ;;
    *) shift ;;
  esac
done

# Record observations as JSON
cat > "$OUTPUT" <<RECORD
{
  "mind_exists": $([ -d /mind ] && echo true || echo false),
  "mind_personas": $([ -d /mind/personas ] && echo true || echo false),
  "physics_exists": $([ -f /universe/physics.md ] && echo true || echo false),
  "faculties_exists": $([ -f /universe/faculties.md ] && echo true || echo false),
  "workspace_exists": $([ -d /workspace ] && echo true || echo false),
  "physics_content": $(cat /universe/physics.md 2>/dev/null | head -20 | python3 -c 'import sys,json; print(json.dumps(sys.stdin.read()))' 2>/dev/null || echo '""'),
  "faculties_content": $(cat /universe/faculties.md 2>/dev/null | head -20 | python3 -c 'import sys,json; print(json.dumps(sys.stdin.read()))' 2>/dev/null || echo '""'),
  "session_id": "$SESSION_ID",
  "resume": $RESUME,
  "pid": $$,
  "exit_code": $EXIT_CODE
}
RECORD

[ "$SLEEP" -gt 0 ] && sleep "$SLEEP"
exit "$EXIT_CODE"

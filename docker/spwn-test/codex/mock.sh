#!/bin/bash

OUTPUT="/tmp/codex-mock.json"
EXIT_CODE=0
SLEEP=0
THREAD_ID=""
JSON_MODE="false"
PROMPT=""
SUBCOMMAND=""
RESUME="false"

if [[ $# -gt 0 && "$1" != --* ]]; then
  SUBCOMMAND="$1"
  shift
fi
if [[ "$SUBCOMMAND" == "exec" && "${1:-}" == "resume" ]]; then
  RESUME="true"
  shift
fi

while [[ $# -gt 0 ]]; do
  case "$1" in
    --exit-code) EXIT_CODE="$2"; shift 2 ;;
    --sleep) SLEEP="$2"; shift 2 ;;
    --thread) THREAD_ID="$2"; shift 2 ;;
    --json) JSON_MODE="true"; shift ;;
    --*) shift ;;
    *)
      if [ "$RESUME" = "true" ] && [ -z "$THREAD_ID" ]; then
        THREAD_ID="$1"
      elif [ -z "$PROMPT" ]; then
        PROMPT="$1"
      fi
      shift
      ;;
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

AGENTS_EXISTS=false
FIRST_AGENTS=""
if [ -d /agents ]; then
  for agent_dir in /agents/*/; do
    if [ -f "${agent_dir}AGENTS.md" ]; then
      AGENTS_EXISTS=true
      FIRST_AGENTS="${agent_dir}AGENTS.md"
      break
    fi
  done
fi

SKILLS_EXISTS=false
FIRST_SKILL=""
if [ -d /agents ]; then
  for skill_file in /agents/*/.agents/skills/*/SKILL.md; do
    if [ -f "$skill_file" ]; then
      SKILLS_EXISTS=true
      FIRST_SKILL="$skill_file"
      break
    fi
  done
fi

if [ -z "$THREAD_ID" ]; then
  THREAD_ID="th_mock_$$"
fi

cat > "$OUTPUT" <<RECORD
{
  "mind_exists": $MIND_EXISTS,
  "agents_md_exists": $AGENTS_EXISTS,
  "workspace_exists": $WORKSPACE_EXISTS,
  "agents_md_content": $(cat "$FIRST_AGENTS" 2>/dev/null | head -40 | python3 -c 'import sys,json; print(json.dumps(sys.stdin.read()))' 2>/dev/null || echo '""'),
  "skills_exists": $SKILLS_EXISTS,
  "skill_content": $(cat "$FIRST_SKILL" 2>/dev/null | head -40 | python3 -c 'import sys,json; print(json.dumps(sys.stdin.read()))' 2>/dev/null || echo '""'),
  "subcommand": "$SUBCOMMAND",
  "thread_id": "$THREAD_ID",
  "prompt": $(printf '%s' "$PROMPT" | python3 -c 'import sys,json; print(json.dumps(sys.stdin.read()))' 2>/dev/null || echo '""'),
  "json_mode": $JSON_MODE,
  "resume": $RESUME,
  "pid": $$,
  "exit_code": $EXIT_CODE
}
RECORD

if [ -d /workspaces ]; then
  FIRST_WS=$(ls /workspaces 2>/dev/null | head -1)
  if [ -n "$FIRST_WS" ] && [ -d "/workspaces/$FIRST_WS" ]; then
    echo "mock-codex was here" > "/workspaces/$FIRST_WS/mock-output.txt" 2>/dev/null || true
  fi
fi

REPLY="mock-codex reply"
if [ "$JSON_MODE" = "true" ]; then
  printf '{"type":"thread.started","thread_id":"%s"}\n' "$THREAD_ID"
  printf '{"type":"turn.started"}\n'
  printf '{"type":"item.completed","item":{"item_type":"agent_message","text":"%s"}}\n' "$REPLY"
  printf '{"type":"turn.completed","usage":{"input_tokens":1,"output_tokens":1}}\n'
else
  printf '%s\n' "$REPLY"
fi

[ "$SLEEP" -gt 0 ] && sleep "$SLEEP"
exit "$EXIT_CODE"

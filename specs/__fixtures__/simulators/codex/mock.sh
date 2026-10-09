#!/bin/bash
# Mock OpenAI Codex CLI for E2E testing
# Records what it observed, writes to workspace, and emits a plausible
# JSONL event stream that packages/runtimes/codex.ParseOneShotResult
# can decode. Parallel to claude/mock.sh.

OUTPUT="/tmp/codex-mock.json"
EXIT_CODE=0
SLEEP=0
THREAD_ID=""
JSON_MODE="false"
PROMPT=""
SUBCOMMAND=""
RESUME="false"

# codex CLI surface: `codex` (interactive) or `codex exec [--thread <id>] [--json] <prompt>`.
# We also accept the legacy --exit-code/--sleep escape hatches that
# tests use to drive error paths.
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
      # In codex >= 0.122, resume is `codex exec resume ... <thread> <prompt>`.
      # The first non-flag positional after `resume` is the thread id.
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

# Codex's entry file is AGENTS.md (vs claude's CLAUDE.md). Its presence
# proves the codex renderer ran and the world context was inlined.
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

# Synthesise a deterministic thread id when codex would have started a
# new thread, so tests can assert on the value ParseOneShotResult pulls
# back out.
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

# Emit a fake reply. In JSON mode this matches codex exec's event
# stream shape (thread.started → item.completed agent_message →
# turn.completed). Otherwise plain text.
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

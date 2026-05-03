#!/bin/bash
# spwn QA harness — 50 scenarios. Run: bash docs/qa/cli-scenarios/harness.sh [first] [last]
# Isolates state under $TMP_ROOT so the host's ~/.spwn is untouched.

set -o pipefail

SPWN=${SPWN:-/Users/jterrazz/Developer/spwn/spwn/bin/spwn}
TMP_ROOT=${TMP_ROOT:-/tmp/qa-50}
export SPWN_HOME="$TMP_ROOT/spwn_home"
export SPWN_BASE_IMAGE=${SPWN_BASE_IMAGE:-spwn-test:latest}

mkdir -p "$TMP_ROOT" "$SPWN_HOME"

PASS=0
FAIL=0
RESULTS=()

# run <label> <expect_zero|expect_nonzero> <shell command>
run() {
    local label="$1"; shift
    local expect="$1"; shift
    local out
    out=$("$@" 2>&1)
    local rc=$?
    local ok=0
    if [ "$expect" = "expect_zero" ] && [ $rc -eq 0 ]; then ok=1; fi
    if [ "$expect" = "expect_nonzero" ] && [ $rc -ne 0 ]; then ok=1; fi
    if [ $ok -eq 1 ]; then
        PASS=$((PASS+1))
        RESULTS+=("PASS  $label")
    else
        FAIL=$((FAIL+1))
        RESULTS+=("FAIL  $label (rc=$rc, expect=$expect)")
        echo "---- FAIL: $label ----"
        echo "cmd: $*"
        echo "rc: $rc"
        echo "$out" | head -20
        echo "----"
    fi
}

# checkstr <label> <needle> <command-output-as-string>
checkstr() {
    local label="$1"; local needle="$2"; local hay="$3"
    if echo "$hay" | grep -qF "$needle"; then
        PASS=$((PASS+1))
        RESULTS+=("PASS  $label")
    else
        FAIL=$((FAIL+1))
        RESULTS+=("FAIL  $label (needle='$needle' not in output)")
        echo "---- FAIL: $label ----"
        echo "needle: $needle"
        echo "haystack:"
        echo "$hay" | head -10
        echo "----"
    fi
}

scenario() {
    local n=$1; local desc=$2
    echo
    echo "=========================================="
    echo "  Scenario $n: $desc"
    echo "=========================================="
    # Clean any residual spwn containers from a prior scenario so
    # docker ps probes aren't confused by them.
    local ids
    ids=$(docker ps -aq --filter 'name=world-' 2>/dev/null)
    if [ -n "$ids" ]; then
        docker rm -f $ids > /dev/null 2>&1
    fi
    local d="$TMP_ROOT/s$(printf '%02d' $n)"
    rm -rf "$d"; mkdir -p "$d"; cd "$d" || exit
}

# ────────────────────────────────────────────────────────────────
# SCENARIOS
# ────────────────────────────────────────────────────────────────

s1() {
    scenario 1 "fresh init + check"
    run "s1.init"   expect_zero "$SPWN" init
    run "s1.check"  expect_zero "$SPWN" check
    local json; json=$("$SPWN" check --json 2>&1)
    checkstr "s1.checkjson-valid" '"valid": true' "$json"
}

s2() {
    scenario 2 "init into non-empty dir refuses without --force"
    echo "version: 1" > spwn.yaml
    run "s2.init-no-force" expect_nonzero "$SPWN" init
    run "s2.init-force"    expect_zero    "$SPWN" init --force
    run "s2.check"         expect_zero    "$SPWN" check
}

s3() {
    scenario 3 "init spwn:matrix scaffold"
    run "s3.init-matrix" expect_zero "$SPWN" init spwn:matrix
    if [ -d "spwn/agents/neo" ]; then
        PASS=$((PASS+1)); RESULTS+=("PASS  s3.neo-present")
    else
        FAIL=$((FAIL+1)); RESULTS+=("FAIL  s3.neo-present")
    fi
    run "s3.check" expect_zero "$SPWN" check
}

s4() {
    scenario 4 "init spwn:startup (3 agents)"
    run "s4.init-startup" expect_zero "$SPWN" init spwn:startup
    run "s4.check"        expect_zero "$SPWN" check
    local out; out=$("$SPWN" agent ls --json 2>&1)
    checkstr "s4.ceo"     "ceo"     "$out"
    checkstr "s4.devops"  "devops"  "$out"
    checkstr "s4.analyst" "analyst" "$out"
}

s5() {
    scenario 5 "init bogus ref errors"
    run "s5.init-bogus" expect_nonzero "$SPWN" init spwn:does-not-exist-xyz-987
}

s6() {
    scenario 6 "spwn up with no project (global-mode fallback)"
    # Global mode is the legacy fallback; should NOT crash. rc=0 is
    # valid (falls back to ~/.spwn/worlds/default.yaml) and rc=1 is
    # valid (no global config yet). Anything else is a bug.
    "$SPWN" up > /dev/null 2>&1
    local rc=$?
    "$SPWN" down > /dev/null 2>&1
    if [ $rc -le 1 ]; then
        PASS=$((PASS+1)); RESULTS+=("PASS  s6.up-no-project-clean")
    else
        FAIL=$((FAIL+1)); RESULTS+=("FAIL  s6.up-no-project-clean (rc=$rc)")
    fi
}

s7() {
    scenario 7 "double spwn up (idempotent) — using matrix scaffold"
    # Default scaffold pulls in local:greet which requires real image
    # build to have the binary; SPWN_BASE_IMAGE=mock has no greet.
    # Use matrix scaffold which has no local tool deps.
    run "s7.init" expect_zero "$SPWN" init spwn:matrix
    run "s7.up-1" expect_zero "$SPWN" up
    # Second up should either no-op or clean error; not a panic.
    "$SPWN" up > /dev/null 2>&1
    local rc=$?
    # Accept 0 or 1; any other rc means crash.
    if [ $rc -eq 0 ] || [ $rc -eq 1 ]; then
        PASS=$((PASS+1)); RESULTS+=("PASS  s7.up-2-clean")
    else
        FAIL=$((FAIL+1)); RESULTS+=("FAIL  s7.up-2-clean (rc=$rc)")
    fi
    "$SPWN" down > /dev/null 2>&1
}

s8() {
    scenario 8 "spwn down with no worlds"
    run "s8.init" expect_zero "$SPWN" init
    # No up; down should be clean.
    "$SPWN" down > /dev/null 2>&1
    local rc=$?
    if [ $rc -le 1 ]; then
        PASS=$((PASS+1)); RESULTS+=("PASS  s8.down-empty")
    else
        FAIL=$((FAIL+1)); RESULTS+=("FAIL  s8.down-empty (rc=$rc)")
    fi
}

s9() {
    scenario 9 "corrupted spwn.yaml → check errors cleanly"
    printf "%s\n" "invalid: [yaml broken" > spwn.yaml
    run "s9.check-corrupt" expect_nonzero "$SPWN" check
}

s10() {
    scenario 10 "invalid dep ref in agent.yaml"
    "$SPWN" init > /dev/null 2>&1
    local f="spwn/agents/neo/agent.yaml"
    python3 -c "
import sys
with open('$f') as fh: t = fh.read()
t = t.replace('spwn:unix', 'spwn:does-not-exist-zz')
with open('$f', 'w') as fh: fh.write(t)
"
    run "s10.check-bad-ref" expect_nonzero "$SPWN" check
}

s11() {
    scenario 11 "agent new"
    "$SPWN" init > /dev/null 2>&1
    run "s11.new" expect_zero "$SPWN" agent new bob
    [ -f "spwn/agents/bob/agent.yaml" ] && [ -f "spwn/agents/bob/SOUL.md" ] \
        && { PASS=$((PASS+1)); RESULTS+=("PASS  s11.files-exist"); } \
        || { FAIL=$((FAIL+1)); RESULTS+=("FAIL  s11.files-exist"); }
}

s12() {
    scenario 12 "agent new duplicate refuses without --force"
    "$SPWN" init > /dev/null 2>&1
    "$SPWN" agent new bob > /dev/null 2>&1
    run "s12.dup-no-force" expect_nonzero "$SPWN" agent new bob
    run "s12.dup-force"    expect_zero    "$SPWN" agent new bob --force
}

s13() {
    scenario 13 "agent ls + --json"
    "$SPWN" init > /dev/null 2>&1
    "$SPWN" agent new bob > /dev/null 2>&1
    run "s13.ls"     expect_zero "$SPWN" agent ls
    local json; json=$("$SPWN" agent ls --json 2>&1)
    checkstr "s13.ls-json-bob"  "bob"  "$json"
    checkstr "s13.ls-json-neo"  "neo"  "$json"
}

s14() {
    scenario 14 "agent inspect / missing agent"
    "$SPWN" init > /dev/null 2>&1
    run "s14.inspect-neo"     expect_zero    "$SPWN" agent inspect neo
    run "s14.inspect-missing" expect_nonzero "$SPWN" agent inspect does-not-exist
}

s15() {
    scenario 15 "agent rm"
    "$SPWN" init > /dev/null 2>&1
    "$SPWN" agent new bob > /dev/null 2>&1
    run "s15.rm"         expect_zero    "$SPWN" agent rm bob
    run "s15.rm-missing" expect_nonzero "$SPWN" agent rm bob
}

s16() {
    scenario 16 "agent export produces tar.gz"
    "$SPWN" init > /dev/null 2>&1
    "$SPWN" agent new bob > /dev/null 2>&1
    run "s16.export" expect_zero "$SPWN" agent export bob
    local tar
    tar=$(ls bob*.tar.gz 2>/dev/null | head -1)
    if [ -n "$tar" ] && file "$tar" | grep -qi gzip; then
        PASS=$((PASS+1)); RESULTS+=("PASS  s16.archive-is-gzip")
    else
        FAIL=$((FAIL+1)); RESULTS+=("FAIL  s16.archive-is-gzip (found='$tar')")
    fi
}

s17() {
    scenario 17 "agent import round-trip"
    "$SPWN" init > /dev/null 2>&1
    "$SPWN" agent new bob > /dev/null 2>&1
    "$SPWN" agent export bob > /dev/null 2>&1
    local tar
    tar=$(ls bob*.tar.gz 2>/dev/null | head -1)
    "$SPWN" agent rm bob > /dev/null 2>&1
    run "s17.import" expect_zero "$SPWN" agent import "$tar"
    local json; json=$("$SPWN" agent ls --json 2>&1)
    checkstr "s17.bob-back" "bob" "$json"
}

s18() {
    scenario 18 "agent fork"
    "$SPWN" init > /dev/null 2>&1
    "$SPWN" agent new bob > /dev/null 2>&1
    run "s18.fork" expect_zero "$SPWN" agent fork bob bob-v2
    local json; json=$("$SPWN" agent ls --json 2>&1)
    checkstr "s18.original" "bob"    "$json"
    checkstr "s18.forked"   "bob-v2" "$json"
}

s19() {
    scenario 19 "spwn inspect project and single agent"
    "$SPWN" init > /dev/null 2>&1
    "$SPWN" agent new bob > /dev/null 2>&1
    run "s19.inspect-all" expect_zero "$SPWN" inspect
    run "s19.inspect-one" expect_zero "$SPWN" inspect bob
    run "s19.inspect-missing" expect_nonzero "$SPWN" inspect does-not-exist
}

s20() {
    scenario 20 "skill new / ls / rm"
    "$SPWN" init > /dev/null 2>&1
    run "s20.skill-new" expect_zero "$SPWN" skill new daily-standup
    [ -f "spwn/skills/daily-standup.md" ] \
        && { PASS=$((PASS+1)); RESULTS+=("PASS  s20.file-exists"); } \
        || { FAIL=$((FAIL+1)); RESULTS+=("FAIL  s20.file-exists"); }
    run "s20.skill-ls"  expect_zero "$SPWN" skill ls
    run "s20.skill-rm"  expect_zero "$SPWN" skill rm daily-standup
    [ ! -f "spwn/skills/daily-standup.md" ] \
        && { PASS=$((PASS+1)); RESULTS+=("PASS  s20.file-gone"); } \
        || { FAIL=$((FAIL+1)); RESULTS+=("FAIL  s20.file-gone"); }
}

s21() {
    scenario 21 "install python globally"
    "$SPWN" init > /dev/null 2>&1
    run "s21.install" expect_zero "$SPWN" install python
    grep -q "spwn:python" spwn/agents/neo/agent.yaml \
        && { PASS=$((PASS+1)); RESULTS+=("PASS  s21.in-agent-yaml"); } \
        || { FAIL=$((FAIL+1)); RESULTS+=("FAIL  s21.in-agent-yaml"); }
    grep -q "spwn:python" spwn.lock \
        && { PASS=$((PASS+1)); RESULTS+=("PASS  s21.in-lockfile"); } \
        || { FAIL=$((FAIL+1)); RESULTS+=("FAIL  s21.in-lockfile"); }
}

s22() {
    scenario 22 "install python --agent bob only"
    "$SPWN" init > /dev/null 2>&1
    "$SPWN" agent new bob > /dev/null 2>&1
    run "s22.install-agent" expect_zero "$SPWN" install node --agent bob
    grep -q "spwn:node" spwn/agents/bob/agent.yaml \
        && { PASS=$((PASS+1)); RESULTS+=("PASS  s22.bob-has"); } \
        || { FAIL=$((FAIL+1)); RESULTS+=("FAIL  s22.bob-has"); }
    grep -q "spwn:node" spwn/agents/neo/agent.yaml \
        && { FAIL=$((FAIL+1)); RESULTS+=("FAIL  s22.neo-clean (bled over)"); } \
        || { PASS=$((PASS+1)); RESULTS+=("PASS  s22.neo-clean"); }
}

s23() {
    scenario 23 "uninstall removes dep"
    "$SPWN" init > /dev/null 2>&1
    "$SPWN" install python > /dev/null 2>&1
    run "s23.uninstall" expect_zero "$SPWN" uninstall python
    grep -q "spwn:python" spwn/agents/neo/agent.yaml \
        && { FAIL=$((FAIL+1)); RESULTS+=("FAIL  s23.still-present"); } \
        || { PASS=$((PASS+1)); RESULTS+=("PASS  s23.removed"); }
}

s24() {
    scenario 24 "install bogus ref errors"
    "$SPWN" init > /dev/null 2>&1
    run "s24.install-bogus" expect_nonzero "$SPWN" install spwn:does-not-exist-xyz
    # Lockfile should be unchanged — let's just verify it doesn't contain the bogus
    grep -q "does-not-exist-xyz" spwn.lock 2>/dev/null \
        && { FAIL=$((FAIL+1)); RESULTS+=("FAIL  s24.lock-polluted"); } \
        || { PASS=$((PASS+1)); RESULTS+=("PASS  s24.lock-clean"); }
}

s25() {
    scenario 25 "install local skill without file errors"
    "$SPWN" init > /dev/null 2>&1
    run "s25.install-missing-skill" expect_nonzero "$SPWN" install skill:does-not-exist --agent neo
}

s26() {
    scenario 26 "install bare name → spwn: scheme"
    "$SPWN" init > /dev/null 2>&1
    run "s26.install-bare" expect_zero "$SPWN" install node
    grep -qE "spwn:node|node" spwn/agents/neo/agent.yaml \
        && { PASS=$((PASS+1)); RESULTS+=("PASS  s26.resolved-to-spwn-node"); } \
        || { FAIL=$((FAIL+1)); RESULTS+=("FAIL  s26.resolved-to-spwn-node"); }
}

s27() {
    scenario 27 "install github:ref planned/error (not crash)"
    "$SPWN" init > /dev/null 2>&1
    "$SPWN" install github:jterrazz/does-not-exist > /dev/null 2>&1
    local rc=$?
    if [ $rc -eq 0 ] || [ $rc -eq 1 ]; then
        PASS=$((PASS+1)); RESULTS+=("PASS  s27.github-clean")
    else
        FAIL=$((FAIL+1)); RESULTS+=("FAIL  s27.github-clean (rc=$rc)")
    fi
}

s28() {
    scenario 28 "install local ref without --agent requires --agent"
    "$SPWN" init > /dev/null 2>&1
    echo "# test" > spwn/skills/myskill.md
    # Without --agent, should error (local refs require target).
    run "s28.local-no-agent" expect_nonzero "$SPWN" install skill:myskill
}

s29() {
    scenario 29 "lockfile determinism"
    "$SPWN" init > /dev/null 2>&1
    "$SPWN" install python > /dev/null 2>&1
    "$SPWN" install node > /dev/null 2>&1
    local sorted
    sorted=$(grep '^spwn:' spwn.lock | sort)
    local actual
    actual=$(grep '^spwn:' spwn.lock)
    if [ "$sorted" = "$actual" ]; then
        PASS=$((PASS+1)); RESULTS+=("PASS  s29.lock-sorted")
    else
        FAIL=$((FAIL+1)); RESULTS+=("FAIL  s29.lock-sorted")
    fi
}

s30() {
    scenario 30 "check after multiple installs"
    "$SPWN" init > /dev/null 2>&1
    "$SPWN" install python > /dev/null 2>&1
    "$SPWN" install node > /dev/null 2>&1
    "$SPWN" install git > /dev/null 2>&1
    run "s30.check" expect_zero "$SPWN" check
}

# ───────── Docker-backed (31-40) ─────────

s31() {
    scenario 31 "matrix world spawn, ls shows running"
    "$SPWN" init spwn:matrix > /dev/null 2>&1
    run "s31.up" expect_zero "$SPWN" up
    # spwn ls has no --json; use plaintext and check the table.
    local out; out=$("$SPWN" ls 2>&1)
    checkstr "s31.running" "running" "$out"
    "$SPWN" down > /dev/null 2>&1
}

s32() {
    scenario 32 "world inspect shows project-rooted agent home"
    "$SPWN" init spwn:matrix > /dev/null 2>&1
    "$SPWN" up > /dev/null 2>&1
    local id; id=$("$SPWN" world ls --json 2>&1 | python3 -c 'import sys,json; d=json.load(sys.stdin); w=d.get("worlds") or d; print(w[0]["id"] if isinstance(w,list) and w else (w[0].get("id") or w[0].get("name")))' 2>/dev/null)
    if [ -z "$id" ]; then
        # Fall back to parsing docker ps directly.
        id=$(docker ps --format '{{.Names}}' --filter 'name=world-' | head -1)
    fi
    if [ -n "$id" ]; then
        local out; out=$("$SPWN" world inspect "$id" 2>&1)
        # Expect the project path, not ~/.spwn/agents.
        if echo "$out" | grep -qE "/tmp/qa-50/s32/spwn/agents|qa-50.*spwn/agents"; then
            PASS=$((PASS+1)); RESULTS+=("PASS  s32.home-is-project-path")
        else
            FAIL=$((FAIL+1)); RESULTS+=("FAIL  s32.home-is-project-path")
            echo "got: $out" | head -20
        fi
    else
        FAIL=$((FAIL+1)); RESULTS+=("FAIL  s32.could-not-find-world-id")
    fi
    "$SPWN" down > /dev/null 2>&1
}

s33() {
    scenario 33 "world ls --json shape"
    "$SPWN" init spwn:matrix > /dev/null 2>&1
    "$SPWN" up > /dev/null 2>&1
    local out; out=$("$SPWN" world ls --json 2>&1)
    # Should parse as JSON and contain the world.
    if echo "$out" | python3 -c 'import sys,json; json.load(sys.stdin)' 2>/dev/null; then
        PASS=$((PASS+1)); RESULTS+=("PASS  s33.valid-json")
    else
        FAIL=$((FAIL+1)); RESULTS+=("FAIL  s33.valid-json")
    fi
    checkstr "s33.has-worlds-key" "worlds" "$out"
    "$SPWN" down > /dev/null 2>&1
}

s34() {
    scenario 34 "down individual world"
    "$SPWN" init spwn:matrix > /dev/null 2>&1
    "$SPWN" up > /dev/null 2>&1
    local id; id=$(docker ps --format '{{.Names}}' --filter 'name=world-' | head -1)
    if [ -n "$id" ]; then
        run "s34.down-id" expect_zero "$SPWN" down "$id"
    else
        FAIL=$((FAIL+1)); RESULTS+=("FAIL  s34.no-world-to-destroy")
    fi
}

s35() {
    scenario 35 "ls shows nothing after down"
    "$SPWN" init spwn:matrix > /dev/null 2>&1
    "$SPWN" up > /dev/null 2>&1
    "$SPWN" down > /dev/null 2>&1
    local json; json=$("$SPWN" ls --json 2>&1)
    if echo "$json" | grep -q "running"; then
        FAIL=$((FAIL+1)); RESULTS+=("FAIL  s35.still-running")
    else
        PASS=$((PASS+1)); RESULTS+=("PASS  s35.gone")
    fi
}

s36() {
    scenario 36 "world logs via both forms (config name + runtime id)"
    "$SPWN" init spwn:matrix > /dev/null 2>&1
    "$SPWN" up > /dev/null 2>&1
    local id; id=$(docker ps --format '{{.Names}}' --filter 'name=world-' | head -1)
    # `spwn logs --world` filters by CONFIG name.
    run "s36.logs-by-config" expect_zero "$SPWN" logs --world matrix
    # `spwn world logs <id>` filters by runtime ID.
    if [ -n "$id" ]; then
        run "s36.logs-by-id" expect_zero "$SPWN" world logs "$id"
    fi
    "$SPWN" down > /dev/null 2>&1
}

s37() {
    scenario 37 "world inspect exit code 0 for valid id, nonzero for invalid"
    "$SPWN" init spwn:matrix > /dev/null 2>&1
    "$SPWN" up > /dev/null 2>&1
    local id; id=$(docker ps --format '{{.Names}}' --filter 'name=world-' | head -1)
    if [ -n "$id" ]; then
        run "s37.inspect-valid" expect_zero "$SPWN" world inspect "$id"
    fi
    run "s37.inspect-invalid" expect_nonzero "$SPWN" world inspect world-does-not-exist-12345
    "$SPWN" down > /dev/null 2>&1
}

s38() {
    scenario 38 "knowledge path propagates into container"
    "$SPWN" init spwn:matrix > /dev/null 2>&1
    mkdir -p knowledge
    echo "test fact" > knowledge/fact.md
    # spwn:matrix scaffolds spwn.yaml#worlds.matrix without a knowledge
    # path; insert one via sed (indented 4 spaces under "  matrix:").
    # This is brittle but avoids a python yaml dep.
    python3 - <<'PY'
import re, sys
with open('spwn.yaml') as f: src = f.read()
# Locate the matrix world block and add "    knowledge: ./knowledge" after
# its first line if not present.
if 'knowledge:' not in src:
    src = re.sub(r'(\n  matrix:\n)', r'\1    knowledge: ./knowledge\n', src, count=1)
    with open('spwn.yaml', 'w') as f: f.write(src)
PY
    "$SPWN" up > /dev/null 2>&1
    local id; id=$(docker ps --format '{{.Names}}' --filter 'name=world-' | head -1)
    if [ -n "$id" ]; then
        local out; out=$(docker exec "$id" cat /world/knowledge/fact.md 2>&1)
        if echo "$out" | grep -q "test fact"; then
            PASS=$((PASS+1)); RESULTS+=("PASS  s38.knowledge-mounted")
        else
            FAIL=$((FAIL+1)); RESULTS+=("FAIL  s38.knowledge-mounted (got: $out)")
        fi
    fi
    "$SPWN" down > /dev/null 2>&1
}

s39() {
    scenario 39 "workspace mounted read-write"
    "$SPWN" init spwn:matrix > /dev/null 2>&1
    "$SPWN" up -w . > /dev/null 2>&1
    local id; id=$(docker ps --format '{{.Names}}' --filter 'name=world-' | head -1)
    if [ -n "$id" ]; then
        local out; out=$(docker exec "$id" ls /workspaces/ 2>&1)
        if [ -n "$out" ]; then
            PASS=$((PASS+1)); RESULTS+=("PASS  s39.workspaces-mounted")
        else
            FAIL=$((FAIL+1)); RESULTS+=("FAIL  s39.workspaces-mounted")
        fi
    else
        # -w . without Docker is fine; check cli rc was 0
        FAIL=$((FAIL+1)); RESULTS+=("FAIL  s39.no-world-id")
    fi
    "$SPWN" down > /dev/null 2>&1
}

s40() {
    scenario 40 "container goes away after down (project-scoped)"
    "$SPWN" init spwn:matrix > /dev/null 2>&1
    "$SPWN" up > /dev/null 2>&1
    # Filter only for matrix world containers this scenario spawned.
    local id_before; id_before=$(docker ps --format '{{.Names}}' --filter 'name=world-matrix' | head -1)
    "$SPWN" down > /dev/null 2>&1
    local id_after; id_after=$(docker ps --format '{{.Names}}' --filter 'name=world-matrix' | head -1)
    if [ -z "$id_after" ] && [ -n "$id_before" ]; then
        PASS=$((PASS+1)); RESULTS+=("PASS  s40.container-gone")
    else
        FAIL=$((FAIL+1)); RESULTS+=("FAIL  s40.container-gone (before=$id_before, after=$id_after)")
    fi
}

# ───────── Error paths + edge cases (41-50) ─────────

s41() {
    scenario 41 "malformed yaml → check gives line-number error"
    printf "version: 1\nname: bad\nworlds:\n  mat: [ broken\n" > spwn.yaml
    local out; out=$("$SPWN" check 2>&1)
    local rc=$?
    if [ $rc -ne 0 ]; then
        PASS=$((PASS+1)); RESULTS+=("PASS  s41.exit-nonzero")
    else
        FAIL=$((FAIL+1)); RESULTS+=("FAIL  s41.exit-nonzero")
    fi
    if echo "$out" | grep -qE "line|yaml"; then
        PASS=$((PASS+1)); RESULTS+=("PASS  s41.mentions-yaml-error")
    else
        FAIL=$((FAIL+1)); RESULTS+=("FAIL  s41.mentions-yaml-error")
    fi
}

s42() {
    scenario 42 "unicode agent name"
    "$SPWN" init > /dev/null 2>&1
    # Either accept (slugify) or reject cleanly.
    "$SPWN" agent new "日本" > /dev/null 2>&1
    local rc=$?
    if [ $rc -le 1 ]; then
        PASS=$((PASS+1)); RESULTS+=("PASS  s42.unicode-clean")
    else
        FAIL=$((FAIL+1)); RESULTS+=("FAIL  s42.unicode-clean (rc=$rc)")
    fi
}

s43() {
    scenario 43 "very long agent name (slug boundary)"
    "$SPWN" init > /dev/null 2>&1
    local name; name=$(printf 'x%.0s' {1..80})
    "$SPWN" agent new "$name" > /dev/null 2>&1
    local rc=$?
    # Accept either reject-with-clean-error or accept-with-truncated-slug.
    if [ $rc -le 1 ]; then
        PASS=$((PASS+1)); RESULTS+=("PASS  s43.long-name-clean")
    else
        FAIL=$((FAIL+1)); RESULTS+=("FAIL  s43.long-name-clean (rc=$rc)")
    fi
}

s44() {
    scenario 44 "empty dependencies list"
    "$SPWN" init > /dev/null 2>&1
    # Write a fresh minimal agent.yaml with empty deps (keep name +
    # description so the validator's required-fields rule is happy).
    cat > spwn/agents/neo/agent.yaml <<'YAML'
name: neo
description: minimal agent with no deps
runtime:
  backend: "spwn:claude-code"
dependencies: []
YAML
    run "s44.check" expect_zero "$SPWN" check
}

s45() {
    scenario 45 "corrupted spwn.lock"
    "$SPWN" init > /dev/null 2>&1
    printf "garbage content\n" > spwn.lock
    # check should either flag or silently tolerate (it's regenerated on install).
    "$SPWN" check > /dev/null 2>&1
    local rc=$?
    if [ $rc -le 1 ]; then
        PASS=$((PASS+1)); RESULTS+=("PASS  s45.check-clean")
    else
        FAIL=$((FAIL+1)); RESULTS+=("FAIL  s45.check-clean")
    fi
    # Install should still succeed — regenerates.
    run "s45.install-regens" expect_zero "$SPWN" install python
}

s46() {
    scenario 46 "project discovery walks upward"
    "$SPWN" init > /dev/null 2>&1
    mkdir -p a/b/c
    cd a/b/c || return
    run "s46.check-from-subdir" expect_zero "$SPWN" check
}

s47() {
    scenario 47 "sibling projects isolated"
    mkdir p1 p2
    (cd p1 && "$SPWN" init > /dev/null 2>&1 && "$SPWN" agent new alice > /dev/null 2>&1)
    (cd p2 && "$SPWN" init > /dev/null 2>&1 && "$SPWN" agent new bob > /dev/null 2>&1)
    # From p1, don't see bob.
    cd p1 || return
    local j; j=$("$SPWN" agent ls --json 2>&1)
    if echo "$j" | grep -q bob; then
        FAIL=$((FAIL+1)); RESULTS+=("FAIL  s47.p1-isolated")
    else
        PASS=$((PASS+1)); RESULTS+=("PASS  s47.p1-isolated")
    fi
    checkstr "s47.p1-has-alice" "alice" "$j"
}

s48() {
    scenario 48 "spwn --version"
    run "s48.version" expect_zero "$SPWN" --version
}

s49() {
    scenario 49 "top-level help"
    local out; out=$("$SPWN" --help 2>&1)
    checkstr "s49.has-quickstart" "Quick Start"  "$out"
    checkstr "s49.has-entities"   "Entities"     "$out"
    # `spwn help` is an alias for --help.
    local out2; out2=$("$SPWN" help 2>&1)
    checkstr "s49.help-alias"     "Quick Start"  "$out2"
}

s50() {
    scenario 50 "sanity: the 3 core verbs work end-to-end"
    "$SPWN" init spwn:matrix > /dev/null 2>&1
    run "s50.check" expect_zero "$SPWN" check
    run "s50.up"    expect_zero "$SPWN" up
    local id; id=$(docker ps --format '{{.Names}}' --filter 'name=world-matrix' | head -1)
    if [ -n "$id" ]; then
        # A real probe: can we exec a command inside the container?
        run "s50.exec-inside" expect_zero docker exec "$id" sh -c "ls /agents/neo/CLAUDE.md"
    fi
    run "s50.down" expect_zero "$SPWN" down
}

# ──────────── run selection ────────────
first=${1:-1}
last=${2:-50}
for i in $(seq "$first" "$last"); do
    fn="s$i"
    if declare -f "$fn" > /dev/null; then
        "$fn"
    fi
done

echo
echo "=========================================="
echo "  RESULTS: $PASS passed, $FAIL failed"
echo "=========================================="
for r in "${RESULTS[@]}"; do echo "$r"; done

exit $FAIL

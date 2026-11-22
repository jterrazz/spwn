#!/bin/bash
# hazard: run it with run_in_background, once per launch, never inside `timeout`.
. "$(dirname "$0")/env.sh"
[ $# -eq 3 ] || { echo "usage: tip-mac.sh <worktree> <tip sha, 9 chars> <label>" >&2; exit 2; }
W=$1; T=$2; L=$3
case $L in "" | *[!A-Za-z0-9._-]*) echo "bad label: $L"; exit 2 ;; esac
cd "$W/machines" && test "$(git rev-parse --short=9 HEAD)" = "$T" || { echo "HEAD moved"; exit 2; }
bash $GATE_DIR/gate-slot.sh specs/machine/menubar/run.sh $L-tip > $RUN/station-$L-mbtip.log 2>&1; echo "MB-PLAIN rc=$? $(grep -E '^(PASS|FAIL) menubar' $RUN/station-$L-mbtip.log)"; grep -E "Testing \[" .artifacts/e2e/menubar-$L-tip.log | tail -1
for k in 1 2; do test "$(git rev-parse --short=9 HEAD)" = "$T" || exit 2
  bash $GATE_DIR/gate-slot.sh specs/machine/menubar/run.sh $L-tipu$k --update-goldens > $RUN/station-$L-mbtipu$k.log 2>&1; echo "MB-UPDATE$k rc=$? $(grep -E '^(PASS|FAIL) menubar' $RUN/station-$L-mbtipu$k.log) written=$(find .artifacts/e2e/menubar-$L-tipu$k/changes -type f 2>/dev/null | wc -l | tr -d ' ')"
done
GATE_SLOT_PRIORITY=low bash $GATE_DIR/gate-slot.sh bash $GATE_DIR/darwin-build.sh "$W" $L > $RUN/station-$L-dw.log 2>&1; echo "DARWIN rc=$? $(grep -E '(PASS|FAIL) darwin-build' $RUN/station-$L-dw.log | tail -1)"
echo "STATUS-AFTER[$(git -C "$W" status --porcelain | tr '\n' ' ')]"

#!/bin/bash
# usage: [GATE_SLOT_PRIORITY=landing|normal|low] [GATE_VITEST_WORKERS=12] cold-gate.sh <worktree> <branch> <label>  full cold make check of the branch tip, in a fresh clone on the shared Linux VM, inside the gate slot; log $RUN/station-<label>-cold.log
# The worktree may be the repository's root or its machines/ folder: the gate runs in machines/ either way (given the root, npm ci would meet the root's pnpm devEngines and exit 97).
# Files on the VM are in ~/$VM_DIR/ (GATE_VM_DIR), the VM is GATE_VM_LINUX. The run holds /var/lock/heavy-run exclusive (alone on the VM: the loops' shared holds drain first, other sessions' heavy runs wait),
# and the slot also reads the VM's load. The VM's cron prunes go-build. This script takes the slot itself, so it is never wrapped in gate-slot.sh.
# A landing gate (the train's tip) runs at GATE_SLOT_PRIORITY=landing, which every other waiter of the slot yields to.
# invariant: GEN through STORIES run behind the VM-wide story slot; a child of the gate never inherits a lock's descriptor (see nl in cc-remote.sh).
. "$(dirname "$0")/env.sh"
W=$1; B=$2; L=$3; H=$(git -C "$W" rev-parse "$B")
sub=$(git -C "$W" rev-parse --show-prefix)  # "" at a repo root, "machines/" in a subdir
[ -z "$sub" ] && [ -d "$W/machines/apps/engine" ] && sub=machines/
VM=$VM_LINUX; R='$HOME'/$VM_DIR; RS=$VM_DIR/
git -C "$W" bundle create "$RUN/$L-cold.bundle" "$B" --tags 2>/dev/null || exit 90
cat > "$RUN/$L-cold-remote.sh" <<R
export PATH=\$HOME/.local/bin:\$PATH; eval "\$(mise activate bash --shims)"
df -h ~ | tail -1; echo "vm $VM \$(nproc) cpu \$(free -g | awk '/Mem:/{print \$2}') GB load \$(cut -d' ' -f1-3 /proc/loadavg)"
rm -rf $R/wt-$L-cold; git clone -q -b $B $R/$L-cold.bundle $R/wt-$L-cold && cd $R/wt-$L-cold && git log -1 --oneline || exit 99
test "\$(git rev-parse HEAD)" = "$H" || { echo WRONG-HEAD; exit 98; }
$([ -n "$sub" ] && echo "cd $sub || exit 95")
nl(){ "\$@" 3>&- 4>&- 5>&- 6>&- 7>&- 8>&- 9>&-; }
# GEN through STORIES (and only those) sit behind a VM-wide slot (2 holders: /tmp/story-slot-{1,2}, flock on fd 8): taken once, held to the end of STORIES; else wait (<= 60 min, a line a minute).
take_slot(){ [ -n "\$got" ] && return 0; local P=\${SLOT_POLL:-20} w=0 k
  while :; do
    for k in 1 2; do exec 8>/tmp/story-slot-\$k; flock -n 8 && { got=\$k; echo "STORY-SLOT \$got taken"; return 0; }; exec 8>&-; done
    [ \$w -ge \$((3600/P)) ] && { echo "STORY-SLOT: both held for 60 min"; exit 84; }
    [ \$((w % (60/P))) = 0 ] && echo "STORY-SLOT: both held, waiting \$(date +%T)"
    w=\$((w+1)); command sleep \$P
  done
}; got=
echo "t-npm \$(date +%T)"
npm ci >/tmp/$L-npm.log 2>&1 || { tail -20 /tmp/$L-npm.log; exit 97; }
(cd "\$(ls -d apps/desktop apps/window 2>/dev/null | head -1)" 2>/dev/null && npx playwright install chromium-headless-shell >/tmp/$L-pw.log 2>&1) || { tail -5 /tmp/$L-pw.log; exit 96; }
echo "t-generate \$(date +%T)"
take_slot; go work sync; nl make generate >/tmp/$L-gen.log 2>&1; echo "generate=\$?"; exec 8>&-; git status --porcelain | head
echo "t-check \$(date +%T) workers=${GATE_VITEST_WORKERS:-12}"
VITEST_MAX_WORKERS=${GATE_VITEST_WORKERS:-12} nl make check >/tmp/$L-check.log 2>&1; rc=\$?; echo "check=\$rc"
echo "t-check-done \$(date +%T)"
grep -E "FAIL|Test Files" /tmp/$L-check.log | sed 's/\x1b\[[0-9;]*m//g' | tail -20
tail -15 /tmp/$L-check.log | sed 's/\x1b\[[0-9;]*m//g'
cp /tmp/$L-check.log $R/$L-check.log 2>/dev/null; cd ~ && rm -rf $R/wt-$L-cold $R/$L-cold.bundle /tmp/$L-*.log
echo COLD-DONE
exit \$rc
R
mkdir -p "$RUN/vm-hold"; trap 'rm -f "$RUN/vm-hold/$VM" "$RUN/vm-hold/all"' EXIT
ssh -n $VM "mkdir -p $R" || exit 91
run="\$HOME/.local/bin/heavy-run-excl 5400 bash $R/$L-cold-remote.sh"
scp -q "$RUN/$L-cold.bundle" "$RUN/$L-cold-remote.sh" $VM:$RS || exit 91
rm -f "$RUN/$L-cold.bundle"
GATE_SLOT_VM=$VM "$GATE_DIR/gate-slot.sh" "$GATE_DIR/hold-then.sh" "$VM" ssh -n $VM "$run; rc=\$?; rm -f $R/$L-cold-remote.sh; exit \$rc" > "$RUN/station-$L-cold.log" 2>&1
rc=$?; echo "slot-exit=$rc" >> "$RUN/station-$L-cold.log"; scp -q $VM:$RS$L-check.log "$RUN/station-$L-check-full.log" 2>/dev/null; ssh -n $VM "rm -f $R/$L-check.log"; rm -f "$RUN/$L-cold-remote.sh"; exit $rc

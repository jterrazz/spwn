#!/bin/bash
# invariant: GEN through STORIES run behind the VM-wide story slot; a child of the gate never inherits a lock's descriptor (see nl in cc-remote.sh).
. "$(dirname "$0")/env.sh"
W=$1; B=$2; L=$3; H=$(git -C "$W" rev-parse "$B")
sub=$(git -C "$W" rev-parse --show-prefix)  # "" at a repo root, "machines/" in a subdir
[ -z "$sub" ] && [ -d "$W/machines/apps/engine" ] && sub=machines/
# invariant: the clone is one fixed path, $VM_RAM/cold, since the exclusive heavy-run lock lets one cold gate run at a time, so Go's build cache, keyed by the directory, hits from one gate to the next.
VM=$VM_LINUX; R='$HOME'/$VM_DIR; RS=$VM_DIR/; C=$VM_RAM/cold
git -C "$W" bundle create "$RUN/$L-cold.bundle" "$B" --tags 2>/dev/null || exit 90
cat > "$RUN/$L-cold-remote.sh" <<R
export PATH=\$HOME/.local/bin:\$PATH; eval "\$(mise activate bash --shims)"
mkdir -p $VM_RAM; df -h ~ $VM_RAM /tmp | tail -n +2; echo "vm $VM \$(nproc) cpu \$(free -g | awk '/Mem:/{print \$2}') GB load \$(cut -d' ' -f1-3 /proc/loadavg)"
rm -rf $C; git clone -q -b $B $R/$L-cold.bundle $C && cd $C && git log -1 --oneline || exit 99
test "\$(git rev-parse HEAD)" = "$H" || { echo WRONG-HEAD; exit 98; }
$([ -n "$sub" ] && echo "cd $sub || exit 95")
nl(){ "\$@" 3>&- 4>&- 5>&- 6>&- 7>&- 8>&- 9>&-; }
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
echo "t-rootlint \$(date +%T)"
(cd "\$(git rev-parse --show-toplevel)" && nl pnpm install --frozen-lockfile >/tmp/$L-rootinstall.log 2>&1 && nl make lint >/tmp/$L-rootlint.log 2>&1); rl=\$?; echo "rootlint=\$rl"
[ \$rl = 0 ] || { sed 's/\x1b\[[0-9;]*m//g' /tmp/$L-rootlint.log | grep -vE '^\\s*\$' | tail -25; [ \$rc = 0 ] && rc=\$rl; }
grep -E "FAIL|Test Files" /tmp/$L-check.log | sed 's/\x1b\[[0-9;]*m//g' | tail -20
tail -15 /tmp/$L-check.log | sed 's/\x1b\[[0-9;]*m//g'
cp /tmp/$L-check.log $R/$L-check.log 2>/dev/null; cd ~ && rm -rf $C $R/$L-cold.bundle /tmp/$L-*.log
echo COLD-DONE
exit \$rc
R
mkdir -p "$RUN/vm-hold"; trap 'rm -f "$RUN/vm-hold/$VM" "$RUN/vm-hold/all"' EXIT
"$GATE_SSH" ssh -n $VM "mkdir -p $R" || exit 91
run="\$HOME/.local/bin/heavy-run-excl 5400 bash $R/$L-cold-remote.sh"
"$GATE_SSH" scp -q "$RUN/$L-cold.bundle" "$RUN/$L-cold-remote.sh" $VM:$RS || exit 91
rm -f "$RUN/$L-cold.bundle"
GATE_SLOT_VM=$VM "$GATE_DIR/gate-slot.sh" "$GATE_DIR/hold-then.sh" "$VM" "$GATE_SSH" ssh -n $VM "$run; rc=\$?; rm -f $R/$L-cold-remote.sh; exit \$rc" > "$RUN/station-$L-cold.log" 2>&1
rc=$?; echo "slot-exit=$rc" >> "$RUN/station-$L-cold.log"; "$GATE_SSH" scp -q $VM:$RS$L-check.log "$RUN/station-$L-check-full.log" 2>/dev/null; "$GATE_SSH" ssh -n $VM "rm -f $R/$L-check.log"; rm -f "$RUN/$L-cold-remote.sh"; exit $rc

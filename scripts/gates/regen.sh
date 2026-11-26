#!/bin/bash
# hazard: `one` waits for nothing: run it with run_in_background and wait on its end file, never on a name.
. "$(dirname "$0")/env.sh"
usage(){ echo "usage: regen.sh one <worktree> <sha> <label> [aria vitest filter]; regen.sh all <worktree> <map file> <label prefix> \"<lane A positions>\" \"<lane B positions>\"" >&2; exit 2; }
ok(){ case $1 in "" | *[!A-Za-z0-9._-]*) echo "bad name: $1"; exit 2 ;; esac; }
case $1 in
one)
  [ $# -ge 4 ] || usage; W=$2; SHA=$3; L=$4; ARIA=$5; ok "$L"; B=regen-$L-tmp; VM=$VM_LINUX; RS=$VM_DIR/; R='$HOME'/$VM_DIR
  rm -f "${RUN:?}/$L-regen.end"; trap 'echo END > "$RUN/$L-regen.end"' EXIT
  git -C "$W" branch -f $B $SHA >/dev/null || exit 89
  git -C "$W" bundle create $RUN/$L-regen.bundle "$B" --tags >/dev/null 2>&1 || exit 90
  ssh -n $VM "mkdir -p $R" && scp -q $RUN/$L-regen.bundle $VM:$RS && scp -q $GATE_DIR/regen-remote.sh $VM:$RS$L-regen-remote.sh || exit 91; rm -f $RUN/$L-regen.bundle
  ssh -n $VM "CC_HOME=$R CC_WORK=$VM_RAM bash $R/$L-regen-remote.sh $L $B $ARIA"; rc=$?
  rm -rf "${RUN:?}/${L:?}-regen-out"; mkdir -p $RUN/$L-regen-out; scp -q $VM:${RS}$L-regen-r.tar $RUN/$L-regen-out/r.tar && scp -q $VM:${RS}$L-regen-changes.txt $RUN/$L-regen-out/changes.txt && tar -C $RUN/$L-regen-out -xf $RUN/$L-regen-out/r.tar || rc=1
  ssh -n $VM "rm -rf $VM_RAM/${L}-regen-node; cd $R && rm -rf ${L}-regen.bundle ${L}-regen-r.tar ${L}-regen-remote.sh"
  git -C "$W" branch -D $B >/dev/null; exit $rc ;;
all)
  [ $# -eq 6 ] || usage; W=$2; MAP=$3; P=$4; ok "$P"; [ -f "$MAP" ] || { echo "no map file $MAP"; exit 2; }
  rm -f "${RUN:?}/$P-regen-all.end"; trap 'echo END > "$RUN/$P-regen-all.end"' EXIT
  sha(){ awk -v n=$1 '$1==n{print $2}' "$MAP"; }
  lane(){ local bad=0 rc; for n in $1; do echo "== c$n $(sha $n) $(date +%T)"; bash $GATE_DIR/regen.sh one $W $(sha $n) $P-c$n > $RUN/station-$P-regen-c$n.log 2>&1; rc=$?; echo "== c$n done rc=$rc $(date +%T)"; [ $rc = 0 ] || bad=1; done; return $bad; }
  lane "$5" > $RUN/station-$P-regen-laneA.log 2>&1 &
  pa=$!
  lane "$6" > $RUN/station-$P-regen-laneB.log 2>&1 &
  pb=$!
  wait $pa; ra=$?; wait $pb; rb=$?
  [ $ra = 0 ] && [ $rb = 0 ] || exit 1 ;;
*) usage ;;
esac

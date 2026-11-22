#!/bin/bash
# hazard: run it with run_in_background; a killed shell leaves the run going on the VM.
. "$(dirname "$0")/env.sh"
[ $# -ge 2 ] || { echo "usage: desktop-proof.sh <worktree> <label> [branch]" >&2; exit 2; }
W=$1; L=$2; VM=$VM_LINUX; RS=$VM_DIR/; R='$HOME'/$VM_DIR
case $L in "" | *[!A-Za-z0-9._-]*) echo "bad label: $L"; exit 2 ;; esac
rm -f "${RUN:?}/$L-dt.end"; trap 'echo "END $(date +%T)" > $RUN/$L-dt.end' EXIT
B=${3:-$(git -C "$W" rev-parse --abbrev-ref HEAD)} || exit 2
git -C "$W" bundle create $RUN/$L.bundle "$B" --tags >/dev/null 2>&1 || exit 90
scp -q $RUN/$L.bundle $VM:$RS$L.bundle && scp -q $GATE_DIR/desktop-proof-remote.sh $VM:$RS$L-dt-remote.sh || exit 91; rm -f $RUN/$L.bundle
ssh -n -o ConnectTimeout=10 $VM "rm -f $R/$L-dt.end; setsid nohup bash -c 'CC_HOME=$R CC_WORK=$VM_RAM bash $R/$L-dt-remote.sh $L $B > $R/$L-dt.out 2>&1; touch $R/$L-dt.end' >/dev/null 2>&1 < /dev/null & echo launched"
n=0
while :; do
  sleep 30; n=$((n+1)); [ $n -gt 360 ] && { echo "TIMEOUT 3h"; exit 1; }
  ssh -n -o ConnectTimeout=10 -o BatchMode=yes $VM "test -e $R/$L-dt.end" && break
done
ssh -n $VM "cat $R/$L-dt.out" | tee $RUN/$L-dt.out
ssh -n $VM "cd $R && rm -f ${L}-dt.out ${L}-dt.end ${L}-dt-remote.sh"

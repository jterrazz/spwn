#!/bin/bash
# usage: [GATE_RUN_DIR=<dir>] [GATE_VM_LINUX=<ssh host>] [GATE_VM_DIR=<folder>] cc.sh <worktree> <base> <label> [vitest filters...]  the per-commit loop: every commit of base..HEAD (BRANCH=<name> instead of HEAD's branch), oldest first.
# Two tiers. CHEAP=1: gofmt, vet x3, golangci, make lint, root lint, the world build: no story slot, 4 commits at once (PAR=<n>), about 1 min each. Without CHEAP, the heavy tier: the whole of cc (generate, drift, go test, stories) behind the 2 story slots, one commit at a time:
#   give it PICK="<n> <n>" (the lot tips) rather than every commit. Both: PICK="<n> <n>" runs only those positions (1-based); FROM=<n> skips commits before the n-th.
# ONLY=<step[,step]> runs those steps alone (STATIC GEN GOTEST MAKELINT ROOTLINT STORIES; 1 is STORIES); UPD=1 re-records the stories. Default filters: "cli usecases adapters".
# The loop runs ON the VM under setsid (cc-loop.sh): killing this shell, the agent or the background task never kills it. This script copies, launches, then prints the loop's log
# and waits on its end file. When the loop ends it writes $RUN/<label>-cc.end (wait on that file, never on a name). If this shell dies first it writes $RUN/<label>-cc.detached instead:
# ATTACH=1 cc.sh <same arguments> waits on the running loop again, from the start of its log.
# Each commit ends in one line: "VERDICT <label> <n> <sha9> GREEN | FLAKE(<name>) | RED steps=..." (cc-verdict.py: the five known flakes by signature; a flake reruns its step alone once). Then "SUMMARY".
# Files on the VM are ~/$VM_DIR/<label>-* (GATE_VM_DIR); its end file is ~/$VM_DIR/<label>-cc.end. Exit codes: 86 disk, 87 VM not free, 88 loop already runs, 90 bundle, 91 copy.
. "$(dirname "$0")/env.sh"
[ $# -ge 3 ] || { sed -n 2,2p "$0"; exit 2; }
W=$1; BASE=$2; L=$3; shift 3; VM=$VM_LINUX
case $L in "" | *[!A-Za-z0-9._-]*) echo "bad label: $L"; exit 2 ;; esac
LOCK=$VM-$L; R='$HOME'/$VM_DIR; RS=$VM_DIR/
mkdir -p $RUN/vm-lock; o=$(cat $RUN/vm-lock/$LOCK 2>/dev/null)
if [ -n "$o" ] && kill -0 "$o" 2>/dev/null; then echo "refused: $LOCK already runs loop pid $o"; exit 88; fi
done=; launched=
if [ -z "$ATTACH" ]; then rm -f "${RUN:?}/$L-cc.end"; fi
rm -f "${RUN:?}/$L-cc.detached"; echo $$ > $RUN/vm-lock/$LOCK
# invariant: a refused loop (exit 88) never touches <label>-cc.end: that file belongs to the loop already running.
onexit(){ local rc=$?
  if [ -n "$done" ]; then echo "END $(date +%T)" > $RUN/$L-cc.end
  elif [ -n "$launched" ]; then echo "detached $(date +%T): the loop still runs on $VM; ATTACH=1 bash $0 <same arguments> waits on it" > $RUN/$L-cc.detached
  elif [ $rc != 88 ]; then echo "END $(date +%T) not started (exit $rc)" > $RUN/$L-cc.end; fi
  [ "$(cat $RUN/vm-lock/$LOCK 2>/dev/null)" = $$ ] && rm -f $RUN/vm-lock/$LOCK
}
trap onexit EXIT
SSHP="ssh -n -o ConnectTimeout=10 -o ServerAliveInterval=5 -o ServerAliveCountMax=3 -o BatchMode=yes"
B=${BRANCH:-$(git -C "$W" rev-parse --abbrev-ref HEAD)} || exit 2
if [ -z "$ATTACH" ]; then
  # invariant: PICK and FROM are applied here, on the Mac, once: the VM loop gets the exact "<n> <sha>" lines to run.
  i=0; : > $RUN/$L-cc.items
  for sha in $(git -C "$W" rev-list --reverse "$BASE..$B"); do i=$((i+1)); [ "$i" -lt "${FROM:-1}" ] && continue; [ -n "$PICK" ] && ! echo " $PICK " | grep -q " $i " && continue; echo "$i $sha" >> $RUN/$L-cc.items; done
  [ -s $RUN/$L-cc.items ] || { echo "no commit in $BASE..$B"; exit 2; }
  w=0; until ! { [ -e $RUN/vm-hold/$VM ] || [ -e $RUN/vm-hold/all ]; } && ssh -n -o ConnectTimeout=10 $VM true 2>/dev/null; do [ $w -ge 180 ] && { echo "VM $VM not free after 60 min"; exit 87; }; [ $w = 0 ] && echo "waiting for $VM (hold or ssh) $(date +%T)"; w=$((w+1)); sleep 20; done
  if $SSHP $VM "[ -e $R/$L-cc.pid ] && kill -0 \$(cat $R/$L-cc.pid) 2>/dev/null"; then echo "refused: $L already runs on $VM (ATTACH=1 waits on it)"; exit 88; fi
  $SSHP $VM "mkdir -p $R && find $R -maxdepth 1 -name '*.bundle' -mmin +1440 -delete; df -h ~ | tail -1"
  u=$($SSHP $VM "df --output=pcent ~ | tail -1 | tr -dc 0-9"); [ "${u:-100}" -lt 90 ] || { echo "DISK $VM at ${u}%: free it (old *-node, clones, logs) before rerunning"; exit 86; }
  git -C "$W" bundle create $RUN/$L-cc.bundle "$B" --tags >/dev/null 2>&1 || exit 90
  scp -q $RUN/$L-cc.bundle $VM:$RS && scp -q $GATE_DIR/cc-remote.sh $VM:$RS$L-cc-remote.sh && scp -q $GATE_DIR/cc-verdict.py $VM:$RS$L-cc-verdict.py && scp -q $GATE_DIR/cc-loop.sh $VM:$RS$L-cc-loop.sh && scp -q $RUN/$L-cc.items $VM:$RS$L-cc.items || exit 91
  rm -f $RUN/$L-cc.bundle $RUN/$L-cc.items
  if [ $# -eq 0 ]; then FILTERS="cli usecases adapters"; else FILTERS="$*"; fi
  until $SSHP $VM "rm -f $R/$L-cc.log $R/$L-cc.end; CC_HOME=$R CHEAP=$CHEAP ONLY=$ONLY UPD=$UPD PAR=$PAR setsid nohup bash $R/$L-cc-loop.sh $L $B $FILTERS > $R/$L-cc.log 2>&1 < /dev/null & echo launched" | grep -q launched; do fails=$((fails+1)); [ $fails -ge 15 ] && { echo "launch failed on $VM"; exit 91; }; sleep 20; done
  launched=1; echo "launched $L on $VM $(date +%T): CHEAP=${CHEAP:-0} commits=$(echo ${PICK:-all}) log $R/$L-cc.log"
fi
launched=1; off=0; fails=0; polls=0
while :; do
  sleep 20; polls=$((polls+1))
  st=$($SSHP $VM "[ -e $R/$L-cc.end ] && echo END || echo RUN" 2>/dev/null) || st=
  chunk=$($SSHP $VM "tail -c +$((off+1)) $R/$L-cc.log" 2>/dev/null; echo x) && chunk=${chunk%x} || chunk=
  if [ -z "$st" ]; then fails=$((fails+1)); [ $fails -ge 90 ] && { echo "VM $VM unreachable for 30 min: the loop keeps running there (ATTACH=1 waits on it)"; exit 1; }; continue; fi
  fails=0
  [ -n "$chunk" ] && { printf %s "$chunk"; off=$((off+$(printf %s "$chunk" | wc -c))); }
  if [ "$st" = END ]; then
    chunk=$($SSHP $VM "tail -c +$((off+1)) $R/$L-cc.log" 2>/dev/null; echo x) && chunk=${chunk%x} || chunk=
    [ -n "$chunk" ] && printf %s "$chunk"
    break
  fi
  [ $((polls % 15)) = 0 ] && echo "... $L waiting on the VM loop $(date +%T)"
  [ $polls -ge 1440 ] && { echo "loop still running after 8 h: left in place ($R/$L-cc.log)"; exit 1; }
done
done=1
$SSHP $VM "rm -f $R/$L-cc.log $R/$L-cc.end $R/$L-cc-loop.sh"

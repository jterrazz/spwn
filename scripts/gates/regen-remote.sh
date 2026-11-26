#!/bin/bash
export PATH=$HOME/.local/bin:$PATH; eval "$(mise activate bash --shims)"
# invariant: a child never inherits a lock's descriptor (3 to 9: flock's own, the story and cheap slots, the heavy-run lock): an orphaned fixture server that did kept the lock held for 70 min and starved a cold gate (2026-11-21).
nl(){ "$@" 3>&- 4>&- 5>&- 6>&- 7>&- 8>&- 9>&-; }
L=${1:?label}; B=${2:?branch}; ARIA=$3; R=${CC_HOME:?CC_HOME is the folder of the files}; W=${CC_WORK:?CC_WORK is the folder of the clones}; RUNTMP=/tmp/sp-$L-regen-$$
trap 'cd ~; rm -rf "${RUNTMP:?}"' EXIT
bad=; st() { echo "$1=$2"; [ "$2" = 0 ] || bad=1; }
~/.local/bin/heavy-run-wait; exec 9>/var/lock/heavy-run; flock -w 7200 -s 9 || exit 85
export TMPDIR=$RUNTMP; mkdir -p "$TMPDIR"
# invariant: GEN through STORIES (and only those) sit behind a VM-wide slot (2 holders: /tmp/story-slot-{1,2}, flock on fd 8), taken once and held to the end of STORIES.
take_slot(){ [ -n "$got" ] && return 0; local P=${SLOT_POLL:-20} w=0 k
  while :; do
    for k in 1 2; do exec 8>/tmp/story-slot-$k; flock -n 8 && { got=$k; echo "STORY-SLOT $got taken"; return 0; }; exec 8>&-; done
    [ $w -ge $((3600/P)) ] && { echo "STORY-SLOT: both held for 60 min"; exit 84; }
    [ $((w % (60/P))) = 0 ] && echo "STORY-SLOT: both held, waiting $(date +%T)"
    w=$((w+1)); command sleep $P
  done
}; got=
mkdir -p "$W"; df -h ~ "$W" /tmp | tail -n +2
rm -rf "${W:?}/$L-regen-node"; git clone -q -b $B $R/$L-regen.bundle $W/$L-regen-node && cd $W/$L-regen-node/machines || exit 9
npm ci >$R/$L-regen-npmci.log 2>&1; st NPMCI $?
app=$(ls -d apps/desktop apps/window 2>/dev/null | head -1); (cd $app && npx playwright install chromium-headless-shell >/dev/null 2>&1)
mkdir -p .artifacts/go; go work sync
take_slot
for i in 1 2; do
  WINDOW_RECORD_FORCE=1 nl make generate > $R/$L-regen$i.log 2>&1; st GEN$i $?
  (cd apps/engine && nl go test ./ui/render -update >> $R/$L-regen$i.log 2>&1); st RENDER$i $?
  git add -A . >/dev/null 2>&1; git diff --cached --binary | sha256sum > $R/$L-regen-r$i.sum
done
cmp $R/$L-regen-r1.sum $R/$L-regen-r2.sum && echo DETERMINISTIC || { echo NONDETERMINISTIC; bad=1; }
exec 8>&-
[ -n "$ARIA" ] && { (cd apps/desktop && nl timeout 1200 npx vitest run -u --maxWorkers=2 $ARIA > $R/$L-regen-aria.log 2>&1); st ARIA $?; tail -5 $R/$L-regen-aria.log; git add -A . >/dev/null 2>&1; }
git diff --cached --name-status > $R/$L-regen-changes.txt; head -60 $R/$L-regen-changes.txt
cd .. && git diff --cached --name-only --diff-filter=AM | tar -cf $R/$L-regen-r.tar -T -
[ -z "$bad" ]

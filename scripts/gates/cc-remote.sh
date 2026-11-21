#!/bin/bash
# usage: cc-remote.sh <label> <branch> <sha> <n> [vitest filters...]  the per-commit checks of ONE commit, on the VM (cc.sh and cc-loop.sh launch it)
# CHEAP=1: the cheap tier (gofmt, vet x3, golangci, make lint, the world build); no story slot, up to 4 at once (/tmp/cheap-slot-1..4).
# ONLY=<step[,step]>: those steps alone (STATIC GEN GOTEST MAKELINT STORIES; DRIFT is GEN; 1 is STORIES). UPD=1 re-records the stories.
# CC_HOME: the folder its files go in (default ~). CC_SHARED=1 (the shared VM): holds /var/lock/heavy-run shared for the commit, a TMPDIR of its own per run,
# and the stories in their own network namespace (netns-run), so two runs never share a port, a clone or a temp folder.
export PATH=$HOME/.local/bin:$PATH; eval "$(mise activate bash --shims)"
L=$1; B=$2; sha=$3; n=$4; shift 4
R=${CC_HOME:-$HOME}; NS=; RUNTMP=
# invariant: a child never inherits a lock's descriptor (3 to 9: flock's own, the story and cheap slots, the heavy-run lock): an orphaned fixture server that did kept the lock held for 70 min and starved a cold gate (2026-11-21).
nl(){ "$@" 3>&- 4>&- 5>&- 6>&- 7>&- 8>&- 9>&-; }
WT=
trap 'cd ~; [ -n "$WT" ] && rm -rf "${WT:?}"; [ -n "$RUNTMP" ] && rm -rf "${RUNTMP:?}"' EXIT
if [ -n "$CC_SHARED" ]; then
  $HOME/.local/bin/heavy-run-wait; exec 9>/var/lock/heavy-run; flock -w 3600 -s 9 || { echo "HEAVY-RUN lock not shared within 60 min"; exit 85; }
  RUNTMP=/tmp/sp-$L-$$; export TMPDIR=$RUNTMP; mkdir -p "$TMPDIR"; NS=netns-run
fi
ONLYS=,${ONLY//DRIFT/GEN},; ONLYS=${ONLYS/,1,/,STORIES,}; [ "$ONLYS" = ,, ] && ONLYS=
want(){ [ -z "$ONLYS" ] && return 0; case "$ONLYS" in *",$1,"*) return 0 ;; esac; return 1; }
tier(){ [ -z "$CHEAP" ] && return 0; case $1 in STATIC | MAKELINT | BUILD) return 0 ;; esac; return 1; }
run(){ want "$1" && tier "$1"; }
take_cheap(){ local w=0 k
  while :; do
    for k in 1 2 3 4; do exec 7>/tmp/cheap-slot-$k; flock -n 7 && { CK=$k; echo "CHEAP-SLOT $k taken"; return 0; }; exec 7>&-; done
    [ $w -ge 180 ] && { echo "CHEAP-SLOT: all 4 held for 60 min"; exit 83; }
    [ $((w % 3)) = 0 ] && echo "CHEAP-SLOT: all 4 held, waiting $(date +%T)"
    w=$((w+1)); command sleep 20
  done
}
# GEN through STORIES (and only those) sit behind a VM-wide slot (2 holders: /tmp/story-slot-{1,2}, flock on fd 8): taken once, held to the end of STORIES; else wait (<= 60 min, a line a minute).
take_slot(){ [ -n "$got" ] && return 0; local P=${SLOT_POLL:-20} w=0 k
  while :; do
    for k in 1 2; do exec 8>/tmp/story-slot-$k; flock -n 8 && { got=$k; echo "STORY-SLOT $got taken"; return 0; }; exec 8>&-; done
    [ $w -ge $((3600/P)) ] && { echo "STORY-SLOT: both held for 60 min"; exit 84; }
    [ $((w % (60/P))) = 0 ] && echo "STORY-SLOT: both held, waiting $(date +%T)"
    w=$((w+1)); command sleep $P
  done
}; got=
df -h ~ | tail -1
[ -n "$CHEAP" ] && take_cheap
# STORIES_WORKERS: vitest workers of the stories (default 8; 12 is for an exclusive gate).
# The per-label node clone is built once, under a lock: runs of one label in parallel wait for it instead of racing on it.
exec 6>$R/$L-node.lock; flock 6
if [ ! -d $R/$L-node/machines/node_modules ]; then
  rm -rf "${R:?}/$L-node"; git clone -q -b $B $R/$L-cc.bundle $R/$L-node && cd $R/$L-node && git reset -q --hard $sha && cd machines && npm ci >$R/$L-npmci.log 2>&1; echo NPMCI=$?
  (cd $R/$L-node/machines && cd "$(ls -d apps/desktop apps/window 2>/dev/null | head -1)" && npx playwright install chromium-headless-shell >/dev/null 2>&1)
  cd ~
fi
flock -u 6
# a lane clones to a fixed path, so Go's build cache (keyed by the directory) hits on the next commit: <label>-cc-wt; under CHEAP, one per cheap slot. Two loops of a label never meet (cc-loop.sh refuses the second).
WT=$R/$L-cc-wt${CK:+-s$CK}; rm -rf "${WT:?}"
git clone -q -b $B $R/$L-cc.bundle "$WT" && cd "$WT" && git reset -q --hard $sha && git log -1 --oneline && cd machines && cp -al $R/$L-node/machines/node_modules . || exit 9
if run STATIC; then
echo GOFMT[$(gofmt -l apps/engine catalog specs)]
TIMEFORMAT='VET-CPU user %U sys %S real %R'
time { (cd apps/engine && go vet ./... && go vet -tags world ./... && echo VET-OK)
(cd specs && go vet ./... && echo SPECSVET-OK); }
GOOS=linux $(go tool -n golangci-lint) run --allow-parallel-runners ./apps/engine/... ./catalog/... ./specs/... > $R/$L-lint$n.log 2>&1; echo LINT=$?; grep -E "\.go:[0-9]+|issues" $R/$L-lint$n.log | head -20
fi
if run GEN; then
take_slot
mkdir -p .artifacts/go
STATION_BENCH=${STATION_BENCH:-$HOME/bench} nl make generate >$R/$L-gen$n.log 2>&1; echo GEN=$?; echo "DRIFT[$(git status --porcelain | tr '\n' ' ')]"; [ -n "$(git status --porcelain)" ] && git diff > $R/$L-drift$n.patch && echo "drift diff kept: $R/$L-drift$n.patch"
fi
if run GOTEST; then
take_slot
(cd apps/engine && nl timeout 40m go test -p 4 -count=1 -timeout 20m ./... ../../specs/architecture/ > $R/$L-gotest$n.log 2>&1; echo GOTEST=$?; grep -E "^(--- FAIL|FAIL|panic)" $R/$L-gotest$n.log | head -20)
fi
if run MAKELINT; then nl make lint >$R/$L-makelint$n.log 2>&1; echo MAKELINT=$?; sed 's/\x1b\[[0-9;]*m//g' $R/$L-makelint$n.log | grep -vE '^\s*$' | tail -25; fi
if { want STORIES || want BUILD; } && { [ -z "$CHEAP" ] || run BUILD; }; then
mkdir -p .artifacts/go
go build -tags world -ldflags "-X github.com/jterrazz/studio-station/apps/engine/wire.version=$(git describe --tags --always --dirty --long)" -o .artifacts/go/station-test ./apps/engine/cmd/station && touch .artifacts/go/station-test; echo BUILD=$?
fi
if run STORIES; then
take_slot
nl $NS env ${UPD:+TEST_UPDATE=1} STATION_SPEC_BUILT=1 ${TMPDIR:+TMPDIR=$TMPDIR} timeout 60m npm --workspace specs exec -- vitest --run --maxWorkers=${STORIES_WORKERS:-8} "${@:-cli}" > $R/$L-stories$n.log 2>&1; echo STORIES=$?; exec 8>&-
sed 's/\x1b\[[0-9;]*m//g' $R/$L-stories$n.log | grep -E "^ +(×|❯) |Test Files|Tests |FAIL|Error" | cut -c1-220 | head -40
fi
echo CC-DONE $sha
exit 0

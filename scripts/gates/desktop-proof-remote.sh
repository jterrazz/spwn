#!/bin/bash
# usage (on the VM, copied and run by desktop-proof.sh): desktop-proof-remote.sh <label> <branch>: desktop tsc, vitest (2 workers), proof build, website, typescript check, architecture suite and make lint at the branch tip, own clone, deleted at the end
export PATH=$HOME/.local/bin:$PATH; eval "$(mise activate bash --shims)"
# invariant: a child never inherits a lock's descriptor (3 to 9: flock's own, the story and cheap slots, the heavy-run lock): an orphaned fixture server that did kept the lock held for 70 min and starved a cold gate (2026-11-21).
nl(){ "$@" 3>&- 4>&- 5>&- 6>&- 7>&- 8>&- 9>&-; }
R=${CC_HOME:?CC_HOME is the folder of the files}; L=${1:?label}; B=${2:?branch}; C=$R/$L-wt; cd $R || exit 2
echo "waiting for lock $(date +%T)"; ~/.local/bin/heavy-run-wait; exec 9>/var/lock/heavy-run; flock -w 7200 -s 9 || { echo LOCK-TIMEOUT; exit 85; }
export TMPDIR=$(mktemp -d /tmp/sp-$L-XXXXXX); df -h ~ | tail -1; uptime
rm -rf "${C:?}" && git clone -q -b $B $R/$L.bundle $C && cd $C && git log -1 --oneline && cd machines || exit 9
npm ci > $R/$L-npmci.log 2>&1; echo NPMCI=$?
cd apps/desktop
timeout 900 npx tsc --noEmit -p . > $R/$L-tsc.log 2>&1; echo TSC=$?; grep -E "error TS" $R/$L-tsc.log | head -40
nl timeout 2400 npx vitest run --maxWorkers=4 > $R/$L-vitest.log 2>&1; echo VITEST=$?
sed 's/\x1b\[[0-9;]*m//g' $R/$L-vitest.log | grep -E "^ +(×|❯) | FAIL |Test Files|Tests " | cut -c1-260 | head -60
cd $C/machines && (cd apps/window 2>/dev/null && npx playwright install chromium-headless-shell >/dev/null 2>&1); (cd apps/desktop && npx playwright install chromium-headless-shell >/dev/null 2>&1)
nl timeout 1200 npx turbo run build:proof --filter=station-desktop > $R/$L-proof.log 2>&1; echo PROOF=$?
(cd specs && nl timeout 2400 npx vitest run --project website --maxWorkers=4 > $R/$L-web.log 2>&1; echo WEBSITE=$?)
sed 's/\x1b\[[0-9;]*m//g' $R/$L-web.log | grep -E "^ +(×|❯) | FAIL |Test Files|Tests " | cut -c1-260 | head -40
cd $C/machines && nl timeout 1800 npx typescript check > $R/$L-tscheck.log 2>&1; echo TSCHECK=$?; sed 's/\x1b\[[0-9;]*m//g' $R/$L-tscheck.log | grep -vE '^\s*$' | tail -30
cd $C/machines && (cd specs && nl timeout 1800 go test ./architecture/ > $R/$L-arch.log 2>&1; echo ARCH=$?); grep -E "^(--- FAIL|FAIL|ok)|retired" $R/$L-arch.log | head -20
cd $C/machines && nl timeout 1800 make lint > $R/$L-lint.log 2>&1; echo MAKELINT=$?; sed 's/\x1b\[[0-9;]*m//g' $R/$L-lint.log | grep -E "issues|error|Error|✗" | head -20
cd $R; rm -rf "${C:?}" "${R:?}/$L.bundle" "${TMPDIR:?}"
echo DT-DONE

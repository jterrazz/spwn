#!/usr/bin/env bash
# outside: the one bench script run from the Mac: it copies and asks, and never installs, builds nor tests there.
# value: look.sh <lot> <test path> [--update | --new]: the tree synced to the VM's window-<lot>/repo, the one spec file
# value: or folder run there (its goldens rewritten with --update), then every golden of src/ the run changed copied
# value: back to the Mac, named one per line, to be looked at before a commit. `--new` runs in update mode too, but
# value: copies back only the goldens the Mac does not have yet, so a new test's golden is born and no other moves.
# value: STATION_VM names the host (design).
# hazard: vitest's -u takes the next word as its own value, so the path is given before it.
set -euo pipefail
[ $# -ge 2 ] || {
    echo "usage: look.sh <lot> <test path, from apps/window> [--update | --new]" >&2
    exit 2
}
lot=$1
path=$2
update=
only_new=()
case ${3:-} in
    '') ;;
    --update) update=-u ;;
    --new)
        update=-u
        only_new=(--ignore-existing)
        ;;
    *)
        echo "usage: look.sh <lot> <test path, from apps/window> [--update | --new]" >&2
        exit 2
        ;;
esac
host=${STATION_VM:-design}
window=$(cd "$(dirname "$0")/.." && pwd)
root=$(cd "$window/../.." && pwd)
copy=window-$lot/repo
ssh -o BatchMode=yes "$host" "mkdir -p $copy"
rsync -a --delete --exclude .git --exclude node_modules --exclude target --exclude .artifacts "$root/" "$host:$copy/"
status=0
ssh -o BatchMode=yes "$host" "set -o pipefail; cd $copy/apps/window && source scripts/bench-env.sh \
    && bash scripts/bench-setup.sh >/dev/null \
    && timeout 520 npx vitest --run --maxWorkers=${WINDOW_TEST_WORKERS:-2} '$path' $update 2>&1 | tail -40" || status=$?
echo "goldens changed, to look at:"
rsync -a ${only_new[@]+"${only_new[@]}"} --out-format 'apps/window/src/%n' --include '*/' --include '__expected__/**' --exclude '*' --prune-empty-dirs \
    "$host:$copy/apps/window/src/" "$window/src/"
exit "$status"

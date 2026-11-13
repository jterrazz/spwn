#!/usr/bin/env bash
# outside: a Linux VM only.
# value: `typescript fix` from the root, after bench-setup.sh. It exits 0 when apps/window holds no finding the fix left; the root's findings are counted apart.
set -uo pipefail
# shellcheck source=bench-env.sh
source "$(dirname "$0")/bench-env.sh"
# shellcheck source=findings.sh
source "$scripts/findings.sh"
bash "$scripts/bench-setup.sh" >/dev/null || exit 1
cd "$root"
logs=$window/.artifacts/bench
mkdir -p "$logs"
LD_PRELOAD=$NORESERVE timeout 520 npx typescript fix >"$logs/fix.log" 2>&1
fixed=$?
summary "$logs/fix.log"
status=$?
[ "$fixed" -ne 124 ] || status=1
echo "fix: window exit $status · root $root_findings findings, typescript fix exit $fixed · log in $logs/fix.log"
exit "$status"

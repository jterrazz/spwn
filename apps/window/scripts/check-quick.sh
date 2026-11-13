#!/usr/bin/env bash
# outside: a Linux VM only.
# value: after bench-setup.sh, the three passes of `typescript check` that read one file at a time, scoped to
# value: apps/window: the native compiler over its project, oxlint as the check runs it (type-aware, warnings
# value: denied) and oxfmt. The tree's passes (knip, names, architecture, test conventions, comments, markdown,
# value: secrets) judge the whole repository and only check.sh runs them. Exit 0 when apps/window has no finding.
set -uo pipefail
# shellcheck source=bench-env.sh
source "$(dirname "$0")/bench-env.sh"
# shellcheck source=findings.sh
source "$scripts/findings.sh"
bash "$scripts/bench-setup.sh" >/dev/null || exit 1
cd "$root"
logs=$window/.artifacts/bench
mkdir -p "$logs"
tools=$root/node_modules/.bin
start=$(date +%s)
(npx typescript tsc --noEmit --project apps/window >"$logs/quick-tsc.log" 2>&1; echo "exit $?" >>"$logs/quick-tsc.log") &
mapfile -t files < <(find apps/window/src apps/window/scripts apps/window/*.ts -type f \( -name '*.ts' -o -name '*.tsx' \))
LD_PRELOAD=$NORESERVE "$tools/oxlint" --type-aware --deny-warnings "${files[@]}" >"$logs/quick-lint.log" 2>&1 &
LD_PRELOAD=$NORESERVE "$tools/oxfmt" --check apps/window >"$logs/quick-format.log" 2>&1
format=$?
wait
summary "$logs"/quick-tsc.log "$logs"/quick-lint.log
status=$?
if [ "$format" -ne 0 ]; then
    echo "format: run scripts/fix.sh, or oxfmt on these files:"
    sed 's/\x1b\[[0-9;]*m//g' "$logs/quick-format.log" | grep -aE '^apps/window' | sed 's/^/  /'
    status=1
fi
grep -q '^exit 0$' "$logs/quick-tsc.log" || status=1
echo "check-quick: exit $status in $(($(date +%s) - start)) s, logs in $logs"
exit "$status"

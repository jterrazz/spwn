#!/usr/bin/env bash

set -euo pipefail

# hazard: actool's Icon Composer path fails with "Bad file descriptor" and writes nothing when Tauri's
# bundler calls it, where it compiles from a shell, so each app's build calls this script itself, and
# only outputs that are all present pass.
usage='usage: app-icon-compile.sh <document.icon> <icon name> <out dir> <partial plist>'
document=${1:?$usage}
name=${2:?$usage}
out=${3:?$usage}
plist=${4:?$usage}
[ -d "$document" ] || {
    printf 'app-icon-compile: %s is not an Icon Composer document\n' "$document" >&2
    exit 1
}
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
cp -R "$document" "$work/$name.icon"
mkdir -p "$out" "$(dirname "$plist")"
rm -f "$out/Assets.car" "$out/$name.icns" "$plist"
xcrun actool "$work/$name.icon" --compile "$out" --app-icon "$name" --include-all-app-icons \
    --target-device mac --minimum-deployment-target 26.0 --platform macosx \
    --output-partial-info-plist "$plist" >"$work/actool.log" 2>&1 || true
if [ -s "$out/Assets.car" ] && [ -s "$out/$name.icns" ] &&
    [ "$(plutil -extract CFBundleIconName raw "$plist" 2>/dev/null)" = "$name" ]; then
    printf 'app-icon-compile: %s compiled into %s\n' "$name" "$out"
    exit 0
fi
reason=$(grep -A1 -E '<key>(description|failure-reason)</key>' "$work/actool.log" | sed -n 's/.*<string>\(.*\)<\/string>.*/\1/p' | paste -sd ' ' - || true)
printf 'app-icon-compile: actool wrote no Assets.car, %s.icns and CFBundleIconName %s: %s\n' "$name" "$name" "${reason:-no error text}" >&2
exit 1

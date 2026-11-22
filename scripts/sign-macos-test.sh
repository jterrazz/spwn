#!/usr/bin/env bash

set -euo pipefail

sign="$(cd "$(dirname "$0")" && pwd)/sign-macos.sh"
work=$(mktemp -d "${TMPDIR:-/tmp}/sign-macos-test.XXXXXX")
trap 'rm -rf "$work"' EXIT
export APPLE_SIGNING_IDENTITY=-
app="$work/Spwn.app"
menubar="$app/Contents/Library/LoginItems/Station.app"

fail() {
    printf 'FAIL sign-macos-test: %s\n' "$*" >&2
    exit 1
}

printf 'int main(void) { return 0; }\n' > "$work/main.c"
clang -o "$work/exe" "$work/main.c"

bundle() {
    local path=$1 identifier=$2 executable=$3
    mkdir -p "$path/Contents/MacOS"
    cp "$work/exe" "$path/Contents/MacOS/$executable"
    plutil -create xml1 "$path/Contents/Info.plist"
    plutil -insert CFBundleIdentifier -string "$identifier" "$path/Contents/Info.plist"
    plutil -insert CFBundleExecutable -string "$executable" "$path/Contents/Info.plist"
    plutil -insert CFBundlePackageType -string APPL "$path/Contents/Info.plist"
}

layout() {
    rm -rf "$app"
    bundle "$app" com.jterrazz.spwn spwn-window
    cp "$work/exe" "$app/Contents/MacOS/Spwn Engine"
    bundle "$menubar" com.jterrazz.station.menubar StationMenuBar
    codesign --force --sign - --timestamp=none "$menubar"
}

refused() {
    local why=$1 out
    if out=$("$sign" verify "$app" 2>&1); then
        fail "verify accepts a bundle where $why"
    fi
    grep -qF "$2" <<< "$out" || fail "verify refuses a bundle where $why, but not for that: $out"
    printf 'PASS verify refuses a bundle where %s\n' "$why"
}

layout
codesign --force --sign - --options runtime --timestamp=none "$app/Contents/MacOS/Spwn Engine"
codesign --force --sign - --options runtime --timestamp=none "$app"
refused "the login item is signed as its own build signs it, without the hardened runtime" \
    "$menubar/Contents/MacOS/StationMenuBar is signed without the hardened runtime"

layout
codesign --force --sign - --options runtime --timestamp=none "$menubar"
codesign --force --sign - --options runtime --timestamp=none "$app"
refused "the engine keeps the signature its linker wrote" \
    "$app/Contents/MacOS/Spwn Engine is signed without the hardened runtime"

layout
out=$("$sign" sign "$app" 2>&1) || fail "sign fails on the bundle: $out"
grep -qF "verify: $app and every Mach-O in it are signed with the hardened runtime, by one team" <<< "$out" ||
    fail "sign does not verify the bundle it signed: $out"
printf 'PASS sign seals the bundle inside out, and verify accepts it\n'

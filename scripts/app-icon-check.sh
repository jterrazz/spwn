#!/usr/bin/env bash

set -euo pipefail

# hazard: a bundle passes only if macOS can draw its icon in every appearance: Tauri skips
# the asset catalog with a logged error and still exits 0 when actool is missing or older than
# 26, and actool compiles a document it half reads, so the check reads what the bundle holds.
app=${1:?usage: app-icon-check.sh <bundle.app>}
fail() {
    printf '%s: %s\n' "$(basename "$app")" "$1" >&2
    exit 1
}
plist="$app/Contents/Info.plist"
car="$app/Contents/Resources/Assets.car"
name=$(plutil -extract CFBundleIconName raw "$plist" 2>/dev/null || true)
file=$(plutil -extract CFBundleIconFile raw "$plist" 2>/dev/null || true)
[ -n "$name" ] || fail "no CFBundleIconName in Info.plist: the icon is a flat bitmap in every appearance"
[ -f "$car" ] || fail "no Contents/Resources/Assets.car: the icon document was not compiled"
info=$(xcrun assetutil --info "$car")
grep -q "\"Name\" : \"$name\"" <<<"$info" || fail "Assets.car holds no icon named $name"
for appearance in NSAppearanceNameAqua NSAppearanceNameDarkAqua ISAppearanceTintable; do
    grep -q "\"Appearance\" : \"$appearance\"" <<<"$info" || fail "Assets.car holds no $appearance rendition"
done
[ -n "$file" ] && [ -f "$app/Contents/Resources/${file%.icns}.icns" ] || fail "CFBundleIconFile names no .icns in Contents/Resources"
printf 'icon %s: Assets.car (light, dark, tintable) and %s.icns\n' "$name" "${file%.icns}"

#!/bin/sh
set -eu
root=$(cd "$(dirname "$0")/../../.." && pwd)
package="$root/apps/menubar"
app="$root/.artifacts/menubar/Station.app"
scratch="${STATION_MENUBAR_SCRATCH:-$root/.artifacts/menubar/build}"
[ "${STATION_SYMBOLS_DRAWN:-}" = 1 ] || swift "$package/scripts/symbols.swift" "$package/sources/station-kit/token/symbols.xcassets"
swift build -c release --package-path "$package" --scratch-path "$scratch" --product StationMenuBar
bin=$(swift build -c release --package-path "$package" --scratch-path "$scratch" --show-bin-path)
rm -rf "$app"
mkdir -p "$app/Contents/MacOS" "$app/Contents/Resources"
cp "$bin/StationMenuBar" "$app/Contents/MacOS/StationMenuBar"
cp "$package/resources/Info.plist" "$app/Contents/Info.plist"
cp -R "$bin/StationMenuBar_StationKit.bundle" "$app/Contents/Resources/"
"$root/../scripts/app-icon-compile.sh" "$package/resources/app-icon.icon" AppIcon "$app/Contents/Resources" "$scratch/AppIcon.plist"
/usr/libexec/PlistBuddy -c "Merge $scratch/AppIcon.plist" "$app/Contents/Info.plist" >/dev/null
version=$(git -C "$root" describe --tags --always --dirty | sed 's/^v//')
plutil -replace CFBundleShortVersionString -string "$version" "$app/Contents/Info.plist"
cp "$root/apps/engine/ui/testdata/pages/menubar.home.engine-stopped.json" "$app/Contents/Resources/stopped.json"
cp "$root/apps/engine/ui/testdata/pages/menubar.home.engine-outdated.json" "$app/Contents/Resources/outdated.json"
codesign --force --sign - --timestamp=none "$app"
codesign --verify --strict "$app"
"$root/../scripts/app-icon-check.sh" "$app"
printf '● Built %s (%s, signed ad hoc)\n' ".artifacts/menubar/Station.app" "$version"

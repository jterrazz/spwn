#!/bin/sh
# value: with no argument, .artifacts/menubar/Station.app, the menu bar on its own, signed ad hoc; `library`, the
# value: panel's library Spwn.app links, with the resources it reads, into .artifacts/window/menubar/.
set -eu
root=$(cd "$(dirname "$0")/../../.." && pwd)
package="$root/apps/menubar"
app="$root/.artifacts/menubar/Station.app"
scratch="${STATION_MENUBAR_SCRATCH:-$root/.artifacts/menubar/build}"
mode=${1:-app}
case $mode in app | library) ;; *) echo "usage: build.sh [library]" >&2; exit 2 ;; esac
[ "${STATION_SYMBOLS_DRAWN:-}" = 1 ] || swift "$package/scripts/symbols.swift" "$package/sources/station-kit/token/symbols.xcassets"
if [ "$mode" = library ]; then
  swift build -c release --package-path "$package" --scratch-path "$scratch" --product SpwnMenuBar
  bin=$(swift build -c release --package-path "$package" --scratch-path "$scratch" --show-bin-path)
  out="$root/.artifacts/window/menubar"
  rm -rf "$out"
  mkdir -p "$out"
  cp "$bin/libSpwnMenuBar.dylib" "$out/libSpwnMenuBar.dylib"
  cp -R "$bin/StationMenuBar_StationKit.bundle" "$out/"
  cp "$root/apps/engine/ui/testdata/pages/menubar.home.engine-stopped.json" "$out/stopped.json"
  cp "$root/apps/engine/ui/testdata/pages/menubar.home.engine-outdated.json" "$out/outdated.json"
  printf '● Built %s\n' ".artifacts/window/menubar/libSpwnMenuBar.dylib"
  exit 0
fi
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

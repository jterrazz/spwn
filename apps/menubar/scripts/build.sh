#!/bin/sh
set -eu
root=$(cd "$(dirname "$0")/../../.." && pwd)
package="$root/apps/menubar"
app="$root/.artifacts/menubar/Station.app"
scratch="$root/.artifacts/menubar/build"
swift "$package/scripts/symbols.swift" "$package/sources/station-kit/token/symbols.xcassets"
swift build -c release --package-path "$package" --scratch-path "$scratch" --product StationMenuBar
bin=$(swift build -c release --package-path "$package" --scratch-path "$scratch" --show-bin-path)
rm -rf "$app"
mkdir -p "$app/Contents/MacOS" "$app/Contents/Resources"
cp "$bin/StationMenuBar" "$app/Contents/MacOS/StationMenuBar"
cp "$package/resources/Info.plist" "$app/Contents/Info.plist"
cp -R "$bin/StationMenuBar_StationKit.bundle" "$app/Contents/Resources/"
icons="$scratch/AppIcon.iconset"
rm -rf "$icons" && mkdir -p "$icons"
qlmanage -t -s 1024 -o "$scratch" "$package/resources/AppIcon.svg" >/dev/null
for size in 16 32 128 256 512; do
    sips -z $size $size "$scratch/AppIcon.svg.png" --out "$icons/icon_${size}x${size}.png" >/dev/null
    sips -z $((size * 2)) $((size * 2)) "$scratch/AppIcon.svg.png" --out "$icons/icon_${size}x${size}@2x.png" >/dev/null
done
iconutil -c icns "$icons" -o "$app/Contents/Resources/AppIcon.icns"
version=$(git -C "$root" describe --tags --always --dirty | sed 's/^v//')
plutil -replace CFBundleShortVersionString -string "$version" "$app/Contents/Info.plist"
cp "$root/apps/engine/ui/testdata/pages/menubar.home.engine-stopped.json" "$app/Contents/Resources/stopped.json"
cp "$root/apps/engine/ui/testdata/pages/menubar.home.engine-outdated.json" "$app/Contents/Resources/outdated.json"
codesign --force --sign - --timestamp=none "$app"
codesign --verify --strict "$app"
printf '● Built %s (%s, signed ad hoc)\n' ".artifacts/menubar/Station.app" "$version"

#!/usr/bin/env bash
# outside: Xvfb, openbox and xdotool exist in the Linux VM alone; a Mac never opens this window.
# value: usage: capture-tauri.sh [<out-dir> [<world>]]: .artifacts/captures/tauri and stories by default. It opens
# value: the window on the world (tauri-session.sh) and captures Home, then the first machine of the sidebar.
set -euo pipefail
# shellcheck source=bench-env.sh
source "$(dirname "$0")/bench-env.sh"
# shellcheck source=tauri-session.sh
source "$scripts/tauri-session.sh"
cd "$window"
out=$(realpath -m "${1:-.artifacts/captures/tauri}")
world=${2:-stories}
mkdir -p "$out"
tauri_open "$world"
import -window "$win" "$out/tauri-$world-home.png"
echo "captured $out/tauri-$world-home.png"
xdotool mousemove --window "$win" 110 250 click 1
sleep 6
import -window "$win" "$out/tauri-$world-machine.png"
echo "captured $out/tauri-$world-machine.png"

#!/usr/bin/env bash
# outside: Xvfb, openbox and xdotool exist in the Linux VM alone; a Mac never opens this window, and the world build
# outside: is the only engine it ever reaches: every gesture it makes changes the fake world serve-world.py serves.
# value: usage: live-tauri.sh <steps> <out-dir> [<world>]: opens the window on the world (stories by default, at
# value: 1280 x 820) and plays <steps>, one per line: `click <x> <y>` in the window, `menu <x> <y>` its context menu,
# value: `point <x> <y>` the pointer moved there with no click (away from a row, so no tooltip is in a capture),
# value: `type <text>`, `wait <seconds>`, `scroll <x> <y> <clicks>` (the wheel at that point, down, up when <clicks> is
# value: negative), `see <seconds> <text>` (waits until <text> is read on the window by OCR, case ignored, and fails
# value: the run past <seconds> with the window captured as <out-dir>/unseen.png and what was read as unseen.txt),
# value: `shot <name>` to <out-dir>/<name>.png, `key <keys>` for xdotool, `engine stop` and `engine start` to end the
# value: world's engine and serve it again on the same socket, `ask <call> [<input as JSON>]` one call sent to that
# value: engine beside the window, as another client would (ask-engine.py), `probe <name>` the window's threads and sockets and the
# value: connections the engine holds to <out-dir>/<name>.txt, `relaunch` to end the window and open it again in
# value: the same sandbox HOME. A line opening on # is skipped.
set -euo pipefail
# shellcheck source=bench-env.sh
source "$(dirname "$0")/bench-env.sh"
# shellcheck source=tauri-session.sh
source "$scripts/tauri-session.sh"
live_engine() {
    if [ "$1" = stop ]; then
        kill "$bridge" && wait "$bridge" 2>/dev/null || true
        rm -f "$sandbox/engine.sock"
    else
        python3 "$scripts/serve-world.py" "$2" "$sandbox/engine.sock" >>"$window/.artifacts/tauri-bridge.log" 2>&1 &
        bridge=$!
        for _ in $(seq 1 60); do
            [ -S "$sandbox/engine.sock" ] && break
            sleep 0.5
        done
    fi
}
live_probe() {
    echo "threads=$(ls "/proc/$app/task" | wc -l)"
    echo "sockets=$(find "/proc/$app/fd" -lname 'socket:*' 2>/dev/null | wc -l)"
    echo "engine-connections=$(($(find "/proc/$bridge/fd" -lname 'socket:*' 2>/dev/null | wc -l) - 1))"
}
live_scroll() {
    local x=$1 y=$2 clicks=$3 button=5
    if [ "$clicks" -lt 0 ]; then
        button=4 clicks=$((-clicks))
    fi
    xdotool mousemove --window "$win" "$x" "$y" click --repeat "$clicks" --delay 60 "$button"
}
live_read() {
    import -window "$win" "$out/.seen.png" </dev/null
    convert "$out/.seen.png" -colorspace Gray -negate -resize 200% -level 0%,70% "$out/.read.png" </dev/null
    tesseract "$out/.read.png" stdout </dev/null 2>/dev/null | tr -s ' \n' '  '
    rm -f "$out/.read.png"
}
live_see() {
    local seconds=$1 text=$2 said=''
    command -v tesseract >/dev/null || { echo "see needs tesseract (apt install tesseract-ocr)" >&2 && exit 2; }
    local deadline=$((SECONDS + seconds))
    while [ "$SECONDS" -le "$deadline" ]; do
        said=$(live_read)
        if grep -qiF -- "$text" <<<"$said"; then
            rm -f "$out/.seen.png"
            echo "saw \"$text\""
            return
        fi
        sleep 1
    done
    mv "$out/.seen.png" "$out/unseen.png" 2>/dev/null || true
    printf '%s\n' "$said" >"$out/unseen.txt"
    echo "\"$text\" not seen in ${seconds} s: $out/unseen.png, $out/unseen.txt" >&2
    exit 3
}
steps=$(realpath "$1")
out=$(realpath -m "$2")
cd "$window"
mkdir -p "$out"
export TAURI_WIDTH=1280 TAURI_HEIGHT=820
tauri_open "${3:-stories}"
while read -r verb rest; do
    case "$verb" in
        click) read -r x y <<<"$rest" && xdotool mousemove --window "$win" "$x" "$y" click 1 ;;
        point) read -r x y <<<"$rest" && xdotool mousemove --window "$win" "$x" "$y" ;;
        menu) read -r x y <<<"$rest" && xdotool mousemove --window "$win" "$x" "$y" click 3 ;;
        type) xdotool windowfocus --sync "$win" type --delay 40 "$rest" ;;
        key) xdotool windowfocus --sync "$win" key "$rest" ;;
        engine) live_engine "$rest" "${3:-stories}" ;;
        ask) read -r call given <<<"$rest" && python3 "$scripts/ask-engine.py" "$sandbox/engine.sock" "$call" ${given:+"$given"} ;;
        probe) live_probe >"$out/$rest.txt" && echo "probed $out/$rest.txt: $(tr '\n' ' ' <"$out/$rest.txt")" ;;
        relaunch) tauri_relaunch ;;
        wait) sleep "$rest" ;;
        scroll) read -r x y clicks <<<"$rest" && live_scroll "$x" "$y" "$clicks" ;;
        see) read -r seconds text <<<"$rest" && live_see "$seconds" "$text" ;;
        shot) import -window "$win" "$out/$rest.png" && echo "captured $out/$rest.png" ;;
        '' | '#'*) ;;
        *) echo "unknown step: $verb" >&2 && exit 2 ;;
    esac
done <"$steps"

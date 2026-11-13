# outside: sourced by capture-tauri.sh and live-tauri.sh, in a Linux VM only: Xvfb, openbox and xdotool exist there.
# value: tauri_open <world> builds the pages and the debug binary (its target in $STATION_BENCH, shared by every
# value: copy), serves the world through serve-world.py, takes a free X display of its own, opens the window in a
# value: sandboxed HOME on that socket and sets $win to the window of THAT process. Its trap removes the sandbox and
# value: what the build leaves in the tree (src-tauri/gen, src-tauri/Cargo.lock), so no leftover fails a check.
# value: tauri_relaunch ends the window and opens it again in the same sandbox HOME, so what it stored is kept.
# shellcheck shell=bash
[ "$(uname)" = Linux ] || {
    echo "tauri-session.sh: the window opens on Xvfb in a Linux VM only, never on a Mac's screen" >&2
    exit 1
}
export CARGO_TARGET_DIR=$STATION_BENCH/tauri-target WEBKIT_DISABLE_DMABUF_RENDERER=1 NO_AT_BRIDGE=1

tauri_close() {
    kill "${app:-}" "${wm:-}" "${xvfb:-}" "${bridge:-}" 2>/dev/null || true
    rm -rf "${sandbox:-}" "$window/src-tauri/gen" "$window/src-tauri/Cargo.lock" "$scripts/__pycache__"
}

tauri_open() {
    local world=$1
    if [ -s "$HOME/.cargo/env" ]; then
        # shellcheck disable=SC1091
        source "$HOME/.cargo/env"
    fi
    trap tauri_close EXIT
    mkdir -p "$window/.artifacts"
    (cd "$window" && npm run build) >"$window/.artifacts/tauri-vite.log" 2>&1
    tauri_inputs_dated
    # hazard: the binary cargo leaves at debug/station-window is the last copy's to build, whichever folder it
    # hazard: is: each copy builds and takes its own under one lock, and opens that one.
    (cd "$window/src-tauri" && flock "$CARGO_TARGET_DIR/.window-binary.lock" sh -c \
        'cargo build -j "$1" --features tauri/custom-protocol && cp "$CARGO_TARGET_DIR/debug/station-window" "$2"' \
        build "${TAURI_JOBS:-6}" "$window/.artifacts/station-window") >"$window/.artifacts/tauri-cargo.log" 2>&1
    sandbox=$(mktemp -d)
    local socket=$sandbox/engine.sock
    mkdir -p "$sandbox/config" "$sandbox/data" "$sandbox/state" "$sandbox/cache" "$sandbox/runtime"
    chmod 700 "$sandbox/runtime"
    python3 "$scripts/serve-world.py" "$world" "$socket" >"$window/.artifacts/tauri-bridge.log" 2>&1 &
    bridge=$!
    Xvfb -displayfd 3 -screen 0 1600x1100x24 3>"$sandbox/display" >/dev/null 2>&1 &
    xvfb=$!
    for _ in $(seq 1 120); do
        [ -S "$socket" ] && [ -s "$sandbox/display" ] && break
        sleep 0.5
    done
    export DISPLAY=:$(tr -d '\n' <"$sandbox/display")
    openbox >/dev/null 2>&1 &
    wm=$!
    sleep 1
    : >"$window/.artifacts/tauri-window.log"
    tauri_launch
}

tauri_launch() {
    HOME=$sandbox XDG_CONFIG_HOME=$sandbox/config XDG_DATA_HOME=$sandbox/data XDG_STATE_HOME=$sandbox/state \
        XDG_CACHE_HOME=$sandbox/cache XDG_RUNTIME_DIR=$sandbox/runtime STATION_ENGINE_SOCKET=$sandbox/engine.sock \
        "$window/.artifacts/station-window" >>"$window/.artifacts/tauri-window.log" 2>&1 &
    app=$!
    win=
    for _ in $(seq 1 120); do
        win=$(xdotool search --onlyvisible --pid "$app" --name '^Station$' 2>/dev/null | head -1 || true)
        [ -n "$win" ] && break
        sleep 0.5
    done
    [ -n "$win" ] || { echo "no Station window of pid $app appeared on $DISPLAY" >&2; return 1; }
    xdotool windowsize "$win" "${TAURI_WIDTH:-1440}" "${TAURI_HEIGHT:-1000}"
    xdotool windowmove "$win" 0 0
    sleep 8
}

tauri_relaunch() {
    kill "$app" && wait "$app" 2>/dev/null || true
    tauri_launch
}

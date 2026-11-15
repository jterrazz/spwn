# outside: sourced by every bench script, in a Linux VM only: a Mac never installs, builds nor runs the window's toolchain.
# value: STATION_BENCH names the cache every copy of the repository shares (~/bench by default): npm's cache,
# value: Playwright's browsers, the oxc allocator shim and the world builds; Go's caches are only seeded from it.
# shellcheck shell=bash
[ "$(uname)" = Linux ] || {
    echo "bench-env.sh: the bench runs in a Linux VM only; a Mac never installs, builds nor runs the window's toolchain" >&2
    exit 1
}
export STATION_BENCH=${STATION_BENCH:-$HOME/bench}
scripts=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
window=$(dirname "$scripts")
root=$(cd "$window/../.." && pwd)
bench=$STATION_BENCH
mkdir -p "$bench"
if [ -s "$HOME/.nvm/nvm.sh" ]; then
    # shellcheck disable=SC1091
    source "$HOME/.nvm/nvm.sh" >/dev/null
fi
export PATH=$HOME/go-sdk/go/bin:$PATH
unset GOFLAGS GOENV GH_TOKEN GITHUB_TOKEN
# invariant: Go's caches are this copy's own, under its artifacts folder: two copies building at once never write one
# invariant: cache. A copy's first run seeds them from the shared bench's, hard-linked where the disk allows, else copied.
# hazard: Go writes its module cache read-only and cp keeps the modes, so a copy's folder would refuse rm -rf: its own
# hazard: directories are made writable, never the files, which are the bench's own under a hard link.
go_caches=$window/.artifacts/go
for cache in gomod gocache gopath; do
    if [ ! -d "$go_caches/$cache" ]; then
        mkdir -p "$go_caches"
        if [ -d "$bench/$cache" ]; then
            cp -al "$bench/$cache" "$go_caches/$cache" 2>/dev/null || {
                rm -rf "${go_caches:?}/$cache"
                cp -a "$bench/$cache" "$go_caches/$cache"
            }
            find "$go_caches/$cache" -type d ! -perm -u+w -exec chmod u+w {} +
        else
            mkdir -p "$go_caches/$cache"
        fi
    fi
done
export GOMODCACHE=$go_caches/gomod GOCACHE=$go_caches/gocache GOPATH=$go_caches/gopath GOTOOLCHAIN=local
export GOFLAGS=-modcacherw
export PLAYWRIGHT_BROWSERS_PATH=$bench/pw
export npm_config_cache=$bench/npm-cache
export CI=1
export PYTHONDONTWRITEBYTECODE=1
export NORESERVE=$bench/noreserve/noreserve.so

# hazard: cargo judges an input changed by its mtime, and rsync -a carries the Mac's: a copy synced again with a file
# hazard: dated before its last build but different (bridged.ts, which build.rs reads; a .rs of another branch) is
# hazard: served the shared target's stale output. Each copy's folder has units of its own there (cargo keys a
# hazard: path package by its path), so the inputs are touched when their content moved since this copy's last build.
tauri_inputs_dated() {
    local stamps=$bench/tauri-target/.inputs inputs sum key
    mkdir -p "$stamps"
    mapfile -t inputs < <(find "$window/src-tauri/build.rs" "$window/src-tauri/Cargo.toml" \
        "$window/src-tauri/tauri.conf.json" "$window/src-tauri/src" "$window/src-tauri/capabilities" \
        "$window/src-tauri/icons" "$window/src/engine/bridged.ts" -type f | sort)
    sum=$(sha256sum "${inputs[@]}" | sha256sum | cut -c1-64)
    key=$(printf %s "$window" | sha256sum | cut -c1-16)
    [ "$(cat "$stamps/$key" 2>/dev/null)" != "$sum" ] || return 0
    touch "${inputs[@]}"
    printf %s "$sum" >"$stamps/$key"
}

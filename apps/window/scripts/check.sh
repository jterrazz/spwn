#!/usr/bin/env bash
# outside: a Linux VM only.
# value: the full proof of the window, after bench-setup.sh:
# value: the window's tests in Chromium, `typescript check` over the repository, the release and proof builds, the
# value: scenarios of specs/website/ in Chromium against that proof build, then
# value: the bridge's Rust tests (`cargo test`, in the shared Tauri target, WINDOW_CARGO_JOBS jobs, 3 by default),
# value: `cargo fmt --check` and `cargo clippy --all-targets` with warnings denied (the rustfmt and clippy components),
# value: then recordings-check.py, which fails a recording the world build of this tree no longer answers so.
# value: One exit code: 0 when apps/window has no finding and its tests and builds pass; a finding outside
# value: apps/window is the root's and is counted apart. `--update` rewrites the
# value: goldens the tests compare, before the toolchain's named-path pass asks for a golden no run wrote yet.
# value: `--no-replay` skips recordings-check.py, most of the run's time, for the inner loop: it says so on its last
# value: line and is never the proof. WINDOW_TEST_WORKERS sets the tests' workers (6 by default): fewer when other
# value: lots share the VM.
set -uo pipefail
# shellcheck source=bench-env.sh
source "$(dirname "$0")/bench-env.sh"
# shellcheck source=findings.sh
source "$scripts/findings.sh"
bash "$scripts/bench-setup.sh" >/dev/null || exit 1
cd "$root"
logs=$window/.artifacts/bench
mkdir -p "$logs"
start=$(date +%s)
update=()
replay=1
for flag in "$@"; do
    case $flag in
        --update) update=(-u) ;;
        --no-replay) replay=0 ;;
        *)
            echo "usage: check.sh [--update] [--no-replay]" >&2
            exit 2
            ;;
    esac
done
(cd "$window" && timeout 520 npx vitest --run --maxWorkers="${WINDOW_TEST_WORKERS:-6}" "${update[@]}") >"$logs/test.log" 2>&1
test=$?
LD_PRELOAD=$NORESERVE timeout 520 npx typescript check >"$logs/check.log" 2>&1
check=$?
(cd "$window" && npm run build && npm run build:proof) >"$logs/build.log" 2>&1
build=$?
timeout 520 npm --workspace specs exec -- vitest --run --project website >"$logs/scenarios.log" 2>&1
scenarios=$?
if [ -s "$HOME/.cargo/env" ]; then
    # shellcheck disable=SC1091
    source "$HOME/.cargo/env"
fi
tauri_inputs_dated
(cd "$window/src-tauri" && CARGO_TARGET_DIR=$bench/tauri-target timeout 520 cargo test -j "${WINDOW_CARGO_JOBS:-3}") \
    >"$logs/rust.log" 2>&1
rust=$?
(cd "$window/src-tauri" && cargo fmt --check) >"$logs/rustfmt.log" 2>&1
rustfmt=$?
(cd "$window/src-tauri" && CARGO_TARGET_DIR=$bench/tauri-target timeout 520 \
    cargo clippy -j "${WINDOW_CARGO_JOBS:-3}" --all-targets -- -D warnings) >"$logs/clippy.log" 2>&1
clippy=$?
rm -rf "$window/src-tauri/gen" "$window/src-tauri/Cargo.lock"
recordings=0
if [ "$replay" -eq 1 ]; then
    timeout 520 python3 "$scripts/recordings-check.py" >"$logs/recordings.log" 2>&1
    recordings=$?
else
    echo "recordings: skipped by --no-replay, not proved" >"$logs/recordings.log"
fi
echo "typescript check: exit $check, its passes:"
sed 's/\x1b\[[0-9;]*m//g' "$logs/check.log" | grep -aE ' RUN |Passed|Failed' | paste - - \
    | sed -E 's/ *RUN +//; s/✓ Passed.*/ok/; s/✗ Failed.*/FAILED/' | sed 's/^/  /'
summary "$logs/check.log"
window_clean=$?
echo "tests: exit $test, $(sed 's/\x1b\[[0-9;]*m//g' "$logs/test.log" | grep -aE '^ +Tests ' | tr -s ' ')"
echo "builds: exit $build"
echo "scenarios: exit $scenarios, $(sed 's/\x1b\[[0-9;]*m//g' "$logs/scenarios.log" | grep -aE '^ +Tests ' | tr -s ' ')"
echo "rust tests: exit $rust, $(grep -aE '^test result' "$logs/rust.log" | tail -1)"
echo "rustfmt: exit $rustfmt, $(grep -ac '^Diff in' "$logs/rustfmt.log") files to format"
echo "clippy: exit $clippy, $(grep -acE '^(warning|error)' "$logs/clippy.log") findings"
echo "recordings: exit $recordings, $(tail -1 "$logs/recordings.log")"
status=0
[ "$check" -ne 124 ] && [ "$window_clean" -eq 0 ] && [ "$test" -eq 0 ] && [ "$build" -eq 0 ] \
    && [ "$scenarios" -eq 0 ] && [ "$rust" -eq 0 ] \
    && [ "$rustfmt" -eq 0 ] && [ "$clippy" -eq 0 ] && [ "$recordings" -eq 0 ] || status=1
skipped=
[ "$replay" -eq 1 ] || skipped=" · recordings skipped (--no-replay): not the proof"
echo "check: window exit $status · root $root_findings findings, typescript check exit $check · $(($(date +%s) - start)) s${skipped} · logs in $logs"
exit "$status"

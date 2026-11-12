#!/bin/bash
# outside: GitHub's macOS runners are VMs and nest none, so only
# `make e2e-machine KIND=linux|macos` runs this, on a host, never `make check` nor CI.
# invariant: every VM and image made in the owner's tart store is named station-e2e-*,
# outside station's ownership rule, so the product never lists nor touches them; tier
# 2's VMs live in a store of the run's own.
# usage: run.sh linux|macos [label]: the log is .artifacts/e2e/<label>.log
set -uo pipefail

kind=${1:-}
case "$kind" in
  linux | macos) ;;
  *) echo "usage: make e2e-machine KIND=linux|macos [LABEL=<label>]" >&2; exit 2 ;;
esac
label=${2:-$kind}

repo=$(cd "$(dirname "$0")/../.." && pwd)
suite="$repo/specs/machine"
out="$repo/.artifacts/e2e"
log="$out/$label.log"
work="$out/$label"
mkdir -p "$out"
rm -rf "$work"
mkdir -p "$work"
: > "$log"

. "$suite/harness.sh"

for binary in station station-linux-arm64; do
  [ -x "$repo/.artifacts/go/$binary" ] || fail "no binary at .artifacts/go/$binary: make build build-linux"
done

tart_bin="$HOME/.local/bin/tart"
[ -x "$tart_bin" ] || tart_bin=$(command -v tart || true)
[ -n "$tart_bin" ] || fail "tart is not installed: station apps install tart"
# value: the store is the owner's: a clone never prunes its other images to make room.
tart() { TART_NO_AUTO_PRUNE=1 "$tart_bin" "$@"; }

cleanup_tier() { :; }
# invariant: whatever happens, every VM this run made is stopped and deleted and
# every process it started is gone; only the macOS image cache survives.
cleanup() {
  [ -z "${watchdog:-}" ] || { pkill -P "$watchdog" 2>/dev/null; kill "$watchdog" 2>/dev/null; }
  cleanup_tier
  say "-- passed: ${passed[*]:-none}"
  [ ${#failed[@]} -eq 0 ] || say "-- failed: ${failed[*]}"
}
trap cleanup EXIT
trap 'exit 130' INT TERM

# invariant: a run ends at its limit, E2E_LIMIT seconds (3 h for linux, 8 h for macos,
# whose Xcode case alone takes hours): the watchdog ends what the run waits on, so its
# TERM is heard and the cleanup runs; a run that ends first ends the watchdog.
limit=${E2E_LIMIT:-$([ "$kind" = linux ] && echo 10800 || echo 28800)}
(
  sleep "$limit"
  say "FAIL the run passed its limit of $limit s"
  kill -TERM $$
  pkill -TERM -P $$
) > /dev/null 2>&1 &
watchdog=$!

say "== tier $([ "$kind" = linux ] && echo 2 || echo 3): $kind, $(cd "$repo" && git describe --always --dirty 2>/dev/null)"
run_tier "$suite/$kind/tier.sh"
[ ${#failed[@]} -eq 0 ] || exit 1
say "PASS tier $([ "$kind" = linux ] && echo 2 || echo 3): $kind"

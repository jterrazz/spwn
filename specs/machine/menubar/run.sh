#!/bin/bash
# outside: no Xcode on the owner's Mac and no macOS SDK in `os`, so the menu bar's
# gate and its golden rewrites run here, in a throwaway macOS VM with Xcode, on a host
# with tart. Never `make check` nor CI.
# invariant: every VM this run makes is named station-e2e-menubar-<label> and is
# stopped and deleted by the trap, whatever happens; only the base survives, a cache
# like tier 3's image. The VM runs headless: nothing opens a window on the Mac.
# usage: run.sh <label> [--update-goldens]
#   the log is .artifacts/e2e/menubar-<label>.log; what the run wrote under
#   apps/menubar (an .actual.png, or with --update-goldens the rewritten goldens and
#   INDEX) comes back to .artifacts/e2e/menubar-<label>/changes/, the tree untouched.
#   STATION_XCODE_TAG picks the cirruslabs/macos-tahoe-xcode tag (default below).
set -uo pipefail

label=${1:-}
update=0
case "${2:-}" in
  "") ;;
  --update-goldens) update=1 ;;
  *) label="" ;;
esac
[ -n "$label" ] || { echo "usage: specs/machine/menubar/run.sh <label> [--update-goldens]" >&2; exit 2; }

repo=$(cd "$(dirname "$0")/../../.." && pwd)
suite="$repo/specs/machine"
out="$repo/.artifacts/e2e"
log="$out/menubar-$label.log"
work="$out/menubar-$label"
mkdir -p "$out"
rm -rf "$work"
mkdir -p "$work/changes"
: > "$log"

. "$suite/harness.sh"

tart_bin="$HOME/.local/bin/tart"
[ -x "$tart_bin" ] || tart_bin=$(command -v tart || true)
[ -n "$tart_bin" ] || fail "tart is not installed: station apps install tart"
# value: the store is the owner's: a clone never prunes its other images to make room.
tart() { TART_NO_AUTO_PRUNE=1 "$tart_bin" "$@"; }

tag=${STATION_XCODE_TAG:-26.5}
image="ghcr.io/cirruslabs/macos-tahoe-xcode:$tag"
hash=$(printf '%s' "$image" | shasum -a 256 | cut -c1-12)
base="station-e2e-xcode-$hash"
vm="station-e2e-menubar-$label"
user=admin
t0=$SECONDS

cleanup() {
  local left
  tart stop "$vm" > /dev/null 2>&1 || true
  tart delete "$vm" > /dev/null 2>&1 || true
  [ -z "${tart_pid:-}" ] || kill "$tart_pid" > /dev/null 2>&1 || true
  left=$(tart list --quiet --source local 2> /dev/null | grep '^station-e2e-menubar-' || true)
  [ -z "$left" ] || say "WARN left behind: $left"
  say "-- cleaned: $vm deleted, base $base kept; $((SECONDS - t0)) s in all"
}
trap cleanup EXIT
trap 'exit 130' INT TERM

# invariant: at most one macOS VM of the run, and Apple allows two: a running
# station-e2e VM, the cleanup of another run, is a reason to stop, not to share.
running=$(tart list --format json 2> /dev/null | jq -r '.[] | select(.Running == true or .State == "running") | .Name' | grep '^station-e2e-' || true)
[ -z "$running" ] || fail "a station-e2e VM is running ($running): the menu bar VM runs alone"
tart list --quiet --source local | grep -qx "$vm" && fail "$vm exists: a run of this label left it, tart delete it"

say "== menu bar in a VM: $label, $(cd "$repo" && git describe --always --dirty 2> /dev/null), $image"

# value: the base is the upstream image cloned once from the OCI cache, kept like
# tier 3's image; a new tag is a new hash, and the older bases are deleted then.
if ! tart list --quiet --source local | grep -qx "$base"; then
  say "-- making $base from $image (the pull is tens of GB, once)"
  for old in $(tart list --quiet --source local | grep '^station-e2e-xcode-'); do
    tart delete "$old" > /dev/null 2>&1 || true
  done
  t=$SECONDS
  tart clone "$image" "$base" >> "$log" 2>&1 || fail "pull $image"
  say "-- pulled in $((SECONDS - t)) s"
fi

t=$SECONDS
tart clone "$base" "$vm" >> "$log" 2>&1 || fail "clone $base"
tart set "$vm" --memory 8192 >> "$log" 2>&1 || fail "size $vm"
tart run --no-graphics "$vm" >> "$work/tart.log" 2>&1 &
tart_pid=$!
ip=$(tart ip "$vm" --wait 180) || fail "no address for $vm"

# value: a fresh guest knows only admin/admin; the first ssh answers the password
# through SSH_ASKPASS and installs this run's key, every later ssh uses it.
ssh_key="$work/id_ed25519"
ssh-keygen -q -t ed25519 -N '' -C "station-e2e" -f "$ssh_key" || fail "ssh-keygen"
askpass="$work/askpass"
printf '#!/bin/sh\necho admin\n' > "$askpass"
chmod 700 "$askpass"
n=0
until SSH_ASKPASS="$askpass" SSH_ASKPASS_REQUIRE=force DISPLAY=none \
  ssh "${ssh_opts[@]/BatchMode=yes/BatchMode=no}" -o PubkeyAuthentication=no "$user@$ip" \
  "mkdir -p ~/.ssh && cat >> ~/.ssh/authorized_keys" < "$ssh_key.pub" 2> /dev/null; do
  n=$((n + 1))
  [ $n -lt 60 ] || fail "ssh into $vm at $ip"
  sleep 2
done
say "-- $vm at $ip, booted in $((SECONDS - t)) s"
in_vm() { ssh -n "${ssh_opts[@]}" -i "$ssh_key" "$user@$ip" "$1"; }
say "xcode: $(in_vm 'xcodebuild -version | head -1; sw_vers -productVersion' 2>&1 | tr '\n' ' ')"

# invariant: the branch goes in by git, the committed tree and nothing of the
# Mac's folders; the VM's repo takes it on a branch it does not hold checked out.
in_vm 'rm -rf ~/station && git init -q -b scratch ~/station' || fail "git init in the VM"
GIT_SSH_COMMAND="ssh ${ssh_opts[*]} -i $ssh_key" \
  git -C "$repo" push -q "$user@$ip:station" "HEAD:refs/heads/run" >> "$log" 2>&1 || fail "git push into the VM"
in_vm 'cd ~/station && git checkout -q run' || fail "checkout in the VM"
say "-- pushed $(git -C "$repo" rev-parse --short HEAD)"

t=$SECONDS
# value: a login shell for the PATH Xcode and brew give; the gate runs with the
# SDK required, so a VM that lost Xcode fails and never skips.
env_line='STATION_REQUIRE_MENUBAR=1'
[ $update -eq 0 ] || env_line="$env_line STATION_UPDATE_GOLDENS=1 STATION_UPDATE_INDEX=1"
ssh -n "${ssh_opts[@]}" -i "$ssh_key" "$user@$ip" "zsh -lc 'cd ~/station && $env_line make menubar-check'" >> "$log" 2>&1
code=$?
say "-- make menubar-check exit $code in $((SECONDS - t)) s"

# value: what the run wrote under apps/menubar (an .actual.png, a rewritten golden,
# INDEX) comes back as one tar, whatever the verdict: the .actual.png are gitignored,
# so the listing takes the ignored untracked files too.
in_vm "cd ~/station && git ls-files -m -o apps/menubar | tar -cf - -T -" > "$work/changes.tar" 2>> "$log"
tar -xf "$work/changes.tar" -C "$work/changes" 2>> "$log"
changed=$(tar -tf "$work/changes.tar" 2> /dev/null | grep -c . || true)
say "-- $changed files came back to $work/changes"
tar -tf "$work/changes.tar" 2> /dev/null | sed 's/^/   /' | tee -a "$log" | head -80
grep -E "differ|golden|error:|Test run|Executed" "$log" | tail -60 | sed 's/^/   log: /'

if [ $code -ne 0 ]; then
  say "FAIL menubar-check exit $code"
  exit 1
fi
say "PASS menubar-check"

#!/bin/bash
# outside: no Xcode on the owner's Mac and no macOS SDK in `os`, so the menu bar's
# gate and its golden rewrites run here, in a throwaway macOS VM with Xcode, on a host
# with tart. Never `make check` nor CI.
# invariant: every VM this run makes is named station-e2e-menubar-<label> and is
# stopped and deleted by the trap, whatever happens; only the base survives, a cache
# like tier 3's image. The VM runs headless: nothing opens a window on the Mac.
# usage: run.sh <label> [--update-goldens | --capture]
#   the log is .artifacts/e2e/menubar-<label>.log; what the run wrote under
#   apps/menubar (an .actual.png, or with --update-goldens the rewritten goldens and
#   INDEX) comes back to .artifacts/e2e/menubar-<label>/changes/, the tree untouched.
#   STATION_XCODE_IMAGE picks another image (default below, pinned by digest);
#   STATION_VM_DISPLAY=<W>x<H>[pt|px] sets the guest's display.
#   --capture builds Station.app instead of the gate and reads its live panel
#   back over the Mac's VNC framebuffer (capture/drive.py), over a fake engine
#   serving home.exposed, on M1's three wallpapers; the captures, compare
#   sheets and contrast table (capture/measure.py) land in
#   .artifacts/e2e/menubar-<label>/capture/. STATION_CAPTURE_SET picks the
#   shots (walls,behave,trial by default; sweep, a11y and old drive.py names),
#   STATION_CAPTURE_ITEM=x,y,w,h gives old's item frame.
set -uo pipefail

label=${1:-}
update=0
capture=0
case "${2:-}" in
  "") ;;
  --update-goldens) update=1 ;;
  --capture) capture=1 ;;
  *) label="" ;;
esac
[ -n "$label" ] || { echo "usage: specs/machine/menubar/run.sh <label> [--update-goldens | --capture]" >&2; exit 2; }

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

# invariant: the goldens are drawn by this image, so it is pinned by digest, not by a
# tag a registry may move; the digest is macos-tahoe-xcode:27.2 (macOS 26.6.2, Xcode 27.2).
image=${STATION_XCODE_IMAGE:-ghcr.io/cirruslabs/macos-tahoe-xcode@sha256:f8315a2f627daf64ab2c4e11d29a4d267953902a16151ee28fcdd7185927c317}
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
# value: STATION_VM_DISPLAY=<W>x<H>[pt|px] sets the guest's display, since the
# icon goldens are drawn at the screen's scale.
# value: the capture reads M1's scenes at 2x on a 1024 x 768 pt screen, as
# the play rig did.
[ $capture -eq 0 ] || STATION_VM_DISPLAY=${STATION_VM_DISPLAY:-1024x768pt}
[ -z "${STATION_VM_DISPLAY:-}" ] || tart set "$vm" --display "$STATION_VM_DISPLAY" >> "$log" 2>&1 || fail "display $vm"
vnc_flag=()
[ $capture -eq 0 ] || vnc_flag=(--vnc-experimental)
tart run --no-graphics ${vnc_flag[@]+"${vnc_flag[@]}"} "$vm" >> "$work/tart.log" 2>&1 &
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
say "display: $(in_vm 'system_profiler SPDisplaysDataType | grep -E "Chipset|Resolution|UI Looks|Retina|Display Type"' 2>&1 | tr -s ' ' | tr '\n' ';')"
say "xcode: $(in_vm 'xcodebuild -version | head -1; sw_vers -productVersion' 2>&1 | tr '\n' ' ')"

# invariant: the branch goes in by git, the committed tree and nothing of the
# Mac's folders; the VM's repo takes it on a branch it does not hold checked out.
in_vm 'rm -rf ~/station && git init -q -b scratch ~/station' || fail "git init in the VM"
GIT_SSH_COMMAND="ssh ${ssh_opts[*]} -i $ssh_key" \
  git -C "$repo" push -q "$user@$ip:station" "HEAD:refs/heads/run" >> "$log" 2>&1 || fail "git push into the VM"
in_vm 'cd ~/station && git checkout -q run' || fail "checkout in the VM"
say "-- pushed $(git -C "$repo" rev-parse --short HEAD)"

if [ $capture -eq 1 ]; then
  # invariant: the VNC server is the Mac's Virtualization framebuffer, read
  # by vncdotool in a venv the runs share; nothing opens on the Mac.
  url=$(grep -o 'vnc://[^ ]*' "$work/tart.log" | head -1)
  [ -n "$url" ] || fail "no vnc:// line in $work/tart.log"
  venv="$out/menubar-venv"
  [ -x "$venv/bin/python" ] || { python3 -m venv "$venv" && "$venv/bin/pip" install -q vncdotool pillow numpy scipy; } >> "$log" 2>&1 || fail "the capture's venv"
  t=$SECONDS
  in_vm "zsh -lc 'cd ~/station && make menubar'" >> "$log" 2>&1 || fail "make menubar in the VM"
  say "-- built Station.app in $((SECONDS - t)) s"
  shots="$work/capture"
  "$venv/bin/python" -I "$suite/menubar/capture/drive.py" "$ip" "$ssh_key" "$url" "$repo" "$shots" \
    "${STATION_CAPTURE_SET:-walls,behave,trial}" "${STATION_CAPTURE_ITEM:-}" 2>&1 | tee -a "$log"
  code=${PIPESTATUS[0]}
  in_vm 'cat ~/cap/replay.log' >> "$log" 2>&1
  "$venv/bin/python" -I "$suite/menubar/capture/measure.py" "$shots" "$repo/apps/menubar/tests/station-kit-tests/testdata/stories" \
    ${STATION_CAPTURE_M1:+"$STATION_CAPTURE_M1"} 2>&1 | tee -a "$log"
  [ $code -eq 0 ] || fail "capture: drive.py exit $code"
  say "PASS capture: $shots"
  exit 0
fi

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
# so the listing takes the ignored untracked files too, but not the symbols the
# build regenerates.
in_vm "cd ~/station && git ls-files -m -o apps/menubar | grep -v '/symbols.xcassets/' | tar -cf - -T -" > "$work/changes.tar" 2>> "$log"
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

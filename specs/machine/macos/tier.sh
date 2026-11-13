#!/bin/bash
# value: tier 3: one fresh macOS VM plays a Mac: install.sh, init, the apps of every
# method, every setting, the shell block, Activity, then free and erase on a clone.
# Sourced by run.sh.

# invariant: Apple runs two macOS VMs at once at most, so another running
# macOS VM is refused and a running Linux one is not; the 8 GB tier 3's VM
# boots with must be free beside the running VMs, macOS keeping 4 GB.
running=$(tart list --format json 2>/dev/null | jq -r '.[] | select(.State == "running" or .Running == true) | .Name')
held=0
macos=""
for vm in $running; do
  info=$(tart get "$vm" --format json 2>/dev/null) || continue
  [ "$(jq -r '.OS' <<< "$info")" = darwin ] && macos="$macos $vm"
  held=$((held + $(jq -r '.Memory // 0' <<< "$info")))
done
[ -z "$macos" ] || fail "a macOS VM is running ($macos ): tier 3 runs one of its own, and Apple runs two at most"
free_mb=$(($(sysctl -n hw.memsize) / 1048576 - 4096 - held))
[ "$free_mb" -ge 8192 ] || fail "not enough memory for tier 3's 8 GB VM: $free_mb MB left beside the running VMs"

run_id="$$"
made=()
cleanup_tier() {
  local vm left
  for vm in "${made[@]:-}"; do
    [ -n "$vm" ] || continue
    tart stop "$vm" > /dev/null 2>&1 || true
    tart delete "$vm" > /dev/null 2>&1 || true
  done
  left=$(tart list --quiet --source local 2>/dev/null | grep "^station-e2e-macos-$run_id" || true)
  [ -z "$left" ] || say "WARN left behind: $left"
}

ssh_key="$work/id_ed25519"
ssh-keygen -q -t ed25519 -N '' -C "station-e2e" -f "$ssh_key" || fail "ssh-keygen"
hash=$(cd "$repo" && find apps/engine/catalog catalog -type f | LC_ALL=C sort | xargs cat | shasum -a 256 | cut -c1-12)
base="station-e2e-macos-$hash"
user=admin
say "-- base $base"

# value: a fresh macOS guest knows only admin/admin; the first ssh answers the
# password through SSH_ASKPASS and installs this run's key, every later ssh uses it;
# ssh keeps the first value of an option, so BatchMode is turned off in place.
askpass="$work/askpass"
printf '#!/bin/sh\necho admin\n' > "$askpass"
chmod 700 "$askpass"
boot() {
  local vm=$1 ip n=0
  tart run --no-graphics "$vm" >> "$work/$vm.tart.log" 2>&1 &
  ip=$(tart ip "$vm" --wait 180) || return 1
  until SSH_ASKPASS="$askpass" SSH_ASKPASS_REQUIRE=force DISPLAY=none \
    ssh "${ssh_opts[@]/BatchMode=yes/BatchMode=no}" -o PubkeyAuthentication=no "$user@$ip" \
    "mkdir -p ~/.ssh && cat >> ~/.ssh/authorized_keys" < "$ssh_key.pub" 2> /dev/null; do
    n=$((n + 1))
    [ $n -lt 60 ] || return 1
    sleep 2
  done
  echo "$ip"
}

if ! tart list --quiet --source local | grep -qx "$base"; then
  say "-- prebaking $base from ghcr.io/cirruslabs/macos-tahoe-base:latest"
  for old in $(tart list --quiet --source local | grep '^station-e2e-macos-' | grep -v -- "-$run_id"); do
    tart delete "$old" > /dev/null 2>&1 || true
  done
  bake="station-e2e-macos-bake-$run_id"
  made+=("$bake")
  tart clone ghcr.io/cirruslabs/macos-tahoe-base:latest "$bake" >> "$log" 2>&1 || fail "pull the macOS base"
  ip=$(boot "$bake") || fail "boot $bake"
  ssh "${ssh_opts[@]}" -i "$ssh_key" "admin@$ip" 'bash -s' < "$suite/macos/prebake.sh" >> "$log" 2>&1 || fail "prebake"
  ssh "${ssh_opts[@]}" -i "$ssh_key" "admin@$ip" 'rm -f ~/.ssh/authorized_keys' || true
  tart stop "$bake" >> "$log" 2>&1
  tart rename "$bake" "$base" >> "$log" 2>&1 || fail "keep $base"
  made=()
fi

vm="station-e2e-macos-$run_id"
made+=("$vm")
tart clone "$base" "$vm" >> "$log" 2>&1 || fail "clone $base"
tart set "$vm" --memory 8192 >> "$log" 2>&1 || fail "size $vm"
ip=$(boot "$vm") || fail "boot $vm"
say "-- $vm at $ip"

# invariant: inside, the guest plays a Mac: STATION_MACHINE_KIND=mac on every call
# but those that play a macOS workspace's guest: the Activity read, and the npm and
# uv apps the catalogue places only there.
in_vm() { ssh -n "${ssh_opts[@]}" -i "$ssh_key" "$user@$ip" "export STATION_MACHINE_KIND=mac; $1"; }
say "kern.hv_vmm_present: $(in_vm 'sysctl -n kern.hv_vmm_present' 2>&1)"

in_vm 'rm -rf /tmp/station-e2e && mkdir -p /tmp/station-e2e/hello /tmp/station-e2e/npm' || fail "scratch folder"
scp "${ssh_opts[@]}" -i "$ssh_key" -q "$repo/.artifacts/go/station" "$user@$ip:/tmp/station-e2e/station" || fail "copy the binary"
scp "${ssh_opts[@]}" -i "$ssh_key" -q "$repo/scripts/install.sh" "$user@$ip:/tmp/station-e2e/install.sh" || fail "copy install.sh"

listing() { in_vm 'cd ~ && find . -mindepth 1 -maxdepth 4 2>/dev/null | sed "s|^\./||" | LC_ALL=C sort'; }
station="\$HOME/.local/bin/station"
brew_path='PATH=/opt/homebrew/bin:$PATH'
mise_path="PATH=\$HOME/.local/share/mise/shims:\$HOME/.local/bin:/opt/homebrew/bin:\$PATH"
fp="$work/home"
mkdir -p "$fp"
steps=()
n=0
# invariant: the home is listed after every step, so an unmatched path names the
# step that made it.
step() {
  local name=$1 command=$2 code
  n=$((n + 1))
  say "-- step $n: $name"
  say "   \$ $command"
  in_vm "$command" >> "$log" 2>&1
  code=$?
  say "   exit $code"
  listing > "$fp/$n.txt"
  steps+=("$n:$name")
  [ $code -eq 0 ] || fail "step $n ($name) exited $code"
}


case_begin install-then-init SU-08
listing > "$fp/0-before.txt" || fail "list the home"
# value: brew and sudo recorded on their way to the real ones, so the run says
# whether init reached either; the record lives outside the home the footprint reads.
in_vm 'mkdir -p /tmp/station-e2e/watch && for c in brew sudo; do real=$(PATH=/opt/homebrew/bin:$PATH command -v $c || echo /usr/bin/false); printf "#!/bin/sh\necho \"%s \$*\" >> /tmp/station-e2e/.calls\nexec %s \"\$@\"\n" "$c" "$real" > /tmp/station-e2e/watch/$c; chmod +x /tmp/station-e2e/watch/$c; done' || fail "the brew and sudo recorders"
began=$(date +%s)
step "install.sh" 'PATH=/tmp/station-e2e/watch:/opt/homebrew/bin:$PATH STATION_INSTALL_FROM=/tmp/station-e2e/station sh /tmp/station-e2e/install.sh --name e2e </dev/null'
say "   install.sh and init took $(($(date +%s) - began)) s"
# value: the seam's verdict is said in the log with what it recorded, so a pass
# is read there, not inferred from a silence.
seam=$(in_vm 'grep -E "^(brew|sudo) " /tmp/station-e2e/.calls 2>/dev/null')
if [ -n "$seam" ]; then
  say "$(sed 's/^/   seam: /' <<< "$seam")"
  fail "init ran brew or sudo"
fi
say "   seam: install.sh and init ran no brew and no sudo"
init=$(in_vm "PATH=/tmp/station-e2e/watch:/opt/homebrew/bin:\$PATH $station init </dev/null")
say "$(sed 's/^/   init: /' <<< "$init")"
for row in name zshrc layout; do
  grep -q "^●	$row" <<< "$init" || fail "init shows no ● $row step"
done
grep -qE "^	?none bound by init" <<< "$init" || fail "init ends on no empty Machinefile"
in_vm 'test -e ~/Library/LaunchAgents/com.jterrazz.station.engine.plist' && fail "init installed the engine"
while read -r app; do
  case "$app" in '' | '#'*) continue ;; esac
  step "apps install $app" "$brew_path $mise_path $station apps install $app </dev/null"
done < "$suite/macos/one-per-method.txt"
step "go build" "$mise_path; cd /tmp/station-e2e/hello && printf 'module hello\n\ngo 1.21\n' > go.mod && printf 'package main\n\nfunc main() { println(\"hello\") }\n' > main.go && go build -o hello . && ./hello"
step "npm i" "$mise_path; cd /tmp/station-e2e/npm && printf '{\"name\":\"e2e\",\"private\":true}\n' > package.json && npm i is-number@7.0.0"
step "bun add -g" "$mise_path; bun add -g is-number@7.0.0"
step "bunx" "$mise_path; cd /tmp/station-e2e && bunx cowsay@1.6.0 hello"
step "uv tool install" "$mise_path; uv tool install pyflakes==3.2.0"
step "brew install" "$brew_path; brew install hello && hello"
step "gopls" "$mise_path; go install golang.org/x/tools/gopls@v0.20.0"
in_vm 'test -x ~/.local/share/go/bin/gopls' || fail "go install put no gopls in ~/.local/share/go/bin: mise set GOBIN"
in_vm 'test -f ~/.bunfig.toml' || fail "init wrote no ~/.bunfig.toml"
in_vm 'test -e ~/.bun' && fail "~/.bun present after apps install skills, bun add -g and bunx: bun ignored the bunfig"
in_vm "$station disk --json" > "$fp/disk.json" || fail "disk --json"
footprint "$fp" "$fp/disk.json" "$suite/macos/os-noise.txt" || fail "the footprint test"
case_pass


case_begin shell-block-native-calls SU-11
# value: init applies no setting (a setting comes from its own gesture, later the
# Machinefile), so the shortcuts the login shell is probed for are applied first.
step "settings apply docker-shortcuts" "$brew_path $station settings apply docker-shortcuts </dev/null"
login=$(in_vm "zsh -lic 'command -v station; print -r -- \$PATH; node --version; go version; whence -w _station dclean gst; print -r -- \${STATION_TERMINAL:-unset}'" 2> /dev/null)
say "$(sed 's/^/   login: /' <<< "$login")"
grep -q '/.local/bin/station$' <<< "$login" || fail "a login shell finds no station in ~/.local/bin"
grep -q '^v[0-9]' <<< "$login" || fail "a login shell runs no mise node"
grep -q '^go version go' <<< "$login" || fail "a login shell runs no mise go"
grep -q '^_station: function' <<< "$login" || fail "a login shell loads no station completion"
grep -q '^dclean: function' <<< "$login" || fail "a login shell defines no Docker shortcuts"
! grep -qx unset <<< "$login" || fail "a login shell sets no STATION_TERMINAL"
# value: a non-interactive zsh -c reads ~/.zshenv only, and the one block is in
# ~/.zshrc: the call is the one an agent or a task runner makes from the owner's
# session, inheriting its environment.
plain=$(in_vm "zsh -lic 'zsh -c \"command -v station mise; go env GOPATH\"'" 2> /dev/null)
[ "$(wc -l <<< "$plain" | tr -d ' ')" -ge 3 ] || fail "zsh -c finds station, mise and go only partly: $plain"
step "IDE: go env" "$mise_path; d=\$(mise where go)/bin; env -i HOME=\$HOME PATH=\$d go env GOPATH"
step "IDE: gopls version" "$mise_path; d=\$(mise where go)/bin; env -i HOME=\$HOME PATH=\$HOME/.local/share/go/bin:\$d gopls version"
step "IDE: npm config get cache" "$mise_path; d=\$(mise where node)/bin; env -i HOME=\$HOME PATH=\$d:/usr/bin:/bin npm config get cache"
if in_vm "grep -n -e '/private/tmp' -e '/Developer/' -e '.artifacts' ~/.zshrc 2>/dev/null"; then
  fail "the shell block names a clone path"
fi
case_pass


case_begin apps-every-method AP-24
while read -r app method kind; do
  case "$app" in '' | '#'*) continue ;; esac
  pin=""
  [ -z "$kind" ] || pin="STATION_MACHINE_KIND=$kind"
  if [ "$method" = mas ]; then
    in_vm "$brew_path $station apps install mas </dev/null" >> "$log" 2>&1 || miss "apps install mas exited $?"
    reply=$(perl -e 'alarm 120; exec @ARGV' ssh -n "${ssh_opts[@]}" -i "$ssh_key" "$user@$ip" "export STATION_MACHINE_KIND=mac; $brew_path $mise_path $station apps install $app </dev/null" 2>&1)
    code=$?
    [ $code -ne 142 ] || { in_vm 'sudo -n pkill -f "mas install"; pkill -f "station apps install"' || true; reply="no answer in 120 s"; }
    say "   mas $app: exit $code: $(tr '\n' ' ' <<< "$reply" | cut -c1-200)"
    [ $code -ne 0 ] && [ $code -ne 142 ] && grep -qi 'app store\|sign' <<< "$reply" || miss "mas $app signed out gave no sign-in hint ($reply)"
    continue
  fi
  for verb in install update remove; do
    if [ "$verb" = remove ] && [ "$method" = script ]; then
      reply=$(in_vm "$brew_path $mise_path $station apps remove $app --yes </dev/null" 2>&1)
      code=$?
      say "   script $app: remove exit $code: $(tr '\n' ' ' <<< "$reply" | cut -c1-160)"
      [ $code -ne 0 ] && grep -q 'remove it by hand' <<< "$reply" || miss "script $app: remove gave no by-hand hint"
      break
    fi
    flag=""
    [ "$verb" != remove ] || flag=--yes
    in_vm "$pin $brew_path $mise_path $station apps $verb $app $flag </dev/null" >> "$log" 2>&1 || { miss "apps $verb $app ($method) exited $?"; break; }
    state=$(in_vm "$pin $brew_path $mise_path $station apps --all --json" | jq -r --arg a "$app" '.apps[] | select(.name == $a) | .state')
    say "   $method $app${kind:+ as $kind}: $verb, then $state"
    case "$verb" in
      remove) [ "$state" = not-installed ] || [ "$state" = not-managed ] || miss "$app is $state after remove" ;;
      *) [ "$state" != not-installed ] || miss "$app is not installed after $verb" ;;
    esac
  done
done < "$suite/macos/every-method.txt"
case_pass


case_begin settings-every-one SE-34
in_vm "$brew_path $station settings --json" > "$work/settings-0.json" || fail "settings --json"
jq -r '.settings[] | select(.group == "set") | "\(.name) \(.state)"' "$work/settings-0.json" > "$work/settings-set.txt"
[ -s "$work/settings-set.txt" ] || fail "settings --json lists no Set by station setting"
while read -r name state; do
  # value: a setting that is for VMs only (agent-hooks) is refused on a Mac, by design.
  if [ "$state" = not-for-this-machine ]; then
    say "   $name: not for this machine, left alone"
    continue
  fi
  if grep -qx "$name" "$suite/macos/settings-asked.txt"; then
    say "   $name: asks values on a terminal, left to the owner"
    continue
  fi
  in_vm "$brew_path $station settings apply $name </dev/null" >> "$log" 2>&1 || { miss "settings apply $name exited $?"; continue; }
  now=$(in_vm "$brew_path $station settings --json" | jq -r --arg n "$name" '.settings[] | select(.name == $n) | .state')
  [ "$now" = applied ] || { miss "$name reads $now after apply"; continue; }
  in_vm "$brew_path $station settings off $name --yes </dev/null" >> "$log" 2>&1 || { miss "settings off $name exited $?"; continue; }
  off=$(in_vm "$brew_path $station settings --json" | jq -r --arg n "$name" '.settings[] | select(.name == $n) | .state')
  [ "$off" != applied ] || miss "$name still reads applied after off"
  in_vm "$brew_path $station settings apply $name </dev/null" >> "$log" 2>&1 || miss "settings apply $name again exited $?"
  say "   $name: applied, off ($off), applied again"
done < "$work/settings-set.txt"
# value: the system answers first: its `softwareupdate -l` rewrites the list station
# reads, which Apple's catalogue changes between two calls minutes apart.
real=$(ssh "${ssh_opts[@]}" -i "$ssh_key" "$user@$ip" 'bash -s' < "$suite/macos/checked-truth.sh")
in_vm "$brew_path $station settings --json" > "$work/settings-checked.json" || fail "settings --json"
checked=$(jq -r '[.settings[] | select(.group == "checked")] | length' "$work/settings-checked.json")
[ "$(wc -l <<< "$real" | tr -d ' ')" -eq "$checked" ] || miss "station lists $checked checked settings, checked-truth.sh answers $(wc -l <<< "$real" | tr -d ' ')"
while read -r name value; do
  read_value=$(jq -r --arg n "$name" '.settings[] | select(.name == $n) | .value' "$work/settings-checked.json")
  say "   checked $name: station reads $read_value, the system says $value"
  [ "$read_value" = "$value" ] || miss "checked $name reads $read_value, the system says $value"
done <<< "$real"
case_pass


case_begin activity-in-macos-vm AC-36
act=$(ssh -n "${ssh_opts[@]}" -i "$ssh_key" "$user@$ip" "STATION_MACHINE_KIND=workspace $station activity --json")
ncpu=$(in_vm 'sysctl -n hw.ncpu')
mem=$(in_vm 'sysctl -n hw.memsize')
[ "$(jq -r '.cpu.cpus' <<< "$act")" = "$ncpu" ] || fail "cpus $(jq -r '.cpu.cpus' <<< "$act") vs hw.ncpu $ncpu"
[ "$(jq -r '.memory.total_bytes' <<< "$act")" = "$mem" ] || fail "memory $(jq -r '.memory.total_bytes' <<< "$act") vs hw.memsize $mem"
[ "$(jq -r '.thermal, .battery' <<< "$act" | sort -u)" = null ] || fail "a macOS workspace reads a Thermal or a Battery line"
# value: the binary under test is built without cgo (cross-built by `make e2e-binaries`), so it knows the CPU
# count and the memory size only (apps/engine/platform/darwin/kernel_sysctl_darwin.go); the busy
# share, the used memory and the top processes are the cgo build's, which tier 3 does not
# run: they are read empty here, never asserted.
say "   $(jq -c '{cpus: .cpu.cpus, busy: .cpu.busy_percent, used: .memory.used_bytes, total: .memory.total_bytes, up: .up_s}' <<< "$act")"
case_pass


case_begin disk-free-erase DK-16
# invariant: the home is listed six deep before and after free and erase; a removed
# path lies under one the footprint lets the action remove (or is the OS's own), and
# a made one is in the footprint, so a free that deleted ~/.zshrc or ~/.ssh is caught.
deep() { in_vm 'cd ~ && find . -mindepth 1 -maxdepth 6 2>/dev/null | sed "s|^\./||" | LC_ALL=C sort'; }
snapshot() {
  local before=$1 after=$2 may=$3 what=$4 path count
  [ -s "$may" ] || fail "$what: no row path to remove: the check would prove nothing"
  LC_ALL=C comm -23 "$before" "$after" > "$before.removed"
  LC_ALL=C comm -13 "$before" "$after" > "$before.made"
  [ -s "$before.removed" ] || miss "$what removed nothing from the home"
  grep -v '^#' "$suite/macos/os-noise.txt" >> "$may"
  while IFS= read -r path; do miss "$what removed ~/$path, outside its rows"; done \
    < <(awk -v under=1 -f "$suite/footprint.awk" "$may" "$before.removed")
  while IFS= read -r path; do miss "$what made ~/$path, outside the footprint"; done \
    < <(awk -f "$suite/footprint.awk" "$fp/allowed.txt" "$before.made")
  count=$(wc -l < "$before.removed" | tr -d ' ')
  say "   $what: $count home paths removed, every one under its rows; $(wc -l < "$before.made" | tr -d ' ') made, in the footprint"
}
# value: what disk free may remove, by the footprint: its can rebuild and misplaced
# paths, what its Free lines delete, and the unused runtimes `mise prune` drops
# (every runtime the mise config names is checked still installed after).
free_paths() {
  local ref="$repo/docs/reference/footprint.md"
  grep -E '^\| `~/[^`]*` +\| (can rebuild|misplaced) ' "$ref" | sed -E 's/^\| `~\/([^`]*)`.*/\1/'
  grep '^Free:' "$ref" | sed 's/ Erase:.*//' | grep -o 'delete `~/[^`]*`' | sed 's/^delete `~\///; s/`$//'
  printf '%s\n' '.local/share/mise/installs/**' '.local/share/mise/shims/*'
}
row_paths() { jq -r "$2 | .paths[] | select(startswith(\"~/\")) | ltrimstr(\"~/\")" "$1"; }
data_sizes() { jq -r '.rows[] | select(.group == "your-data") | "\(.app) \(.bytes) \(.paths | join(" "))"' "$1"; }
in_vm "$station disk --json" > "$work/disk-before.json" || fail "disk --json"
data_sizes "$work/disk-before.json" > "$work/data-before.txt"
deep > "$work/free-before.txt" || fail "list the home"
in_vm "$brew_path $mise_path $station disk free" >> "$log" 2>&1 || fail "disk free exited $?"
deep > "$work/free-after.txt" || fail "list the home"
free_paths > "$work/free-may.txt"
in_vm "$station disk --json" > "$work/disk-after.json" || fail "disk --json"
snapshot "$work/free-before.txt" "$work/free-after.txt" "$work/free-may.txt" "disk free"
missing=$(in_vm "$mise_path mise ls --missing" 2>&1)
[ -z "$missing" ] || miss "disk free pruned a runtime the mise config names: $missing"
left=$(jq -r '.rows[] | select(.group == "can-rebuild" and .bytes > 0) | "\(.app) \(.bytes)"' "$work/disk-after.json")
[ -z "$left" ] || fail "disk free left: $left"
data_sizes "$work/disk-after.json" > "$work/data-after.txt"
diff "$work/data-before.txt" "$work/data-after.txt" >> "$log" || fail "disk free touched a Your data row"
say "   every Can rebuild row empty or absent, every Your data row untouched"
in_vm 'sync' || fail "sync before the clone"
tart stop "$vm" >> "$log" 2>&1
erase_vm="station-e2e-macos-$run_id-erase"
made+=("$erase_vm")
tart clone "$vm" "$erase_vm" >> "$log" 2>&1 || fail "clone for the erase"
ip=$(boot "$erase_vm") || fail "boot $erase_vm"
in_vm "$station disk --json" > "$work/erase-before.json" || fail "disk --json"
held=$(jq -r '[.rows[] | select(.app == "go" and .group == "your-data") | .paths[] | select(. == "~/.local/share/go/bin/*")] | length' "$work/erase-before.json")
[ "$held" -gt 0 ] || fail "go's bin holds nothing before the erase: the check would prove nothing"
deep > "$work/erase-before.txt" || fail "list the home"
in_vm "$mise_path $station disk erase go --confirm=go" >> "$log" 2>&1 || fail "disk erase go exited $?"
deep > "$work/erase-after.txt" || fail "list the home"
row_paths "$work/erase-before.json" '.rows[] | select(.app == "go" and (.group == "your-data" or .group == "can-rebuild"))' > "$work/erase-may.txt"
snapshot "$work/erase-before.txt" "$work/erase-after.txt" "$work/erase-may.txt" "disk erase go"
in_vm "$station disk --json" > "$work/erase-after.json" || fail "disk --json"
gone=$(jq -r '.rows[] | select(.app == "go" and (.group == "your-data" or .group == "can-rebuild") and .bytes > 0) | "\(.group) \(.paths | join(" "))"' "$work/erase-after.json")
[ -z "$gone" ] || fail "disk erase go left: $gone"
others() { jq -r '.rows[] | select(.app != "go" and .group == "your-data") | "\(.app) \(.bytes) \(.paths | join(" "))"' "$1"; }
diff <(others "$work/erase-before.json") <(others "$work/erase-after.json") >> "$log" || fail "disk erase go touched another app's data"
say "   only go's data gone"
case_pass


# value: the owner's first minutes on a bare Mac: Homebrew is absent, one process
# installs it and then bundles through it, and a second process finds it where the
# first left it, so Homebrew's installer (and its sudo) runs once. The clone comes
# from the raw base, never the prebaked image that already holds Homebrew, and no
# /opt/homebrew is on the PATH the station is run with.
case_begin bare-mac-homebrew SU-08
tart stop "$erase_vm" >> "$log" 2>&1 || true
bare="station-e2e-macos-$run_id-bare"
made+=("$bare")
tart clone ghcr.io/cirruslabs/macos-tahoe-base:latest "$bare" >> "$log" 2>&1 || fail "clone the raw base"
tart set "$bare" --memory 8192 >> "$log" 2>&1 || fail "size $bare"
ip=$(boot "$bare") || fail "boot $bare"
bare_path='PATH=/usr/bin:/bin:/usr/sbin:/sbin'
in_vm 'sudo -n rm -rf /opt/homebrew /usr/local/Homebrew' || fail "make the base bare"
in_vm 'rm -rf /tmp/station-e2e && mkdir -p /tmp/station-e2e' || fail "scratch folder"
scp "${ssh_opts[@]}" -i "$ssh_key" -q "$repo/.artifacts/go/station" "$user@$ip:/tmp/station-e2e/station" || fail "copy the binary"
scp "${ssh_opts[@]}" -i "$ssh_key" -q "$repo/scripts/install.sh" "$user@$ip:/tmp/station-e2e/install.sh" || fail "copy install.sh"
in_vm 'test ! -e /opt/homebrew/bin/brew' || fail "the base still holds Homebrew"
in_vm "STATION_INSTALL_FROM=/tmp/station-e2e/station sh /tmp/station-e2e/install.sh </dev/null" >> "$log" 2>&1 || miss "install.sh exited $?"
in_vm "$bare_path $station apps install homebrew jq </dev/null" >> "$log" 2>&1 || miss "apps install homebrew jq exited $?: the manager installed in this run was not found in it"
in_vm 'test -x /opt/homebrew/bin/jq' || miss "no jq under /opt/homebrew after the install"
first=$(in_vm 'stat -f %m /opt/homebrew/bin/brew')
reply=$(in_vm "$bare_path $station apps install homebrew </dev/null" 2>&1)
say "   second run: $(tr '\n' ' ' <<< "$reply" | cut -c1-160)"
grep -q 'already installed' <<< "$reply" || miss "a second run does not find the Homebrew the first installed"
[ "$(in_vm 'stat -f %m /opt/homebrew/bin/brew')" = "$first" ] || miss "a second run ran Homebrew's installer again"
case_pass

# value: a Machinefile applied on a bare Mac: install.sh, then init --machinefile on a
# terminal (script gives it one), interrupted by SIGINT while the cask installs, then a
# bare apply that resumes it, drift with every package present, and the footprint test.
case_begin init-then-apply SU-12
tart stop "$bare" >> "$log" 2>&1 || true
applied="station-e2e-macos-$run_id-apply"
made+=("$applied")
tart clone ghcr.io/cirruslabs/macos-tahoe-base:latest "$applied" >> "$log" 2>&1 || fail "clone the raw base"
tart set "$applied" --memory 8192 >> "$log" 2>&1 || fail "size $applied"
ip=$(boot "$applied") || fail "boot $applied"
in_vm 'sudo -n rm -rf /opt/homebrew /usr/local/Homebrew' || fail "make the base bare"
in_vm 'rm -rf /tmp/station-e2e && mkdir -p /tmp/station-e2e' || fail "scratch folder"
scp "${ssh_opts[@]}" -i "$ssh_key" -q "$repo/.artifacts/go/station" "$user@$ip:/tmp/station-e2e/station" || fail "copy the binary"
scp "${ssh_opts[@]}" -i "$ssh_key" -q "$repo/scripts/install.sh" "$user@$ip:/tmp/station-e2e/install.sh" || fail "copy install.sh"
scp "${ssh_opts[@]}" -i "$ssh_key" -q "$suite/macos/e2e-apply.yaml" "$user@$ip:/tmp/station-e2e/e2e-apply.yaml" || fail "copy the Machinefile"
fp="$work/home-apply"
mkdir -p "$fp"
steps=()
n=0
listing > "$fp/0-before.txt"
step "install.sh" "STATION_INSTALL_FROM=/tmp/station-e2e/station sh /tmp/station-e2e/install.sh --name e2e </dev/null"
# hazard: sudo flushes the terminal's pending input when it prompts, so a stream of
# passwords loses characters; one line every few seconds lands whole on a prompt.
password="{ while :; do echo admin; sleep 3; done; }"
# invariant: apply asks Homebrew's password, so it runs on a terminal: script(1)
# gives it one; the run is interrupted once the cask's install has started. The
# shell script(1) runs writes its own pid, then becomes station, so SIGINT reaches
# station alone, never script(1), whose arguments name the same command.
in_vm "($password | script -q /tmp/station-e2e/first.log sh -c 'echo \$\$ > /tmp/station-e2e/station.pid; exec $station init --machinefile /tmp/station-e2e/e2e-apply.yaml' > /dev/null 2>&1; echo \$? > /tmp/station-e2e/first.exit) > /dev/null 2>&1 &" || fail "start init --machinefile"
# hazard: $out is run.sh's artifacts folder and every case shares one shell, so a case keeps its captured output in another name, or this copy lands in a path made of that output.
# value: the transcript is the only witness of an apply that ends early: it is copied
# to the host's artifacts before the VM is deleted, and its tail rides every failure.
first_log() {
  in_vm 'tr -d "\r" < /tmp/station-e2e/first.log 2> /dev/null' > "$out/$label-first.log" 2>&1 || true
}
waited=0
until in_vm 'pgrep -f "cask.*ghostty|install.*ghostty" > /dev/null'; do
  if in_vm 'test -s /tmp/station-e2e/first.exit'; then
    first_log
    fail "init --machinefile ended before the cask's install started, exit $(in_vm 'cat /tmp/station-e2e/first.exit'): $(grep -v '^$' "$out/$label-first.log" | tail -20)"
  fi
  waited=$((waited + 5))
  if [ $waited -ge 1200 ]; then
    first_log
    fail "the cask's install never started: $(grep -v '^$' "$out/$label-first.log" | tail -20)"
  fi
  sleep 5
done
in_vm 'kill -INT "$(cat /tmp/station-e2e/station.pid)"' || fail "interrupt the apply"
waited=0
until in_vm 'test -s /tmp/station-e2e/first.exit'; do
  waited=$((waited + 2))
  [ $waited -lt 300 ] || fail "the interrupted apply never ended"
  sleep 2
done
n=$((n + 1))
listing > "$fp/$n.txt"
steps+=("$n:init --machinefile, interrupted")
first_log
say "   interrupted run: $(in_vm 'tr -d "\r" < /tmp/station-e2e/first.log | grep -E "interrupted|not run" | head -3' | tr '\n' ' ')"
[ "$(in_vm 'cat /tmp/station-e2e/first.exit')" = 130 ] || miss "the interrupted apply exited $(in_vm 'cat /tmp/station-e2e/first.exit'), not 130"
in_vm 'grep -q "station apply resumes" /tmp/station-e2e/first.log' || miss "the interrupted apply did not say station apply resumes"
in_vm 'grep -q "e2e-apply" ~/.config/station/preferences.json' || miss "init --machinefile bound no Machinefile"
step "apply resumes" "$password | script -q /tmp/station-e2e/second.log $station apply > /dev/null"
drift=$(in_vm "$station drift --json")
say "   drift: $(jq -c '[.lines[] | [.package, .condition]]' <<< "$drift")"
[ "$(jq '[.lines[] | select(.condition != "present")] | length' <<< "$drift")" = 0 ] || miss "drift is not clean after the resumed apply"
in_vm 'cat ~/.local/state/station/ledger.json' > "$fp/ledger.json" || miss "apply kept no ledger"
[ "$(jq -r '[.installed[].package] | sort | join(" ")' "$fp/ledger.json")" = "ghostty homebrew jq" ] || miss "the ledger holds other than what apply installed"
in_vm "$station disk --json" > "$fp/disk.json" || fail "disk --json"
footprint "$fp" "$fp/disk.json" "$suite/macos/os-noise.txt" || fail "the footprint test"
case_pass

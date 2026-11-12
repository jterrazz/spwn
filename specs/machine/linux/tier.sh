#!/bin/bash
# value: tier 2: the engine of this build runs real Linux VMs on this Mac, from a Mac home
# of its own, and a second home plays another Mac that reaches them over a real ssh
# hop. Sourced by run.sh.
# invariant: the two homes are folders of this run, never the owner's: their PATH
# holds their own bin and the system's, their Power is the stub, their system root
# a folder of the run, so the owner's Tailscale, keychain and pmset are never
# reached; their tart keeps its VMs in the run's own store.
# outside: the tailnet, the credential store and Claude's sign-in of the Mac are
# stubs (tailscale, studio, claude in the Mac's bin): what a VM receives is real.

mac="$work/mac"
far="$work/far"
real_tart=""
for d in "$HOME/.local/share/station/tart/tart.app" "$HOME/.local/opt/tart.app"; do
  [ -x "$d/Contents/MacOS/tart" ] && { real_tart=$d; break; }
done
[ -n "$real_tart" ] || fail "tart is not installed: station apps install tart"
real_cache=""
for d in "$HOME/.local/share/station/vms/cache" "$HOME/.tart/cache"; do
  [ -d "$d" ] && { real_cache=$d; break; }
done

home_of() {
  local h=$1 name=$2
  mkdir -p "$h/.local/bin" "$h/.local/share/station/tart" "$h/.local/share/station/vms" \
    "$h/.config/station" "$h/power" "$h/root"
  ln -s "$real_tart" "$h/.local/share/station/tart/tart.app"
  # value: the OCI cache is the owner's, read when the base image is already
  # pulled; the VMs and images this run makes stay in its own store, and tart's
  # automatic pruning is off (TART_NO_AUTO_PRUNE in as), or a clone of this run
  # deletes the owner's other base images to make room.
  [ -z "$real_cache" ] || ln -s "$real_cache" "$h/.local/share/station/vms/cache"
  cp "$repo/src/catalog/dotfiles/tart/tart" "$h/.local/bin/tart"
  cp "$repo/.artifacts/go/station" "$h/.local/bin/station"
  cp "$suite/linux/stubs/"* "$h/.local/bin/"
  chmod +x "$h/.local/bin/"*
  printf '{"here":"%s","vm_user":"e2e"}\n' "$name" > "$h/.config/station/preferences.json"
}
as() {
  local h=$1
  shift
  env -i HOME="$h" PATH="$h/.local/bin:/usr/bin:/bin:/usr/sbin:/sbin" TERM=dumb TZ=UTC NO_COLOR=1 \
    TART_NO_AUTO_PRUNE=1 STATION_MACHINE_KIND=mac STATION_POWER_STUB="$h/power" STATION_SYSTEM_ROOT="$h/root" "$@"
}
# value: as, in place of the caller: a process started in the background is then
# the one its pid names, and the cleanup's kill reaches it.
as_exec() {
  local h=$1
  shift
  exec env -i HOME="$h" PATH="$h/.local/bin:/usr/bin:/bin:/usr/sbin:/sbin" TERM=dumb TZ=UTC NO_COLOR=1 \
    TART_NO_AUTO_PRUNE=1 STATION_MACHINE_KIND=mac STATION_POWER_STUB="$h/power" STATION_SYSTEM_ROOT="$h/root" "$@"
}
on_mac() { as "$mac" "$@"; }
on_far() { as "$far" "$@"; }
st() { on_mac station "$@"; }

engine_pid=""
sshd_pid=""
cleanup_tier() {
  [ -d "$mac/.local/share/station/vms" ] || return 0
  local vm
  for vm in $(on_mac tart list --quiet --source local 2>/dev/null); do
    on_mac tart stop "$vm" >/dev/null 2>&1 || true
    on_mac tart delete "$vm" >/dev/null 2>&1 || true
  done
  [ -z "$engine_pid" ] || kill "$engine_pid" 2>/dev/null || true
  [ -z "$sshd_pid" ] || kill "$sshd_pid" 2>/dev/null || true
  wait 2>/dev/null
  rm -rf "$mac/.local/share/station/vms" "$mac/.local/share/station/images" "$mac/.cache"
}

# value: ssh into a VM of the Mac as the image's login, through the product's own
# pipe: the engine's address, never a guess.
key="$mac/.local/state/station/images/key"
vm_ssh() {
  local vm=$1
  shift
  on_mac /usr/bin/ssh "${ssh_opts[@]}" -i "$key" -o ProxyCommand="$mac/.local/bin/station machines pipe $vm" "e2e@$vm" "$@"
}
in_vm() { vm_ssh "$1" "$2" < /dev/null; }

# value: a guest's /proc files as a tar, the recorded inputs of the D specs; each is
# copied first, because /proc reports a size of 0 and tar would archive it empty.
capture() { echo "rm -rf /tmp/e2e-cap && for f in $*; do mkdir -p /tmp/e2e-cap/\${f%/*} && cat /\$f > /tmp/e2e-cap/\$f; done; tar cf - -C /tmp/e2e-cap proc"; }

until_true() {
  local tries=$1
  shift
  local i
  for i in $(seq 1 "$tries"); do
    "$@" && return 0
    sleep 5
  done
  return 1
}

home_of "$mac" macbook-e2e
on_mac tart --version > /dev/null || fail "tart does not answer from the run's Mac home"
(as_exec "$mac" station engine run) >> "$work/engine.log" 2>&1 &
engine_pid=$!
until_true 6 st engine --json > /dev/null 2>&1 || fail "the engine of this build does not answer"
say "-- the engine of this build runs for $mac (pid $engine_pid)"


case_begin image-build IM-09
st images build linux --binary "$repo/.artifacts/go/station-linux-arm64" >> "$log" 2>&1 || fail "images build linux exited $?"
built=$(st images --json | jq -r '[.images[] | select(.name == "linux") | .version] | join(" ")')
[ -n "$built" ] || fail "images lists no linux version"
on_mac tart list --quiet --source local | grep -q '^station-image-linux-' || fail "no station-image-linux-* in the store"
st machines create ws1 >> "$log" 2>&1 || fail "machines create ws1 exited $?"
kind_inside=$(st overview --on ws1 --json | jq -r '.machine.kind')
[ "$kind_inside" = workspace ] || fail "station inside ws1 reports $kind_inside, not workspace"
say "   linux $built boots, its guest answers, station inside reports workspace"
case_pass


case_begin create-workspace CR-23
health=$(st machines ws1 --json | jq -c '.health')
[ "$(jq -r '.running and .address and .ssh' <<< "$health")" = true ] || fail "ws1's health: $health"
[ "$(in_vm ws1 hostname)" = ws1 ] || fail "ssh through the pipe does not reach ws1"
listing() { in_vm ws1 'cd ~ && find . -mindepth 1 -maxdepth 4 2>/dev/null | sed "s|^\./||" | LC_ALL=C sort'; }
fp="$work/ws1"
mkdir -p "$fp"
listing > "$fp/0-before.txt" || fail "list ws1's home"
steps=()
n=0
step() {
  local name=$1 code
  shift
  n=$((n + 1))
  say "-- step $n: $name"
  "$@" >> "$log" 2>&1
  code=$?
  listing > "$fp/$n.txt"
  steps+=("$n:$name")
  [ $code -eq 0 ] || fail "step $n ($name) exited $code"
}
mise_path='PATH=$HOME/.local/share/mise/shims:$HOME/.local/bin:$PATH'
while read -r app; do
  case "$app" in '' | '#'*) continue ;; esac
  step "apps install $app" st apps install "$app" --on ws1
done < "$suite/linux/one-per-method.txt"
step "go build" in_vm ws1 "$mise_path; mkdir -p /tmp/e2e/hello && cd /tmp/e2e/hello && printf 'module hello\n\ngo 1.21\n' > go.mod && printf 'package main\n\nfunc main() { println(\"hello\") }\n' > main.go && go build -o hello . && ./hello"
step "npm i" in_vm ws1 "$mise_path; mkdir -p /tmp/e2e/npm && cd /tmp/e2e/npm && printf '{\"name\":\"e2e\",\"private\":true}\n' > package.json && npm i is-number@7.0.0"
step "bun add -g" in_vm ws1 "$mise_path; bun add -g is-number@7.0.0"
step "uv tool install" in_vm ws1 "$mise_path; uv tool install pyflakes==3.2.0"
step "zsh -ic" in_vm ws1 "zsh -ic 'command -v station && command -v mise && go env GOPATH'"
in_vm ws1 'test -e ~/.bun' && fail "~/.bun present after bun add -g: bun ignored the bunfig"
in_vm ws1 "$mise_path; station disk --json" > "$fp/disk.json" || fail "disk --json"
footprint "$fp" "$fp/disk.json" "$suite/linux/os-noise.txt" || fail "the footprint test"
st disk free --on ws1 >> "$log" 2>&1 || fail "disk free exited $?"
left=$(st disk --on ws1 --json | jq -r '.rows[] | select(.group == "can-rebuild" and .bytes > 0) | "\(.app) \(.bytes)"')
[ -z "$left" ] || fail "disk free left: $left"
say "   ws1 answers ssh, its apps install through the engine, its home holds only the footprint"
case_pass


case_begin apps-every-linux-app AP-25 ADR-038 ADR-040
# value: a workspace of its own, deleted after, so every app installs on a fresh home
# and none leans on what ws1 already holds.
st machines create apps1 >> "$log" 2>&1 || fail "machines create apps1 exited $?"
guest_path='PATH=$HOME/.local/share/mise/shims:$HOME/.local/bin:$PATH'
while read -r app commands; do
  case "$app" in '' | '#'*) continue ;; esac
  st apps install "$app" --on apps1 >> "$log" 2>&1 || { miss "apps install $app exited $?"; continue; }
  state=$(st apps --all --on apps1 --json | jq -r --arg a "$app" '.apps[] | select(.name == $a) | .state')
  say "   $app: $state"
  case "$state" in not-installed | failed | not-known | "") miss "$app reads \"$state\" after its install" ;; esac
  for command in $commands; do
    in_vm apps1 "$guest_path; command -v $command" > /dev/null || miss "$app: $command is not on the guest's PATH"
  done
done < "$suite/linux/every-app.txt"
st machines delete apps1 --confirm=apps1 >> "$log" 2>&1 || on_mac tart delete apps1 >> "$log" 2>&1
case_pass


case_begin docker-shortcuts SE-28
st settings apply docker-shortcuts --on ws1 >> "$log" 2>&1 || fail "settings apply exited $?"
[ "$(in_vm ws1 "zsh -ic 'whence -w dclean'" 2>/dev/null)" = "dclean: function" ] || fail "dclean is not a function of the zsh block"
asked=$(echo n | vm_ssh ws1 "zsh -ic dclean" 2>&1)
code=$?
grep -q '\[y/N\]' <<< "$asked" || fail "dclean asked nothing: $asked"
[ $code -ne 0 ] || fail "dclean answered no and still exited 0"
say "   dclean asks before removing: $(tr '\n' ' ' <<< "$asked")"
case_pass


case_begin claude-token-share AU-13
# outside: how the studio CLI reaches a VM is not designed (docs/reference/accounts.md),
# so the VM's credential store is the stub's file store.
vm_ssh ws1 'cat > /tmp/studio && sudo -n install -m 755 /tmp/studio /usr/local/bin/studio' < "$suite/linux/vm-studio" || fail "put the VM's studio"
st accounts share claude --with ws1 >> "$log" 2>&1 || fail "accounts share exited $?"
held=$(in_vm ws1 "zsh -lic 'echo \${#CLAUDE_CODE_OAUTH_TOKEN}'" 2>/dev/null | tail -1)
placeholder=$(sed -n 's/.*"\(sk-ant-[^"]*\)".*/\1/p' "$suite/linux/stubs/studio")
[ "$held" = "${#placeholder}" ] || fail "the VM's shell exports no token of the shared length: $held"
st accounts take-back claude --from ws1 --yes >> "$log" 2>&1 || fail "accounts take-back exited $?"
held=$(in_vm ws1 "zsh -lic 'echo \${#CLAUDE_CODE_OAUTH_TOKEN}'" 2>/dev/null | tail -1)
[ "$held" = 0 ] || fail "the token is still exported after take-back"
say "   the VM's shell exports CLAUDE_CODE_OAUTH_TOKEN once shared and not once taken back"
say "   owner-gated: \`claude\` answering with the owner's real token (no studio in a VM yet)"
case_pass


case_begin server-check-self-signed KP-21
st machines create srv1 --server --check https://srv1:8443/readyz --cpus 2 --memory 2G --disk 20G >> "$log" 2>&1 || fail "create srv1 exited $?"
vm_ssh srv1 'mkdir -p /tmp/e2e-tls && cd /tmp/e2e-tls && openssl req -x509 -newkey rsa:2048 -nodes -keyout key.pem -out cert.pem -days 1 -subj /CN=cluster-ca 2>/dev/null && cat > https.py && (nohup python3 https.py > /dev/null 2>&1 &)' < "$suite/linux/https.py" || fail "serve readyz in srv1"
check_ok() { [ "$(st machines srv1 --json | jq -r '.health.check.ok')" = "$1" ]; }
until_true 24 check_ok true || fail "the check on srv1's own name never passed with its self-signed certificate"
mkdir -p "$work/proc-capture" && in_vm srv1 "$(capture proc/net/tcp proc/net/tcp6)" > "$work/proc-capture/tcp.tar"
addr=$(on_mac tart ip srv1) || fail "srv1 has no address"
st machines edit srv1 --check "https://$addr:8443/readyz" >> "$log" 2>&1 || fail "edit srv1's check exited $?"
until_true 24 check_ok false || fail "a check on another name ($addr) passed with a certificate not made for it"
st machines delete srv1 --confirm=srv1 >> "$log" 2>&1 || on_mac tart delete srv1 >> "$log" 2>&1
say "   own name: passes with the cluster's certificate; https://$addr: fails"
case_pass


case_begin on-through-hop MA-15 RM-20
home_of "$far" laptop-e2e
hop="$far/hop"
mkdir -p "$hop"
ssh-keygen -q -t ed25519 -N '' -C station-e2e -f "$hop/host_key" || fail "ssh-keygen"
ssh-keygen -q -t ed25519 -N '' -C station-e2e -f "$hop/client_key" || fail "ssh-keygen"
cp "$hop/client_key.pub" "$hop/authorized_keys"
chmod 600 "$hop/authorized_keys"
port=$((22000 + RANDOM % 1000))
# invariant: the far Mac's sshd is this user's, on loopback, and every session it
# opens runs in the Mac home of this run, never the owner's.
cat > "$hop/session" <<EOF
#!/bin/sh
exec env -i HOME="$mac" PATH="$mac/.local/bin:/usr/bin:/bin:/usr/sbin:/sbin" TERM=dumb TZ=UTC NO_COLOR=1 \
  STATION_MACHINE_KIND=mac STATION_POWER_STUB="$mac/power" STATION_SYSTEM_ROOT="$mac/root" /bin/sh -c "\$SSH_ORIGINAL_COMMAND"
EOF
chmod +x "$hop/session"
cat > "$hop/sshd_config" <<EOF
Port $port
ListenAddress 127.0.0.1
HostKey $hop/host_key
AuthorizedKeysFile $hop/authorized_keys
PasswordAuthentication no
KbdInteractiveAuthentication no
UsePAM no
StrictModes no
PidFile $hop/sshd.pid
ForceCommand $hop/session
EOF
/usr/sbin/sshd -D -f "$hop/sshd_config" -E "$hop/sshd.log" &
sshd_pid=$!
cat > "$hop/ssh_config" <<EOF
Host macbook-e2e
  HostName 127.0.0.1
  Port $port
  IdentityFile $hop/client_key
  IdentitiesOnly yes
  IdentityAgent none
  StrictHostKeyChecking no
  UserKnownHostsFile /dev/null
  LogLevel ERROR
EOF
# value: ssh reads the config of the user's real home, never $HOME's: the far
# Mac's ssh is this one config.
printf '#!/bin/sh\nexec /usr/bin/ssh -F %s "$@"\n' "$hop/ssh_config" > "$far/.local/bin/ssh"
chmod +x "$far/.local/bin/ssh"
until_true 6 on_far ssh macbook-e2e true < /dev/null || fail "the far Mac's sshd does not answer"
on_far station machines add macbook-e2e < /dev/null >> "$log" 2>&1 || fail "machines add exited $?"
remote=$(on_far station overview --on ws1 --json < /dev/null | jq -r '"\(.machine.kind) on \(.machine.on)"')
[ "$remote" = "workspace on macbook-e2e" ] || fail "--on ws1 from the other Mac answered: $remote"
st machines ws1 --json > /dev/null
on_far station machines --json < /dev/null > /dev/null
own_config="$hop/owner_config"
sed "s|~/.ssh/id_ed25519|$key|; s|UserKnownHostsFile .*|UserKnownHostsFile /dev/null|" "$far/.ssh/config" > "$own_config"
grep -q 'ProxyCommand ssh macbook-e2e' "$own_config" || fail "the other Mac's ssh block for ws1 has no hop"
owner_ssh() { on_far /usr/bin/ssh -F "$own_config" -o IdentitiesOnly=yes -o IdentityAgent=none -o BatchMode=yes -o LogLevel=ERROR -o ConnectTimeout=5 ws1 "$@" < /dev/null; }
before=$(owner_ssh 'cat /proc/sys/kernel/random/boot_id; hostname -I | cut -d" " -f1') || fail "ssh ws1 through the hop"
st machines stop ws1 --yes >> "$log" 2>&1 || fail "stop ws1 exited $?"
on_mac tart set ws1 --random-mac >> "$log" 2>&1 || fail "give ws1 a new MAC address"
st machines start ws1 >> "$log" 2>&1 || fail "start ws1 exited $?"
moved() { after=$(owner_ssh 'cat /proc/sys/kernel/random/boot_id; hostname -I | cut -d" " -f1' 2>/dev/null) && [ -n "$after" ] && [ "$(tail -1 <<< "$after")" != "$(tail -1 <<< "$before")" ]; }
until_true 36 moved || fail "after a new address, ssh ws1 through the hop did not reach it: before $(tail -1 <<< "$before"), after ${after:-nothing}"
say "   --on ws1 from the other Mac in BatchMode; ssh ws1 hops at $(tail -1 <<< "$before"), then at $(tail -1 <<< "$after")"
case_pass

case_begin activity-under-load AC-29
proc() { in_vm ws1 'cat /proc/meminfo /proc/net/dev /proc/diskstats; nproc'; }
# value: a sample the guest measured, never the Mac's view of the VM process a
# guest that is reconnecting falls back to.
sample() {
  local i
  for i in $(seq 1 12); do
    st activity --on ws1 --json > "$work/sample.json" && [ "$(jq -r .source "$work/sample.json")" = direct ] && { cat "$work/sample.json"; return 0; }
    sleep 2
  done
  cat "$work/sample.json"
  return 1
}
check_proc() {
  local a=$1 p=$2 total avail rx wr
  total=$(awk '/^MemTotal:/ { print $2 * 1024 }' "$p")
  avail=$(awk '/^MemAvailable:/ { print $2 * 1024 }' "$p")
  rx=$(awk '$1 ~ /^(en|eth)/ { s += $2 } END { print s }' "$p")
  wr=$(awk '$3 ~ /^vd[a-z]$/ { s += $10 * 512 } END { print s }' "$p")
  [ "$(jq -r '.cpu.cpus' "$a")" = "$(tail -1 "$p")" ] || { say "   cpus $(jq -r '.cpu.cpus' "$a") vs nproc $(tail -1 "$p")"; return 1; }
  [ "$(jq -r '.memory.total_bytes' "$a")" = "$total" ] || { say "   memory total differs from /proc"; return 1; }
  near "$(jq -r '.memory.used_bytes' "$a")" $((total - avail)) 5 67108864 || { say "   memory used $(jq -r '.memory.used_bytes' "$a") vs /proc $((total - avail))"; return 1; }
  near "$(jq -r '.network.received_since_boot_bytes' "$a")" "$rx" 2 1048576 || { say "   received $(jq -r '.network.received_since_boot_bytes' "$a") vs /proc $rx"; return 1; }
  near "$(jq -r '.disk.written_since_boot_bytes' "$a")" "$wr" 2 1048576 || { say "   written $(jq -r '.disk.written_since_boot_bytes' "$a") vs /proc $wr"; return 1; }
}
sample > "$work/act-0.json" && proc > "$work/proc-0.txt" || fail "read ws1's activity"
check_proc "$work/act-0.json" "$work/proc-0.txt" || fail "idle: station's numbers do not match /proc"
# value: the load lasts 40 s, the CPU's overlapping the disk's and the network's: the
# dd of 1.5 GB is done in 2 s, and the 400 MB received are sent by this Mac over
# ssh, 10 MB a quarter second, since speed.cloudflare.com refuses calls of a burst.
vm_ssh ws1 'setsid -f sh -c "yes > /dev/null & yes > /dev/null & end=\$((\$(date +%s) + 40)); dd if=/dev/zero of=/tmp/e2e-dd bs=1M count=1536 oflag=direct status=none; rm -f /tmp/e2e-dd; while [ \$(date +%s) -lt \$end ]; do sleep 1; done; pkill yes" > /dev/null 2>&1 &' < /dev/null
(for i in $(seq 1 40); do head -c 10000000 /dev/zero; sleep 0.25; done | vm_ssh ws1 'cat > /dev/null') &
feed=$!
busy=0
for i in $(seq 1 12); do
  sleep 2
  sample > "$work/act-load.json" || continue
  jq -e '.cpu.busy_percent >= 40 and ((.disk.write_bytes_per_s > 20000000) or (.network.in_bytes_per_s > 10000000))' "$work/act-load.json" > /dev/null && { busy=1; break; }
done
[ $busy -eq 1 ] || fail "under load: no sample shows CPU over 40 % with disk or network over its idle rate"
jq -r '.top.by_cpu[0].name' "$work/act-load.json" | grep -qx yes || say "   note: the busiest process read was $(jq -r '.top.by_cpu[0].name' "$work/act-load.json")"
until_true 24 in_vm ws1 '! pgrep -x yes > /dev/null' || fail "the load did not end"
wait "$feed" || fail "the 400 MB sent to ws1 over ssh did not all arrive"
mkdir -p "$work/proc-capture"
in_vm ws1 "$(capture proc/stat proc/meminfo proc/net/dev proc/diskstats proc/loadavg proc/uptime proc/pressure/memory)" > "$work/proc-capture/proc.tar"
sample > "$work/act-1.json" && proc > "$work/proc-1.txt" || fail "read ws1's activity after the load"
check_proc "$work/act-1.json" "$work/proc-1.txt" || fail "after the load: station's numbers do not match /proc"
grew() { jq -n --slurpfile a "$work/act-0.json" --slurpfile b "$work/act-1.json" "\$b[0].$1 - \$a[0].$1"; }
[ "$(grew disk.written_since_boot_bytes)" -ge 1500000000 ] || fail "written grew $(grew disk.written_since_boot_bytes) under a 1.5 GB dd"
[ "$(grew network.received_since_boot_bytes)" -ge 400000000 ] || fail "received grew $(grew network.received_since_boot_bytes) under 400 MB sent"
say "   matches /proc idle and after; written +$(grew disk.written_since_boot_bytes) B, received +$(grew network.received_since_boot_bytes) B"
case_pass



case_begin rights-private ADR-035
# value: ws1's Machinefile opens 3000-3999 to private, read at its next run:
# 3000 is shared on a port of the Mac and served on the stub tailnet, 8443
# is flagged and never reaches it.
mkdir -p "$mac/.config/station/machinefiles"
printf 'machinefile: station.machinefile/v1\nname: ws1\nmachine: { kind: workspace }\nports: { open: 3000-3999, to: private }\n' > "$mac/.config/station/machinefiles/ws1.yaml"
st machines restart ws1 >> "$log" 2>&1 || fail "machines restart ws1 exited $?"
in_vm ws1 '(nohup python3 -m http.server 3000 > /dev/null 2>&1 &); (nohup python3 -m http.server 8443 > /dev/null 2>&1 &)' || fail "start ws1's two servers"
state_of() { st ports --json | jq -r --argjson p "$1" '[.ports[] | select(.machine == "ws1" and .port == $p) | .state] | first // "none"'; }
is_state() { [ "$(state_of "$1")" = "$2" ]; }
until_true 24 is_state 3000 shared || fail "3000, inside the rights, is $(state_of 3000), not shared"
until_true 6 is_state 8443 not-allowed || fail "8443, outside the rights, is $(state_of 8443), not flagged"
on=$(st ports --json | jq -r '[.ports[] | select(.machine == "ws1" and .port == 3000) | .reserved] | first')
grep -q -- "serve --bg --https=$on http://127.0.0.1:$on" "$mac/.calls" || miss "3000 was never served on the tailnet at $on"
grep -q -- "8443" "$mac/.calls" && miss "8443 reached the tailnet"
curl -fsS -o /dev/null "http://127.0.0.1:$on/" || miss "3000 does not answer on the Mac's loopback port $on"
rm -f "$mac/.config/station/machinefiles/ws1.yaml"
say "   3000 shared on $on and served; 8443 flagged, never served"
case_pass

st machines delete ws1 --confirm=ws1 >> "$log" 2>&1 || true

#!/bin/bash
set -u
cd "$(dirname "$0")" || exit 2
fail=0; TAG=; bad() { echo "FAIL${TAG:+ [$TAG]} $*"; fail=1; }
for f in *.sh; do bash -n "$f" || bad "syntax $f"; done
python3 -I -c 'import ast, sys; ast.parse(open(sys.argv[1]).read())' cc-verdict.py || bad "syntax cc-verdict.py"
grep -nE '/private/tmp|/tmp/claude' ./*.sh ./*.py | grep -v '^./test.sh' && bad "a script names a scratchpad"
T=$(mktemp -d) || exit 2; trap 'rm -rf "$T"' EXIT; T=$(cd -P "$T" && pwd -P)
d=$(env -u GATE_RUN_DIR bash -c '. ./env.sh; echo "$RUN"'); case $d in */.artifacts/gates) ;; *) bad "default run folder is $d" ;; esac
d=$(GATE_RUN_DIR=$T/run bash -c '. ./env.sh; echo "$RUN"'); [ "$d" = "$T/run" ] || bad "GATE_RUN_DIR is ignored: $d"
d=$(env -u GATE_VM_RAM GATE_RUN_DIR=$T/run GATE_VM_DIR=g bash -c '. ./env.sh; echo "$VM_RAM"'); [ "$d" = /gates/g ] || bad "default RAM folder is $d"
d=$(GATE_RUN_DIR=$T/run GATE_VM_RAM=/r bash -c '. ./env.sh; echo "$VM_RAM"'); [ "$d" = /r ] || bad "GATE_VM_RAM is ignored: $d"
grep -nE 'git clone' ./*.sh | grep -v '^./test.sh' | grep -vE 'git clone -q -b \S+ \S+ ("\$WT"|\$C|\$W/)' && bad "a clone lands outside the RAM folder"
grep -noE '(^|[ ;])(WT|C)=[^ ;]+' ./*.sh | grep -v '^./test.sh' | grep -vE '=\$(W|VM_RAM)/' && bad "a clone folder is not under the RAM folder"
bare_ssh() {
  local f n l s c re='(^|[^A-Za-z0-9_.-])(ssh|scp|sftp|rsync)($|[^A-Za-z0-9_-])' comment='^[[:space:]]*#'
  local exempt='^(.*[^[:space:]])[[:space:]]+# gate-ssh: exempt \([^()#]+\)$' hash='(^|[[:space:]])#'
  for f in "$@"; do
    n=0
    while IFS= read -r l || [ -n "$l" ]; do
      n=$((n + 1))
      [[ $l =~ $comment ]] && continue
      s=${l//'"$GATE_SSH" ssh'/}; s=${s//'"$GATE_SSH" scp'/}; s=${s//'"$GATE_SSH" rsh'/}; s=${s//'rsync -e "$GATE_SSH rsh"'/}
      [[ $s =~ $re ]] || continue
      if [[ $l =~ $exempt ]]; then
        c=${BASH_REMATCH[1]}
        [[ ! $c =~ $hash && $c == *'-F /dev/null'* && $c == *'IdentityAgent=none'* ]] && continue
      fi
      echo "$f:$n: $l"
    done < "$f"
    echo "read $f $n"
  done
}
F=$(mktemp -d) || exit 2
bare_lines=('ssh vm true' '  scp x vm:y' 'a && ssh -n vm true' 'x=$(ssh vm true)' 'S="ssh -n"' 'rsync -a a vm:b' 'if ssh vm true; then'
  'command ssh vm true' 'env ssh vm true' 'exec ssh vm true' 'nohup ssh vm true' 'timeout 5 ssh vm true' 'echo vm | xargs ssh' '/usr/bin/ssh vm true'
  'while ssh vm true; do :; done' 'x=`ssh vm true`' 's=ssh; $s vm true' 'rsync -e "$GATE_SSH rsh" a vm:b; ssh vm true' '"$GATE_SSH" ssh vm true; scp a vm:b'
  'sftp vm' 'rsync -a -e ssh a vm:b' 'echo "(hold or ssh)"' 'ssh vm true  # gate-ssh: exempt (no guard)' 'ssh -F /dev/null vm true  # gate-ssh: exempt (half a guard)'
  'ssh -F /dev/null vm true # -o IdentityAgent=none # gate-ssh: exempt (the options in a comment)' 'ssh vm true  # gate-ssh: exempt (x) -F /dev/null -o IdentityAgent=none'
  'ssh -F /dev/null -o IdentityAgent=none vm true; x=" # gate-ssh: exempt (in a string)"; ssh vm true')
printf '%s\n' "${bare_lines[@]}" > "$F/bare.sh"
printf '%s\n' '"$GATE_SSH" ssh vm true' '# ssh vm true' 'ssh -F /dev/null -o IdentityAgent=none vm true  # gate-ssh: exempt (why)' 'rsync -e "$GATE_SSH rsh" a vm:b' 'SSHP=("$GATE_SSH" ssh -n)' 'ssh-keygen -l -f k' 'cat ~/.ssh/known_hosts' > "$F/clean.sh"
tree=(); for f in ./*.sh ../*.sh ../../machines/apps/*/scripts/*.sh; do case $f in ./gate-ssh.sh | ./test.sh) ;; *) tree+=("$f") ;; esac; done
[ ${#tree[@]} -ge 20 ] || bad "the bare-ssh lint reads only ${#tree[@]} scripts"
cp ./cc.sh "$F/cc-sentinel.sh"; printf '%s\n' 'ssh "$VM" true' >> "$F/cc-sentinel.sh"; last=$(wc -l < "$F/cc-sentinel.sh")
lint=$(bare_ssh "$F/bare.sh" "$F/clean.sh" "$F/cc-sentinel.sh" "${tree[@]}")
n=$(grep -cF "$F/bare.sh:" <<< "$lint"); [ "$n" = ${#bare_lines[@]} ] || bad "the bare-ssh lint misses a form: $n of ${#bare_lines[@]} fixture lines flagged"
[ "$(grep -F "$F/cc-sentinel.sh:" <<< "$lint" | cut -d: -f2)" = "$((last))" ] || bad "the bare-ssh lint does not flag exactly the sentinel line of a real script: $(grep -F "$F/cc-sentinel.sh:" <<< "$lint")"
[ "$(grep -cE '^read \.\.?/' <<< "$lint")" = ${#tree[@]} ] || bad "the bare-ssh lint read $(grep -cE '^read \.\.?/' <<< "$lint") of the ${#tree[@]} scripts of the tree"
for k in ./cc.sh ./cold-gate.sh ./desktop-proof.sh ./regen.sh ./env.sh ./darwin-build.sh ../install.sh ../../machines/apps/desktop/scripts/look.sh; do
  grep -qE "^read ${k//./\\.} [1-9][0-9]*$" <<< "$lint" || bad "the bare-ssh lint did not read $k"
done
rest=$(grep -vF "$F/" <<< "$lint" | grep -v '^read '); [ -z "$rest" ] || bad "bare ssh to a VM (a script reaches a VM only through \"\$GATE_SSH\"): $rest"
rm -rf "$F"
export GATE_RUN_DIR=$T/slot GATE_SLOT_POLL=1 GATE_SLOT_MAX_LOAD=100000
unset GATE_SLOT_VM
: > "$T/order"
bash ./gate-slot.sh sleep 4 > "$T/holder.log" 2>&1 &
held=0; for _ in $(seq 50); do grep -q taken "$T/holder.log" && { held=1; break; }; command sleep 0.1; done
[ $held = 1 ] || bad "the first waiter never took the slot"
for p in low normal landing; do
  GATE_SLOT_PRIORITY=$p bash ./gate-slot.sh sh -c "echo $p >> $T/order" > "$T/$p.log" 2>&1 &
  command sleep 0.3
done
wait
[ "$(tr '\n' ' ' < "$T/order")" = "landing normal low " ] || bad "slot order: $(tr '\n' ' ' < "$T/order")"
GATE_SLOT_PRIORITY=urgent bash ./gate-slot.sh true > /dev/null 2>&1; [ $? = 2 ] || bad "an unknown priority is accepted"
[ -z "$(ls "$T/slot/gate-slots/wanted" 2>/dev/null)" ] || bad "a waiter left its mark"
unset GATE_RUN_DIR GATE_SLOT_POLL GATE_SLOT_MAX_LOAD
umask 077
H=$T/home; R=$T/routes; mkdir -p "$H/.ssh" "$H/k" "$H/bin" "$R"
: > "$H/k/vm_key"; : > "$H/k/hop_key"; : > "$H/.ssh/id_ed25519"; : > "$H/.ssh/other_key"
ln -s "$H/k/vm_key" "$H/k/link_key"; ln "$H/.ssh/other_key" "$H/k/hard_key"
mkdir -p "$H/.ssh/sub/deep"; : > "$H/.ssh/sub/deep/id2"; : > "$H/.ssh/.hidden"; ln "$H/.ssh/sub/deep/id2" "$H/k/hard_sub"; ln "$H/.ssh/.hidden" "$H/k/hard_dot"
mkdir -p "$H/.ssh/a/b/c/d/e"; : > "$H/.ssh/a/b/c/d/e/id_deep"; ln "$H/.ssh/a/b/c/d/e/id_deep" "$H/k/hard_deep"; ln -s "$H/.ssh" "$H/sshalias"
printf '#!/bin/sh\nenv > %s\n' "$T/proxy.env" > "$H/bin/station"; chmod 755 "$H/bin/station"
mkdir -p "$H/nx" "$H/ww" "$H/wd" "$H/st" "$H/Apps/Spwn.app/Contents/MacOS" "$H/Other"; : > "$H/nx/station"
cp "$H/bin/station" "$H/ww/station"; chmod 777 "$H/ww/station"; cp "$H/bin/station" "$H/wd/station"; chmod 777 "$H/wd"
cp "$H/bin/station" "$H/Apps/Spwn.app/Contents/MacOS/Spwn Engine"; cp "$H/bin/station" "$H/Other/Spwn Engine"
mkdir -p "$H/.ssh/bin" "$H/hl"; cp "$H/bin/station" "$H/.ssh/bin/station"; ln "$H/.ssh/bin/station" "$H/hl/station"
ln -s /usr/bin/env "$H/st/station"; ln -s "$H/Apps/Spwn.app/Contents/MacOS/Spwn Engine" "$H/st/app"; ln -s "$H/Other/Spwn Engine" "$H/st/other"
eval "pw=~$(id -un)"
route_bare() { local n=$1; shift; printf '%s\n' "$@" > "$R/$n.env"; chmod 600 "$R/$n.env"; }
route() { local n=$1 a kh="KNOWN_HOSTS=$H/kh"; shift; for a in "$@"; do case $a in KNOWN_HOSTS=*) kh= ;; esac; done; route_bare "$n" "$@" ${kh:+"$kh"}; }
route_bare vm-nokh "KEY=$H/k/vm_key"
route_bare vm-hopnokh HOP=ops@hop-host "HOP_KEY=$H/k/hop_key"
: > "$H/.ssh/known_hosts"; route vm-kh-ssh "KEY=$H/k/vm_key" "KNOWN_HOSTS=$H/.ssh/known_hosts"
ln "$H/.ssh/known_hosts" "$H/kh-hard"; route vm-kh-hard "KEY=$H/k/vm_key" "KNOWN_HOSTS=$H/kh-hard"
route vm-kh-alias "KEY=$H/k/vm_key" "KNOWN_HOSTS=$H/sshalias/known_hosts"
route vm-key-alias "KEY=$H/sshalias/gate-ssh-absent-key"
: > "$H/kh-ww"; chmod 666 "$H/kh-ww"; route vm-kh-ww "KEY=$H/k/vm_key" "KNOWN_HOSTS=$H/kh-ww"
route vm-kh-root "KEY=$H/k/vm_key" KNOWN_HOSTS=/etc/passwd
ln -s "$H/kh" "$H/kh-link"; route vm-kh-link "KEY=$H/k/vm_key" "KNOWN_HOSTS=$H/kh-link"
mkdir -p "$H/khd"; : > "$H/khd/kh"; chmod 777 "$H/khd"; route vm-kh-dir "KEY=$H/k/vm_key" "KNOWN_HOSTS=$H/khd/kh"
for m in 644 640 400 700 200 4600; do : > "$H/k/key$m"; chmod $m "$H/k/key$m"; route vm-key$m "KEY=$H/k/key$m"; done
route vm-hopkey644 HOP=ops@hop-host "HOP_KEY=$H/k/key644"
mkdir -p "$H/kd"; : > "$H/kd/key"; chmod 777 "$H/kd"; route vm-keydir "KEY=$H/kd/key"
route vm-keyroot KEY=/etc/ssh/ssh_host_ed25519_key
route vm-a "KEY=$H/k/vm_key"
: > "$H/k/gone_key"; route vm-gone "KEY=$H/k/gone_key"
route vm-pct "KEY=$H/k/%d/vm_key"
mkdir -p "$H/k/A b"; : > "$H/k/A b/key"; ln -s "$H/k/A b" "$H/k/plain"; route vm-resolved-class "KEY=$H/k/plain/key"
mkdir -p "$H/k/A\${HOME}/.ssh"; : > "$H/k/A\${HOME}/.ssh/id_ed25519"; route vm-var "KEY=$H/k/A\${HOME}/.ssh/id_ed25519"
route vm-hopvar HOP=ops@hop-host "HOP_KEY=$H/k/A\${HOME}/.ssh/id_ed25519"
: > "$H/kh"
route vm-p "KEY=$H/k/vm_key" HOSTNAME=vm-p.internal USER=gate PORT=2222 PROXY=station-pipe "STATION=$H/bin/station" "KNOWN_HOSTS=$H/kh"
route vm-b HOP=ops@hop-host "HOP_KEY=$H/k/hop_key"
route vm-bk HOP=ops@hop-host "HOP_KEY=$H/k/hop_key" 'KNOWN_HOSTS=~/kh'
route vm-kh-rel "KEY=$H/k/vm_key" KNOWN_HOSTS=kh
route vm-kh-text "KEY=$H/k/vm_key" "KNOWN_HOSTS=$H/kh /etc/passwd"
route vm-kh-none "KEY=$H/k/vm_key" "KNOWN_HOSTS=$H/kh-absent"
route vm-def 'KEY=~/.ssh/id_ed25519'
route vm-dslash "KEY=$H//.ssh/id_ed25519"
route vm-dot 'KEY=~/.ssh/./id_ed25519'
route vm-other 'KEY=~/.ssh/other_key'
route vm-hopdef HOP=ops@hop-host 'HOP_KEY=~/.ssh/id_ed25519'
route vm-hopdot HOP=ops@hop-host "HOP_KEY=$H/k/../.ssh/other_key"
route vm-pw "KEY=$pw/.ssh/gate-ssh-absent-key"
route vm-link "KEY=$H/k/link_key"
route vm-hard "KEY=$H/k/hard_key"
route vm-hard-sub "KEY=$H/k/hard_sub"
route vm-hard-dot "KEY=$H/k/hard_dot"
route vm-hard-deep "KEY=$H/k/hard_deep"
route vm-rel KEY=k/vm_key
route vm-mixed "KEY=$H/k/vm_key" HOP=ops@hop-host
route vm-hopextra HOP=ops@hop-host "HOP_KEY=$H/k/hop_key" PORT=22
route vm-extra "KEY=$H/k/vm_key" AGENT=yes
route vm-twice "KEY=$H/k/vm_key" "KEY=$H/k/hop_key"
route vm-noeq "KEY=$H/k/vm_key" junk
route vm-proxy-other "KEY=$H/k/vm_key" PROXY=command
route vm-proxy-nostation "KEY=$H/k/vm_key" PROXY=station-pipe
route vm-proxy-rel "KEY=$H/k/vm_key" PROXY=station-pipe STATION=bin/station
route vm-proxy-text "KEY=$H/k/vm_key" PROXY=station-pipe 'STATION=/bin/sh -c evil'
route vm-proxy-pct "KEY=$H/k/vm_key" PROXY=station-pipe "STATION=$H/bin/st%h"
route vm-proxy-noexec "KEY=$H/k/vm_key" PROXY=station-pipe "STATION=$H/nx/station"
for s in env:/usr/bin/env ssh:/usr/bin/ssh link:$H/st/station ww:$H/ww/station wd:$H/wd/station app:$H/st/app other:$H/st/other inssh:$H/.ssh/bin/station hl:$H/hl/station; do
  route "vm-st-${s%%:*}" "KEY=$H/k/vm_key" PROXY=station-pipe "STATION=${s#*:}" "KNOWN_HOSTS=$H/kh"
done
route vm-station-alone "KEY=$H/k/vm_key" "STATION=$H/bin/station"
route vm-badhost "KEY=$H/k/vm_key" HOSTNAME=-oProxyCommand=x
route vm-baduser "KEY=$H/k/vm_key" 'USER=a b'
route vm-port0 "KEY=$H/k/vm_key" PORT=0
route vm-portbig "KEY=$H/k/vm_key" PORT=70000
route vm-portx "KEY=$H/k/vm_key" PORT=22x
route vm-badhop HOP=-oProxyCommand=x "HOP_KEY=$H/k/hop_key"
route vm-badhop-semi 'HOP=ops@hop;x' "HOP_KEY=$H/k/hop_key"
route vm-badhop-two HOP=a@b@c "HOP_KEY=$H/k/hop_key"
route vm-dl HOP=ops@nowhere.invalid "HOP_KEY=$H/k/hop_key"
route scp "KEY=$H/k/vm_key"
route vm-loose "KEY=$H/k/vm_key"; chmod 666 "$R/vm-loose.env"
ln -s "$R/vm-a.env" "$R/vm-sym.env"
mkdir -p "$T/routes-open"; cp "$R/vm-a.env" "$T/routes-open/"; chmod 777 "$T/routes-open"
ln -s "$R" "$T/routes-link"
GSH=bash
gate_ssh() { env -i PATH="$PATH" HOME="$H" GATE_ROUTES_DIR="$R" GATE_SSH_DRYRUN=1 "$GSH" ./gate-ssh.sh "$@"; }
refused() {
  local want=$1 err rc; shift
  err=$("$@" 2>&1 > /dev/null); rc=$?
  [ $rc = 92 ] && [[ $err == *"$want"* ]] || bad "not refused with '$want' (exit $rc): $* -> $err"
}
argv() { printf '%s\n' "$@"; }
common=$(argv -F /dev/null -o BatchMode=yes -o StrictHostKeyChecking=yes -o UpdateHostKeys=no -o GlobalKnownHostsFile=/dev/null -o IdentitiesOnly=yes -o IdentityAgent=none -o ForwardAgent=no -o ControlMaster=no -o ControlPath=none -o PermitLocalCommand=no -o EscapeChar=none)
b64_of() { tail -n 1 | sed -n "s/.*printf %s \([A-Za-z0-9+\/=]*\) | base64 -d.*/\1/p" | base64 -d; }
child=(/usr/bin/env -i PATH=/usr/bin:/bin:/usr/sbin:/sbin "HOME=$H")
ssh_w() { argv ${child[@]+"${child[@]}"} /usr/bin/ssh -a; echo "$common"; }
scp_w() { argv ${child[@]+"${child[@]}"} /usr/bin/scp; echo "$common"; }
sshg() { local a i; mapfile -t a; for i in "${!a[@]}"; do [ "${a[$i]}" = /usr/bin/ssh ] && break; done; /usr/bin/ssh -G "${a[@]:$((i + 1))}" 2>&1; }
dry_suite() {
: > "$H/k/gone_key"
want=$(ssh_w; argv -o "IdentityFile=$H/k/vm_key" -o "UserKnownHostsFile=$H/kh" -n -o ConnectTimeout=10 -- vm-a true)
got=$(gate_ssh ssh -n -o ConnectTimeout=10 vm-a true); [ "$got" = "$want" ] || bad "direct ssh argv: $got"
want=$(ssh_w; argv -o "IdentityFile=$H/k/vm_key" -o "UserKnownHostsFile=$H/kh" -- vm-a true)
got=$(gate_ssh rsh vm-a true); [ "$got" = "$want" ] || bad "direct rsh argv: $got"
got=$(env -i PATH="$PATH" HOME=/nonexistent GATE_ROUTES_DIR="$R" GATE_SSH_DRYRUN=1 "$GSH" ./gate-ssh.sh rsh vm-a true); [ "$(grep -v '^HOME=' <<< "$got")" = "$(grep -v '^HOME=' <<< "$want")" ] || bad "a direct route needs \$HOME: $got"
want=$(scp_w; argv -o "IdentityFile=$H/k/vm_key" -o "UserKnownHostsFile=$H/kh" -q -- a.txt vm-a:dir/)
got=$(gate_ssh scp -q a.txt vm-a:dir/); [ "$got" = "$want" ] || bad "direct scp argv: $got"
proxied=(-o "IdentityFile=$H/k/vm_key" -o "UserKnownHostsFile=$H/kh" -o HostName=vm-p.internal -o User=gate -o Port=2222 -o "ProxyCommand='$H/bin/station' machines pipe vm-p")
want=$(ssh_w; argv "${proxied[@]}" -- vm-p true)
got=$(gate_ssh ssh vm-p true); [ "$got" = "$want" ] || bad "proxied ssh argv: $got"
want=$(scp_w; argv "${proxied[@]}" -- a.txt vm-p:dir/)
got=$(gate_ssh scp a.txt vm-p:dir/); [ "$got" = "$want" ] || bad "proxied scp argv: $got"
for o in "-J evil@jump" "-o ProxyCommand=evil" "-i /x" "-A" "-F /x" "-oIdentityAgent=/x"; do
  got=$(gate_ssh ssh vm-a $o true)
  want=$(ssh_w; argv -o "IdentityFile=$H/k/vm_key" -o "UserKnownHostsFile=$H/kh" -- vm-a $o true)
  [ "$got" = "$want" ] || bad "an option after the host is not a remote word: $o -> $got"
  cfg=$(sshg <<< "$got")
  [ "$(grep '^identityfile ' <<< "$cfg")" = "identityfile $H/k/vm_key" ] || bad "ssh -G: an identity beside the route's after '$o': $(grep '^identityfile ' <<< "$cfg")"
  grep -qx 'identityagent none' <<< "$cfg" && grep -qx 'forwardagent no' <<< "$cfg" || bad "ssh -G: an agent is reachable after '$o'"
  grep -qE '^(proxyjump|proxycommand) ' <<< "$cfg" && bad "ssh -G: a proxy after '$o': $(grep -E '^(proxyjump|proxycommand) ' <<< "$cfg")"
done
cfg=$(gate_ssh ssh vm-p true | sshg)
grep -qx "proxycommand '$H/bin/station' machines pipe vm-p" <<< "$cfg" && grep -qx 'hostname vm-p.internal' <<< "$cfg" && grep -qx 'user gate' <<< "$cfg" && grep -qx 'port 2222' <<< "$cfg" && grep -qx "userknownhostsfile $H/kh" <<< "$cfg" && grep -qx 'stricthostkeychecking true' <<< "$cfg" || bad "ssh -G: the proxied route is not what the route file says: $cfg"
got=$(gate_ssh ssh vm-gone true); rm -f "$H/k/gone_key"; cfg=$(sshg <<< "$got")
[ "$(grep '^identityfile ' <<< "$cfg")" = "identityfile $H/k/gone_key" ] || bad "ssh -G: a key gone at launch leaves ssh another identity: $(grep '^identityfile ' <<< "$cfg" | tr '\n' ' ')"
refused "KEY holds a character outside" gate_ssh ssh vm-pct true
refused "KEY holds a character outside" gate_ssh ssh vm-var true
refused "HOP_KEY holds a character outside" gate_ssh ssh vm-hopvar true
refused "KEY holds a character outside" gate_ssh ssh vm-resolved-class true
printf '%s\n' 'Host *' '  IdentityFile ~/.ssh/owner_id' '  CertificateFile ~/.ssh/owner_id-cert.pub' '  IdentitiesOnly no' '  IdentityAgent /tmp/evil-agent.sock' '  ForwardAgent yes' \
  '  ProxyCommand /bin/false evil' '  SendEnv SSH_AUTH_SOCK' '  SetEnv EVIL=1' '  LocalForward 4444 127.0.0.1:22' '  UserKnownHostsFile ~/.ssh/known_hosts' '  GlobalKnownHostsFile ~/.ssh/global_hosts' \
  '  StrictHostKeyChecking no' '  UpdateHostKeys yes' '  ControlMaster auto' '  ControlPath ~/.ssh/cm-%C' '  PermitLocalCommand yes' '  LocalCommand touch /tmp/evil-local' '  Hostname evil.example' '  User evil' '  Port 2200' > "$T/hostile.conf"
cfg=$(/usr/bin/ssh -G -F "$T/hostile.conf" vm-a 2>&1)
grep -qx 'hostname evil.example' <<< "$cfg" && grep -qx 'sendenv SSH_AUTH_SOCK' <<< "$cfg" || bad "the hostile config stand-in is not live, so its contrast proves nothing: $cfg"
cfg=$(gate_ssh ssh vm-a true | { mapfile -t a; for i in "${!a[@]}"; do [ "${a[$i]}" = /usr/bin/ssh ] && break; done; /usr/bin/ssh -G -F "$T/hostile.conf" "${a[@]:$((i + 1))}" 2>&1; })
grep -E '^(sendenv|setenv|localforward|certificatefile|proxycommand|proxyjump|localcommand) |^(hostname evil\.example|user evil|port 2200)$' <<< "$cfg" && bad "ssh -G: a config read before the guard's options adds to the route"
[ "$(grep '^identityfile ' <<< "$cfg")" = "identityfile $H/k/vm_key" ] || bad "ssh -G: a config read before the guard's options adds an identity: $(grep '^identityfile ' <<< "$cfg" | tr '\n' ' ')"
for l in 'identityagent none' 'identitiesonly yes' 'forwardagent no' "userknownhostsfile $H/kh" 'globalknownhostsfile /dev/null' 'stricthostkeychecking true' 'updatehostkeys false' 'controlmaster false' 'permitlocalcommand no'; do
  grep -qx "$l" <<< "$cfg" || bad "ssh -G: a config read before the guard's options changes '$l': $(grep "^${l%% *} " <<< "$cfg")"
done
got=$(gate_ssh ssh -n vm-b 'echo "a b"; ls')
want=$(ssh_w; argv -o "IdentityFile=$H/k/hop_key" -o "UserKnownHostsFile=$H/kh" -n -- ops@hop-host)
[ "$(sed '$d' <<< "$got")" = "$want" ] || bad "hop ssh argv: $got"
[ "$(b64_of <<< "$got")" = 'echo "a b"; ls' ] || bad "hop ssh command does not round-trip: $(tail -n 1 <<< "$got")"
case $(tail -n 1 <<< "$got") in "ssh -o BatchMode=yes -- vm-b 'eval "*) ;; *) bad "hop ssh second hop: $(tail -n 1 <<< "$got")" ;; esac
want=$(ssh_w; argv -o "IdentityFile=$H/k/hop_key" -o "UserKnownHostsFile=$H/kh" -- ops@hop-host)
got=$(gate_ssh ssh vm-bk true); [ "$(sed '$d' <<< "$got")" = "$want" ] || bad "hop ssh argv with its known hosts: $got"
rm -rf "$T/sbx"; mkdir -p "$T/sbx/dir x"; printf x > "$T/it's a.txt"; printf x > "$T/up.txt"
got=$(gate_ssh scp "$T/it's a.txt" "vm-b:dir x/"); [ "$(grep -c '^/usr/bin/ssh$' <<< "$got")" = 1 ] || bad "hop scp upload is not one ssh: $got"
remote=$(b64_of <<< "$got"); (cd "$T/sbx" && printf y | sh -c "$remote"); [ "$(cat "$T/sbx/dir x/it's a.txt" 2> /dev/null)" = y ] || bad "hop scp upload names another file: $remote"
remote=$(gate_ssh scp "$T/up.txt" 'vm-b:$(touch pwned)' | b64_of); (cd "$T/sbx" && printf y | sh -c "$remote"); [ ! -e "$T/sbx/pwned" ] && [ -f "$T/sbx/\$(touch pwned)" ] || bad "hop scp upload runs its destination: $remote"
rm -f "$T/down.txt"; got=$(gate_ssh scp 'vm-b:a b;touch pwned' "$T/down.txt"); [ "$(b64_of <<< "$got")" = "/bin/cat -- 'a b;touch pwned'" ] || bad "hop scp download runs its source or a cat from PATH: $(tail -n 1 <<< "$got")"
[ ! -e "$T/down.txt" ] || bad "a dry run of a hop download writes its destination"
printf keep > "$T/down-keep.txt"; gate_ssh scp vm-b:x "$T/down-keep.txt" > /dev/null; [ "$(cat "$T/down-keep.txt")" = keep ] || bad "a dry run of a hop download overwrites its destination"
case $(gate_ssh scp "$T/up.txt" vm-b:x | b64_of) in *'/bin/cat > "$d"') ;; *) bad "hop scp upload runs a cat from PATH" ;; esac
nl_name="q'"$'\n'"touch pwned-nl"; printf x > "$T/$nl_name"; rm -f "$T/sbx/pwned-nl"
remote=$(gate_ssh scp "$T/$nl_name" "vm-b:dir x/" | b64_of); (cd "$T/sbx" && printf y | sh -c "$remote" 2> /dev/null)
[ ! -e "$T/sbx/pwned-nl" ] && [ "$(cat "$T/sbx/dir x/$nl_name" 2> /dev/null)" = y ] || bad "hop scp upload of a name with a quote and a newline runs a line of it or names another file: $remote"
printf z > "$T/sbx/$nl_name"; remote=$(gate_ssh scp "vm-b:$nl_name" "$T/down-nl.txt" | b64_of)
[ "$(cd "$T/sbx" && sh -c "$remote" 2> /dev/null)" = z ] && [ ! -e "$T/sbx/pwned-nl" ] || bad "hop scp download of a name with a quote and a newline reads another file or runs a line of it: $remote"
E=$T/evil; mkdir -p "$E"; for c in base64 tr ls id dirname basename cut readlink find head rm; do printf '#!/bin/sh\necho evil\n' > "$E/$c"; chmod 755 "$E/$c"; done
printf x > "$T/dl.out"; env -i PATH="$E:$PATH" HOME="$H" GATE_ROUTES_DIR="$R" timeout 30 "$GSH" ./gate-ssh.sh scp vm-dl:x "$T/dl.out" > /dev/null 2>&1; rc=$?
[ $rc = 255 ] && [ ! -e "$T/dl.out" ] || bad "a failed hop download is not removed by the system rm (exit $rc)"
mkdir -p "$T/cdp/gates"; printf '#!/bin/sh\necho cdpath > %s\n' "$T/cdp.hit" > "$T/cdp/gates/gate-ssh.sh"; chmod 755 "$T/cdp/gates/gate-ssh.sh"
got=$(cd .. && env -i PATH="$PATH" HOME="$H" GATE_ROUTES_DIR="$R" GATE_SSH_DRYRUN=1 CDPATH="$T/cdp" "$GSH" gates/gate-ssh.sh scp "$T/up.txt" vm-b:x 2>&1)
[ "$(grep -c '^/usr/bin/ssh$' <<< "$got")" = 1 ] && [ ! -e "$T/cdp.hit" ] || bad "CDPATH moves gate-ssh's own path: $got"
cfg=$(gate_ssh ssh vm-a true | sshg); grep -qx 'escapechar none' <<< "$cfg" || bad "ssh -G: an escape character is live: $(grep '^escapechar' <<< "$cfg")"
refused "scp names a URI" gate_ssh scp "$T/up.txt" 'scp://someone@evil.example:2200/x'
refused "scp names a URI" gate_ssh scp 'scp://vm-a/x' "$T/dl.out"
got=$(env -i PATH="$E:$PATH" HOME="$H" GATE_ROUTES_DIR="$R" GATE_SSH_DRYRUN=1 "$GSH" ./gate-ssh.sh ssh vm-b 'echo hi' 2>&1); [ "$(b64_of <<< "$got")" = 'echo hi' ] || bad "gate-ssh takes a tool from PATH: $got"
refused "no route for vm-nope" gate_ssh ssh vm-nope true
for h in vm-def vm-dslash vm-dot vm-other vm-hopdef vm-hopdot; do refused "where the owner's own identities live" gate_ssh ssh $h true; done
if [ -d "$pw/.ssh" ]; then
  refused "$pw/.ssh/gate-ssh-absent-key lies under" env -i PATH="$PATH" HOME=/nonexistent GATE_ROUTES_DIR="$R" GATE_SSH_DRYRUN=1 "$GSH" ./gate-ssh.sh ssh vm-pw true
else
  echo "skip: the passwd home has no ~/.ssh here (the passwd-home test)"
fi
refused "KEY $H/.ssh/gate-ssh-absent-key lies under $H/.ssh" gate_ssh ssh vm-key-alias true
refused "KNOWN_HOSTS $H/.ssh/known_hosts lies under $H/.ssh" gate_ssh ssh vm-kh-alias true
refused "KNOWN_HOSTS $H/kh-hard is the same file as $H/.ssh/known_hosts" gate_ssh ssh vm-kh-hard true
refused "STATION $H/.ssh/bin/station lies under $H/.ssh" gate_ssh ssh vm-st-inssh true
refused "STATION $H/hl/station is the same file as $H/.ssh/bin/station" gate_ssh ssh vm-st-hl true
refused "is a symbolic link: name the file itself" gate_ssh ssh vm-link true
refused "KNOWN_HOSTS is unset" gate_ssh ssh vm-nokh true
refused "KNOWN_HOSTS is unset" gate_ssh ssh vm-hopnokh true
refused "a gate route keeps its host keys outside ~/.ssh" gate_ssh ssh vm-kh-ssh true
refused "KNOWN_HOSTS $H/kh-ww is not yours alone" gate_ssh ssh vm-kh-ww true
refused "KNOWN_HOSTS /etc/passwd is not yours alone" gate_ssh ssh vm-kh-root true
refused "KNOWN_HOSTS $H/kh-link is a symbolic link" gate_ssh ssh vm-kh-link true
refused "the folder of KNOWN_HOSTS" gate_ssh ssh vm-kh-dir true
for h in vm-key644 vm-key640 vm-hopkey644 vm-key700 vm-key200 vm-key4600; do refused "must be owned by you and readable by you alone" gate_ssh ssh $h true; done
refused "the folder of KEY" gate_ssh ssh vm-keydir true
[ "$(gate_ssh ssh vm-key400 true | grep '^IdentityFile=')" = "IdentityFile=$H/k/key400" ] || bad "a key of mode 400 is refused"
if [ -f /etc/ssh/ssh_host_ed25519_key ]; then
  refused "KEY /etc/ssh/ssh_host_ed25519_key must be owned by you" gate_ssh ssh vm-keyroot true
else
  echo "skip: no root-owned key of mode 600 here (the key owner test)"
fi
cfg=$(gate_ssh ssh vm-a true | sshg)
grep -qx 'updatehostkeys false' <<< "$cfg" && grep -qx 'globalknownhostsfile /dev/null' <<< "$cfg" && grep -qx "userknownhostsfile $H/kh" <<< "$cfg" || bad "ssh -G: a gate connection may write or read known hosts beside the route's: $(grep -E '^(updatehostkeys|globalknownhostsfile|userknownhostsfile) ' <<< "$cfg" | tr '\n' ' ')"
for h in vm-hard vm-hard-sub vm-hard-dot vm-hard-deep; do refused "is the same file as" gate_ssh ssh $h true; done
refused "is not an absolute path" gate_ssh ssh vm-rel true
refused "a two-hop route sets HOP, HOP_KEY and KNOWN_HOSTS and nothing else" gate_ssh ssh vm-mixed true
refused "a two-hop route sets HOP, HOP_KEY and KNOWN_HOSTS and nothing else" gate_ssh ssh vm-hopextra true
refused "KNOWN_HOSTS kh is not an absolute path" gate_ssh ssh vm-kh-rel true
refused "KNOWN_HOSTS holds a character" gate_ssh ssh vm-kh-text true
refused "is not a file" gate_ssh ssh vm-kh-none true
refused "which is none of" gate_ssh ssh vm-extra true
refused "sets KEY twice" gate_ssh ssh vm-twice true
refused "not KEY=value" gate_ssh ssh vm-noeq true
refused "PROXY is station-pipe or unset" gate_ssh ssh vm-proxy-other true
refused "needs STATION=<absolute path" gate_ssh ssh vm-proxy-nostation true
refused "needs STATION=<absolute path" gate_ssh ssh vm-proxy-rel true
refused "STATION holds a character" gate_ssh ssh vm-proxy-text true
refused "STATION holds a character" gate_ssh ssh vm-proxy-pct true
refused "is not an executable file" gate_ssh ssh vm-proxy-noexec true
for h in vm-st-env vm-st-ssh vm-st-link vm-st-other; do refused "the two names the installer ships" gate_ssh ssh $h true; done
for h in vm-st-ww vm-st-wd; do refused "or is writable by someone else" gate_ssh ssh $h true; done
[ "$(gate_ssh ssh vm-st-app true | grep '^ProxyCommand=')" = "ProxyCommand='$H/Apps/Spwn.app/Contents/MacOS/Spwn Engine' machines pipe vm-st-app" ] || bad "Spwn.app's engine is not the proxy, quoted: $(gate_ssh ssh vm-st-app true 2>&1 | grep -i proxy)"
[ "$(gate_ssh ssh vm-st-app true | sshg | grep '^proxycommand ')" = "proxycommand '$H/Apps/Spwn.app/Contents/MacOS/Spwn Engine' machines pipe vm-st-app" ] || bad "ssh -G: Spwn.app's engine is not one word"
refused "the two names the installer ships" env -i PATH="$PATH" HOME="$H" GATE_ROUTES_DIR="$R" SSH_AUTH_SOCK=/run/presence.sock SSH_ASKPASS=/usr/bin/true "$GSH" ./gate-ssh.sh ssh vm-st-env true
rm -f "$T/proxy.env"
env -i PATH="$PATH" HOME="$H" GATE_ROUTES_DIR="$R" SSH_AUTH_SOCK=/run/presence.sock SSH_ASKPASS=/usr/bin/true SSH_ASKPASS_REQUIRE=force DISPLAY=:0 timeout 20 "$GSH" ./gate-ssh.sh ssh -o ConnectTimeout=5 vm-p true > /dev/null 2>&1
[ -f "$T/proxy.env" ] || bad "the proxy never ran, so its environment is unproven"
grep -E '^(SSH_AUTH_SOCK|SSH_ASKPASS|SSH_ASKPASS_REQUIRE|DISPLAY|GATE_ROUTES_DIR)=' "$T/proxy.env" && bad "the proxy inherits the caller's agent, askpass or display"
grep -qx 'PATH=/usr/bin:/bin:/usr/sbin:/sbin' "$T/proxy.env" && grep -qx "HOME=$H" "$T/proxy.env" || bad "the proxy's PATH or HOME is not the fixed one: $(grep -E '^(PATH|HOME)=' "$T/proxy.env")"
for c in ssh scp; do
  w=(); [ $c = scp ] && w=(a.txt vm-a:b) || w=(vm-a true)
  [ "$(env -i PATH="$PATH" HOME="$H" GATE_ROUTES_DIR="$R" SSH_AUTH_SOCK=/x GATE_SSH_DRYRUN=1 "$GSH" ./gate-ssh.sh $c "${w[@]}" | head -n 2)" = "$(printf '/usr/bin/env\n-i\n')" ] || bad "$c is not launched with a scrubbed environment"
done
refused "STATION is read only with PROXY" gate_ssh ssh vm-station-alone true
refused "HOSTNAME is not a plain host name" gate_ssh ssh vm-badhost true
refused "USER is not a plain user name" gate_ssh ssh vm-baduser true
for h in vm-port0 vm-portbig vm-portx; do refused "PORT is not a port number" gate_ssh ssh $h true; done
for h in vm-badhop vm-badhop-semi vm-badhop-two; do refused "HOP is not user@host" gate_ssh ssh $h true; done
refused "is not yours alone" gate_ssh ssh vm-loose true
refused "is a symbolic link: write the file itself" gate_ssh ssh vm-sym true
refused "route folder $T/routes-open is not yours alone" env -i PATH="$PATH" HOME="$H" GATE_ROUTES_DIR="$T/routes-open" GATE_SSH_DRYRUN=1 "$GSH" ./gate-ssh.sh ssh vm-a true
refused "route folder /usr is not yours alone" env -i PATH="$PATH" HOME="$H" GATE_ROUTES_DIR=/usr GATE_SSH_DRYRUN=1 "$GSH" ./gate-ssh.sh ssh vm-a true
refused "is a symbolic link: name the folder itself" env -i PATH="$PATH" HOME="$H" GATE_ROUTES_DIR="$T/routes-link" GATE_SSH_DRYRUN=1 "$GSH" ./gate-ssh.sh ssh vm-a true
refused "no route for vm-a" env -i PATH="$PATH" HOME="$H" GATE_SSH_DRYRUN=1 "$GSH" ./gate-ssh.sh ssh vm-a true
refused "is not a plain host name" gate_ssh ssh a/b true
refused "is not a plain host name" gate_ssh ssh user@vm-a true
refused "no route for vm-nope" gate_ssh scp x vm-nope:y
refused "scp names two hosts" gate_ssh scp x vm-a:y vm-b:z
refused "scp between two remote paths" gate_ssh scp vm-a:x vm-a:y
for c in "ssh -i /x vm-a true" "ssh -F /dev/null vm-a true" "ssh -J h vm-a true" "ssh -A vm-a true" "ssh -o IdentityAgent=x vm-a true" "ssh -o ProxyCommand=x vm-a true" "ssh" "scp -r a vm-a:b" "scp -P 1 a vm-a:b" "scp a vm-a:b -o ProxyCommand=x" "scp a b" "bogus"; do
  gate_ssh $c > /dev/null 2>&1; [ $? = 2 ] || bad "an unrouted option is accepted: $c"
done
}
dry_suite
B32=${GATE_BASH32:-$(command -v bash-3.2 || true)}
if [ -n "$B32" ]; then
  [ "$("$B32" -c 'echo "${BASH_VERSINFO[0]}.${BASH_VERSINFO[1]}"')" = 3.2 ] || bad "GATE_BASH32 $B32 is not bash 3.2"
  GSH=$B32; TAG=bash-3.2; dry_suite; GSH=bash; TAG=
  echo "gate-ssh.sh dry runs and quoting proved under $("$B32" -c 'echo "$BASH_VERSION"')"
else
  echo "SKIP: no bash 3.2 here (set GATE_BASH32 or put bash-3.2 on PATH): gate-ssh.sh is NOT proved under the Mac's /bin/bash"
fi
launcher() {
  local t0 t1 ms err rc
  t0=$EPOCHREALTIME
  err=$(env -i PATH="$PATH" HOME="$H" GATE_ROUTES_DIR="$R" GATE_RUN_DIR="$T/lrun" GATE_VM_LINUX=vm-nope timeout 10 "$@" 2>&1 > /dev/null); rc=$?
  t1=$EPOCHREALTIME; ms=$(((10#${t1//[.,]/} - 10#${t0//[.,]/}) / 1000))
  [ $rc = 92 ] && [[ $err == *"no route for vm-nope"* ]] || bad "a launcher does not stop on gate-ssh's refusal (exit $rc): $* -> $err"
  [ $ms -lt 1000 ] || bad "a launcher took $ms ms to stop on gate-ssh's refusal: $*"
}
G=$T/repo; git init -q -b main "$G"; for m in a b; do git -C "$G" -c user.name=t -c user.email=t@t.invalid commit -q --allow-empty -m $m; done
printf '1 %s\n' "$(git -C "$G" rev-parse HEAD)" > "$T/regen.map"
launcher bash ./cc.sh "$G" HEAD~1 lbl
launcher bash ./cold-gate.sh "$G" main lbl
launcher bash ./desktop-proof.sh "$G" lbl
launcher bash ./regen.sh one "$G" HEAD lbl
launcher bash ./regen.sh all "$G" "$T/regen.map" lbl 1 ""
launcher env GATE_SLOT_VM=vm-nope bash ./gate-slot.sh true
launcher env STATION_VM=vm-nope bash ../../machines/apps/desktop/scripts/look.sh lot src/x.test.ts
if command -v sshd > /dev/null && command -v ssh-keygen > /dev/null && command -v python3 > /dev/null; then
  S=$T/sd; O=$T/owner; mkdir -p "$S" "$O/.ssh"; chmod 700 "$S" "$O" "$O/.ssh"
  ssh-keygen -q -t ed25519 -N '' -f "$S/host"; ssh-keygen -q -t ed25519 -N '' -f "$S/client"; ssh-keygen -q -t ed25519 -N '' -C synthetic-owner -f "$O/.ssh/id_ed25519"
  cat "$S/client.pub" "$O/.ssh/id_ed25519.pub" > "$S/authorized_keys"
  port=$(python3 -I -c 'import socket; s = socket.socket(); s.bind(("127.0.0.1", 0)); print(s.getsockname()[1])')
  printf 'Port %s\nListenAddress 127.0.0.1\nHostKey %s\nAuthorizedKeysFile %s\nPidFile %s\nUsePAM no\nStrictModes no\nPasswordAuthentication no\nLogLevel QUIET\n' "$port" "$S/host" "$S/authorized_keys" "$S/pid" > "$S/sshd_config"
  "$(command -v sshd)" -D -f "$S/sshd_config" > /dev/null 2>&1 & sp=$!
  python3 -I -c 'import socket, sys, time; s = socket.socket(socket.AF_UNIX); s.bind(sys.argv[1]); s.listen(8); time.sleep(120)' "$S/silent.sock" & kp=$!
  up=0; for _ in $(seq 50); do (exec 3<> "/dev/tcp/127.0.0.1/$port") 2> /dev/null && { up=1; break; }; command sleep 0.1; done
  if [ $up = 1 ]; then
    base=(-F /dev/null -p "$port" -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o LogLevel=ERROR -o ConnectTimeout=5 -o BatchMode=yes -l "$(id -un)")
    timeout 8 ssh "${base[@]}" -o IdentityAgent="$S/silent.sock" 127.0.0.1 true > /dev/null 2>&1; [ $? = 124 ] || bad "BatchMode bounds a stuck agent: the guard is not the only protection"
    timeout 8 ssh "${base[@]}" -o IdentitiesOnly=yes -o IdentityAgent=none -i "$S/client" 127.0.0.1 true > /dev/null 2>&1; [ $? = 0 ] || bad "the guard options do not reach a server with the dedicated key alone"
    printf '[127.0.0.1]:%s %s\n' "$port" "$(cut -d' ' -f1-2 "$S/host.pub")" > "$S/kh"
    live=(HOSTNAME=127.0.0.1 "PORT=$port" "USER=$(id -un)" "KNOWN_HOSTS=$S/kh")
    live_ssh() { env -i PATH="$PATH" HOME="$O" GATE_ROUTES_DIR="$R" timeout 20 bash ./gate-ssh.sh ssh "$@"; }
    route sd-ok "KEY=$S/client" "${live[@]}"
    [ "$(live_ssh sd-ok echo routed 2> /dev/null)" = routed ] || bad "the dedicated key does not reach a local sshd through gate-ssh"
    P=$T/pp; mkdir -p "$P/A\${HOME}/.ssh"; : > "$P/A\${HOME}/.ssh/id_ed25519"; chmod 600 "$P/A\${HOME}/.ssh/id_ed25519"; ln -s / "$P/A"
    route sd-decoy "KEY=$P/A\${HOME}/.ssh/id_ed25519" "${live[@]}"
    out=$(live_ssh sd-decoy echo owner-identity 2>&1); rc=$?
    [ $rc = 92 ] && [[ $out != *owner-identity* ]] || bad "a KEY ssh expands reached the server with the owner's identity (exit $rc): $out"
    cp "$S/client" "$S/gone"; chmod 600 "$S/gone"; route sd-gone "KEY=$S/gone" "${live[@]}"
    mapfile -t a < <(env -i PATH="$PATH" HOME="$O" GATE_ROUTES_DIR="$R" GATE_SSH_DRYRUN=1 "$GSH" ./gate-ssh.sh ssh sd-gone echo reached)
    rm -f "$S/gone"
    out=$(env -i PATH="$PATH" HOME="$O" timeout 20 "${a[@]}" 2>&1); rc=$?
    [ $rc = 255 ] && [[ $out != *reached* ]] || bad "a key gone at launch still authenticates (exit $rc): $out"
  else
    echo "skip: no sshd could listen here (the live route tests)"
  fi
  kill "$sp" "$kp" 2> /dev/null; wait "$sp" "$kp" 2> /dev/null
else
  echo "skip: no sshd, ssh-keygen or python3 here (the live route tests)"
fi
[ $fail = 0 ] && echo "gate scripts: ok"
exit $fail

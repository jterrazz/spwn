#!/bin/bash
set -u
cd "$(dirname "$0")" || exit 2
fail=0; bad() { echo "FAIL $*"; fail=1; }
for f in *.sh; do bash -n "$f" || bad "syntax $f"; done
python3 -I -c 'import ast, sys; ast.parse(open(sys.argv[1]).read())' cc-verdict.py || bad "syntax cc-verdict.py"
grep -nE '/private/tmp|/tmp/claude' ./*.sh ./*.py | grep -v '^./test.sh' && bad "a script names a scratchpad"
T=$(mktemp -d) || exit 2; trap 'rm -rf "$T"' EXIT
d=$(env -u GATE_RUN_DIR bash -c '. ./env.sh; echo "$RUN"'); case $d in */.artifacts/gates) ;; *) bad "default run folder is $d" ;; esac
d=$(GATE_RUN_DIR=$T/run bash -c '. ./env.sh; echo "$RUN"'); [ "$d" = "$T/run" ] || bad "GATE_RUN_DIR is ignored: $d"
d=$(env -u GATE_VM_RAM GATE_RUN_DIR=$T/run GATE_VM_DIR=g bash -c '. ./env.sh; echo "$VM_RAM"'); [ "$d" = /gates/g ] || bad "default RAM folder is $d"
d=$(GATE_RUN_DIR=$T/run GATE_VM_RAM=/r bash -c '. ./env.sh; echo "$VM_RAM"'); [ "$d" = /r ] || bad "GATE_VM_RAM is ignored: $d"
grep -nE 'git clone' ./*.sh | grep -v '^./test.sh' | grep -vE 'git clone -q -b \S+ \S+ ("\$WT"|\$C|\$W/)' && bad "a clone lands outside the RAM folder"
grep -noE '(^|[ ;])(WT|C)=[^ ;]+' ./*.sh | grep -v '^./test.sh' | grep -vE '=\$(W|VM_RAM)/' && bad "a clone folder is not under the RAM folder"
bare_ssh() {
  local f n l s re='(^|[^A-Za-z0-9_.-])(ssh|scp|sftp|rsync)($|[^A-Za-z0-9_-])' comment='^[[:space:]]*#'
  for f in "$@"; do
    n=0
    while IFS= read -r l || [ -n "$l" ]; do
      n=$((n + 1))
      [[ $l =~ $comment ]] && continue
      s=${l//'"$GATE_SSH" ssh'/}; s=${s//'"$GATE_SSH" scp'/}; s=${s//'"$GATE_SSH" rsh'/}; s=${s//'rsync -e "$GATE_SSH rsh"'/}
      [[ $s =~ $re ]] || continue
      [[ $l == *'# gate-ssh: exempt'* && $l == *'-F /dev/null'* && $l == *'IdentityAgent=none'* ]] && continue
      echo "$f:$n: $l"
    done < "$f"
  done
}
F=$(mktemp -d) || exit 2
bare_lines=('ssh vm true' '  scp x vm:y' 'a && ssh -n vm true' 'x=$(ssh vm true)' 'S="ssh -n"' 'rsync -a a vm:b' 'if ssh vm true; then'
  'command ssh vm true' 'env ssh vm true' 'exec ssh vm true' 'nohup ssh vm true' 'timeout 5 ssh vm true' 'echo vm | xargs ssh' '/usr/bin/ssh vm true'
  'while ssh vm true; do :; done' 'x=`ssh vm true`' 's=ssh; $s vm true' 'rsync -e "$GATE_SSH rsh" a vm:b; ssh vm true' '"$GATE_SSH" ssh vm true; scp a vm:b'
  'sftp vm' 'rsync -a -e ssh a vm:b' 'echo "(hold or ssh)"' 'ssh vm true  # gate-ssh: exempt (no guard)' 'ssh -F /dev/null vm true  # gate-ssh: exempt (half a guard)')
printf '%s\n' "${bare_lines[@]}" > "$F/bare.sh"
printf '%s\n' '"$GATE_SSH" ssh vm true' '# ssh vm true' 'ssh -F /dev/null -o IdentityAgent=none vm true  # gate-ssh: exempt (why)' 'rsync -e "$GATE_SSH rsh" a vm:b' 'SSHP=("$GATE_SSH" ssh -n)' 'ssh-keygen -l -f k' 'cat ~/.ssh/known_hosts' > "$F/clean.sh"
tree=(); for f in ./*.sh ../*.sh ../../machines/apps/*/scripts/*.sh; do case $f in ./gate-ssh.sh | ./test.sh) ;; *) tree+=("$f") ;; esac; done
[ ${#tree[@]} -ge 20 ] || bad "the bare-ssh lint reads only ${#tree[@]} scripts"
lint=$(bare_ssh "$F/bare.sh" "$F/clean.sh" "${tree[@]}")
n=$(grep -cF "$F/bare.sh:" <<< "$lint"); [ "$n" = ${#bare_lines[@]} ] || bad "the bare-ssh lint did not run on the tree or misses a form: $n of ${#bare_lines[@]} fixture lines flagged"
rest=$(grep -vF "$F/bare.sh:" <<< "$lint"); [ -z "$rest" ] || bad "bare ssh to a VM (a script reaches a VM only through \"\$GATE_SSH\"): $rest"
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
H=$T/home; R=$T/routes; mkdir -p "$H/.ssh" "$H/k" "$H/bin" "$R"; chmod 700 "$R"
: > "$H/k/vm_key"; : > "$H/k/hop_key"; : > "$H/.ssh/id_ed25519"; : > "$H/.ssh/other_key"
ln -s "$H/k/vm_key" "$H/k/link_key"; ln "$H/.ssh/other_key" "$H/k/hard_key"
printf '#!/bin/sh\n' > "$H/bin/station"; chmod 755 "$H/bin/station"; : > "$H/bin/plain"
eval "pw=~$(id -un)"
route() { local n=$1; shift; printf '%s\n' "$@" > "$R/$n.env"; chmod 600 "$R/$n.env"; }
route vm-a "KEY=$H/k/vm_key"
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
route vm-proxy-noexec "KEY=$H/k/vm_key" PROXY=station-pipe "STATION=$H/bin/plain"
route vm-station-alone "KEY=$H/k/vm_key" "STATION=$H/bin/station"
route vm-badhost "KEY=$H/k/vm_key" HOSTNAME=-oProxyCommand=x
route vm-baduser "KEY=$H/k/vm_key" 'USER=a b'
route vm-port0 "KEY=$H/k/vm_key" PORT=0
route vm-portbig "KEY=$H/k/vm_key" PORT=70000
route vm-portx "KEY=$H/k/vm_key" PORT=22x
route vm-badhop HOP=-oProxyCommand=x "HOP_KEY=$H/k/hop_key"
route vm-loose "KEY=$H/k/vm_key"; chmod 666 "$R/vm-loose.env"
ln -s "$R/vm-a.env" "$R/vm-sym.env"
mkdir -p "$T/routes-open"; cp "$R/vm-a.env" "$T/routes-open/"; chmod 777 "$T/routes-open"
ln -s "$R" "$T/routes-link"
gate_ssh() { env -i PATH="$PATH" HOME="$H" GATE_ROUTES_DIR="$R" GATE_SSH_DRYRUN=1 bash ./gate-ssh.sh "$@"; }
refused() {
  local want=$1 err rc; shift
  err=$("$@" 2>&1 > /dev/null); rc=$?
  [ $rc = 92 ] && [[ $err == *"$want"* ]] || bad "not refused with '$want' (exit $rc): $* -> $err"
}
argv() { printf '%s\n' "$@"; }
common=$(argv -F /dev/null -o BatchMode=yes -o StrictHostKeyChecking=yes -o IdentitiesOnly=yes -o IdentityAgent=none -o ForwardAgent=no -o ControlMaster=no -o ControlPath=none -o PermitLocalCommand=no)
b64_of() { tail -n 1 | sed -n "s/.*printf %s \([A-Za-z0-9+\/=]*\) | base64 -d.*/\1/p" | base64 -d; }
want=$(argv /usr/bin/ssh -a; echo "$common"; argv -i "$H/k/vm_key" -n -o ConnectTimeout=10 -- vm-a true)
got=$(gate_ssh ssh -n -o ConnectTimeout=10 vm-a true); [ "$got" = "$want" ] || bad "direct ssh argv: $got"
want=$(argv /usr/bin/ssh -a; echo "$common"; argv -i "$H/k/vm_key" -- vm-a true)
got=$(gate_ssh rsh vm-a true); [ "$got" = "$want" ] || bad "direct rsh argv: $got"
got=$(env -i PATH="$PATH" HOME=/nonexistent GATE_ROUTES_DIR="$R" GATE_SSH_DRYRUN=1 bash ./gate-ssh.sh rsh vm-a true); [ "$got" = "$want" ] || bad "a direct route needs \$HOME: $got"
want=$(argv /usr/bin/scp; echo "$common"; argv -i "$H/k/vm_key" -q -- a.txt vm-a:dir/)
got=$(gate_ssh scp -q a.txt vm-a:dir/); [ "$got" = "$want" ] || bad "direct scp argv: $got"
proxied=(-i "$H/k/vm_key" -o "UserKnownHostsFile=$H/kh" -o HostName=vm-p.internal -o User=gate -o Port=2222 -o "ProxyCommand=$H/bin/station machines pipe vm-p")
want=$(argv /usr/bin/ssh -a; echo "$common"; argv "${proxied[@]}" -- vm-p true)
got=$(gate_ssh ssh vm-p true); [ "$got" = "$want" ] || bad "proxied ssh argv: $got"
want=$(argv /usr/bin/scp; echo "$common"; argv "${proxied[@]}" -- a.txt vm-p:dir/)
got=$(gate_ssh scp a.txt vm-p:dir/); [ "$got" = "$want" ] || bad "proxied scp argv: $got"
for o in "-J evil@jump" "-o ProxyCommand=evil" "-i /x" "-A" "-F /x" "-oIdentityAgent=/x"; do
  got=$(gate_ssh ssh vm-a $o true)
  want=$(argv /usr/bin/ssh -a; echo "$common"; argv -i "$H/k/vm_key" -- vm-a $o true)
  [ "$got" = "$want" ] || bad "an option after the host is not a remote word: $o -> $got"
  mapfile -t a <<< "$got"
  cfg=$(/usr/bin/ssh -G "${a[@]:1}" 2>&1)
  [ "$(grep '^identityfile ' <<< "$cfg")" = "identityfile $H/k/vm_key" ] || bad "ssh -G: an identity beside the route's after '$o': $(grep '^identityfile ' <<< "$cfg")"
  grep -qx 'identityagent none' <<< "$cfg" && grep -qx 'forwardagent no' <<< "$cfg" || bad "ssh -G: an agent is reachable after '$o'"
  grep -qE '^(proxyjump|proxycommand) ' <<< "$cfg" && bad "ssh -G: a proxy after '$o': $(grep -E '^(proxyjump|proxycommand) ' <<< "$cfg")"
done
mapfile -t a < <(gate_ssh ssh vm-p true); cfg=$(/usr/bin/ssh -G "${a[@]:1}" 2>&1)
grep -qx "proxycommand $H/bin/station machines pipe vm-p" <<< "$cfg" && grep -qx 'hostname vm-p.internal' <<< "$cfg" && grep -qx 'user gate' <<< "$cfg" && grep -qx 'port 2222' <<< "$cfg" && grep -qx "userknownhostsfile $H/kh" <<< "$cfg" && grep -qx 'stricthostkeychecking true' <<< "$cfg" || bad "ssh -G: the proxied route is not what the route file says: $cfg"
got=$(gate_ssh ssh -n vm-b 'echo "a b"; ls')
want=$(argv /usr/bin/ssh -a; echo "$common"; argv -i "$H/k/hop_key" -n -- ops@hop-host)
[ "$(sed '$d' <<< "$got")" = "$want" ] || bad "hop ssh argv: $got"
[ "$(b64_of <<< "$got")" = 'echo "a b"; ls' ] || bad "hop ssh command does not round-trip: $(tail -n 1 <<< "$got")"
case $(tail -n 1 <<< "$got") in "ssh -o BatchMode=yes -- vm-b 'eval "*) ;; *) bad "hop ssh second hop: $(tail -n 1 <<< "$got")" ;; esac
want=$(argv /usr/bin/ssh -a; echo "$common"; argv -i "$H/k/hop_key" -o "UserKnownHostsFile=$H/kh" -- ops@hop-host)
got=$(gate_ssh ssh vm-bk true); [ "$(sed '$d' <<< "$got")" = "$want" ] || bad "hop ssh argv with its known hosts: $got"
mkdir -p "$T/sbx/dir x"; printf x > "$T/it's a.txt"; printf x > "$T/up.txt"
got=$(gate_ssh scp "$T/it's a.txt" "vm-b:dir x/"); [ "$(grep -c '^/usr/bin/ssh$' <<< "$got")" = 1 ] || bad "hop scp upload is not one ssh: $got"
remote=$(b64_of <<< "$got"); (cd "$T/sbx" && printf y | sh -c "$remote"); [ "$(cat "$T/sbx/dir x/it's a.txt" 2> /dev/null)" = y ] || bad "hop scp upload names another file: $remote"
remote=$(gate_ssh scp "$T/up.txt" 'vm-b:$(touch pwned)' | b64_of); (cd "$T/sbx" && printf y | sh -c "$remote"); [ ! -e "$T/sbx/pwned" ] && [ -f "$T/sbx/\$(touch pwned)" ] || bad "hop scp upload runs its destination: $remote"
gate_ssh scp 'vm-b:a b;touch pwned' "$T/down.txt"; [ "$(b64_of < "$T/down.txt")" = "cat -- 'a b;touch pwned'" ] || bad "hop scp download runs its source: $(tail -n 1 "$T/down.txt")"
E=$T/evil; mkdir -p "$E"; for c in base64 tr ls id dirname basename cut; do printf '#!/bin/sh\necho evil\n' > "$E/$c"; chmod 755 "$E/$c"; done
got=$(env -i PATH="$E:$PATH" HOME="$H" GATE_ROUTES_DIR="$R" GATE_SSH_DRYRUN=1 bash ./gate-ssh.sh ssh vm-b 'echo hi' 2>&1); [ "$(b64_of <<< "$got")" = 'echo hi' ] || bad "gate-ssh takes a tool from PATH: $got"
refused "no route for vm-nope" gate_ssh ssh vm-nope true
for h in vm-def vm-dslash vm-dot vm-other vm-hopdef vm-hopdot; do refused "where the owner's own identities live" gate_ssh ssh $h true; done
refused "$pw/.ssh/gate-ssh-absent-key lies under" env -i PATH="$PATH" HOME=/nonexistent GATE_ROUTES_DIR="$R" GATE_SSH_DRYRUN=1 bash ./gate-ssh.sh ssh vm-pw true
refused "is a symbolic link: name the key file itself" gate_ssh ssh vm-link true
refused "is the same file as" gate_ssh ssh vm-hard true
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
refused "STATION is read only with PROXY" gate_ssh ssh vm-station-alone true
refused "HOSTNAME is not a plain host name" gate_ssh ssh vm-badhost true
refused "USER is not a plain user name" gate_ssh ssh vm-baduser true
for h in vm-port0 vm-portbig vm-portx; do refused "PORT is not a port number" gate_ssh ssh $h true; done
refused "HOP is not user@host" gate_ssh ssh vm-badhop true
refused "is not yours alone" gate_ssh ssh vm-loose true
refused "is a symbolic link: write the file itself" gate_ssh ssh vm-sym true
refused "route folder $T/routes-open is not yours alone" env -i PATH="$PATH" HOME="$H" GATE_ROUTES_DIR="$T/routes-open" GATE_SSH_DRYRUN=1 bash ./gate-ssh.sh ssh vm-a true
refused "route folder /usr is not yours alone" env -i PATH="$PATH" HOME="$H" GATE_ROUTES_DIR=/usr GATE_SSH_DRYRUN=1 bash ./gate-ssh.sh ssh vm-a true
refused "is a symbolic link: name the folder itself" env -i PATH="$PATH" HOME="$H" GATE_ROUTES_DIR="$T/routes-link" GATE_SSH_DRYRUN=1 bash ./gate-ssh.sh ssh vm-a true
refused "no route for vm-a" env -i PATH="$PATH" HOME="$H" GATE_SSH_DRYRUN=1 bash ./gate-ssh.sh ssh vm-a true
refused "is not a plain host name" gate_ssh ssh a/b true
refused "is not a plain host name" gate_ssh ssh user@vm-a true
refused "no route for vm-nope" gate_ssh scp x vm-nope:y
refused "scp names two hosts" gate_ssh scp x vm-a:y vm-b:z
refused "scp between two remote paths" gate_ssh scp vm-a:x vm-a:y
for c in "ssh -i /x vm-a true" "ssh -F /dev/null vm-a true" "ssh -J h vm-a true" "ssh -A vm-a true" "ssh -o IdentityAgent=x vm-a true" "ssh -o ProxyCommand=x vm-a true" "ssh" "scp -r a vm-a:b" "scp -P 1 a vm-a:b" "scp a vm-a:b -o ProxyCommand=x" "scp a b" "bogus"; do
  gate_ssh $c > /dev/null 2>&1; [ $? = 2 ] || bad "an unrouted option is accepted: $c"
done
if command -v sshd > /dev/null && command -v ssh-keygen > /dev/null && command -v python3 > /dev/null; then
  S=$T/sd; mkdir -p "$S"; ssh-keygen -q -t ed25519 -N '' -f "$S/host"; ssh-keygen -q -t ed25519 -N '' -f "$S/client"; cp "$S/client.pub" "$S/authorized_keys"
  port=$(python3 -I -c 'import socket; s = socket.socket(); s.bind(("127.0.0.1", 0)); print(s.getsockname()[1])')
  printf 'Port %s\nListenAddress 127.0.0.1\nHostKey %s\nAuthorizedKeysFile %s\nPidFile %s\nUsePAM no\nStrictModes no\nPasswordAuthentication no\nLogLevel QUIET\n' "$port" "$S/host" "$S/authorized_keys" "$S/pid" > "$S/sshd_config"
  "$(command -v sshd)" -D -f "$S/sshd_config" > /dev/null 2>&1 & sp=$!
  python3 -I -c 'import socket, sys, time; s = socket.socket(socket.AF_UNIX); s.bind(sys.argv[1]); s.listen(8); time.sleep(120)' "$S/silent.sock" & kp=$!
  up=0; for _ in $(seq 50); do (exec 3<> "/dev/tcp/127.0.0.1/$port") 2> /dev/null && { up=1; break; }; command sleep 0.1; done
  if [ $up = 1 ]; then
    base=(-F /dev/null -p "$port" -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o LogLevel=ERROR -o ConnectTimeout=5 -o BatchMode=yes -l "$(id -un)")
    timeout 8 ssh "${base[@]}" -o IdentityAgent="$S/silent.sock" 127.0.0.1 true > /dev/null 2>&1; [ $? = 124 ] || bad "BatchMode bounds a stuck agent: the guard is not the only protection"
    timeout 8 ssh "${base[@]}" -o IdentitiesOnly=yes -o IdentityAgent=none -i "$S/client" 127.0.0.1 true > /dev/null 2>&1; [ $? = 0 ] || bad "the guard options do not reach a server with the dedicated key alone"
  else
    echo "skip: no sshd could listen here (the BatchMode test)"
  fi
  kill "$sp" "$kp" 2> /dev/null; wait "$sp" "$kp" 2> /dev/null
fi
[ $fail = 0 ] && echo "gate scripts: ok"
exit $fail

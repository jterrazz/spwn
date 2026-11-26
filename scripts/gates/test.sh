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
  local p1='(^|[;&|({!]|\$\(|(then|do|until|if|else)[[:space:]])[[:space:]]*(ssh|scp|rsync)[[:space:]]' p2=$'=\\(?["\']?(ssh|scp)[[:space:]"\']'
  grep -nE -e "$p1" -e "$p2" "$@" | grep -vE '^[^:]*:[0-9]+:[[:space:]]*#' | grep -v 'gate-ssh: exempt' | grep -vE 'rsync .*-e "\$GATE_SSH rsh"'
}
bare=(); for f in ./*.sh; do case $f in ./gate-ssh.sh | ./test.sh | ./*-remote.sh) ;; *) bare+=("$f") ;; esac; done
bare_ssh "${bare[@]}" && bad "bare ssh to a VM: a script reaches a VM only through \$GATE_SSH"
F=$(mktemp -d) || exit 2
printf '%s\n' 'ssh vm true' '  scp x vm:y' 'a && ssh -n vm true' 'x=$(ssh vm true)' 'S="ssh -n"' 'rsync -a a vm:b' 'if ssh vm true; then' > "$F/bare.sh"
printf '%s\n' '"$GATE_SSH" ssh vm true' '# ssh vm true' 'ssh vm true  # gate-ssh: exempt (why)' 'rsync -e "$GATE_SSH rsh" a vm:b' 'echo "(hold or ssh)"' 'SSHP=("$GATE_SSH" ssh -n)' > "$F/clean.sh"
[ "$(bare_ssh "$F/bare.sh" | wc -l | tr -d ' ')" = 7 ] || bad "the bare-ssh lint misses a form: $(bare_ssh "$F/bare.sh" | wc -l)"
[ -z "$(bare_ssh "$F/clean.sh")" ] || bad "the bare-ssh lint flags a routed line: $(bare_ssh "$F/clean.sh")"
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
H=$T/home; R=$T/routes; mkdir -p "$H/.ssh" "$H/k" "$R"
: > "$H/k/vm_key"; : > "$H/k/hop_key"; : > "$H/.ssh/id_ed25519"
printf 'KEY=%s\n' "$H/k/vm_key" > "$R/vm-a.env"
printf 'KEY=~/.ssh/id_ed25519\n' > "$R/vm-def.env"
printf 'HOP=ops@hop-host\nHOP_KEY=%s\n' "$H/k/hop_key" > "$R/vm-b.env"
printf 'KEY=%s\nHOP=ops@hop-host\n' "$H/k/vm_key" > "$R/vm-mixed.env"
printf 'KEY=%s\nAGENT=yes\n' "$H/k/vm_key" > "$R/vm-extra.env"
printf 'KEY=%s\n' "$H/k/vm_key" > "$R/vm-loose.env"; chmod 666 "$R/vm-loose.env"
chmod 600 "$R"/vm-a.env "$R"/vm-def.env "$R"/vm-b.env "$R"/vm-mixed.env "$R"/vm-extra.env
gate_ssh() { env -i PATH="$PATH" HOME="$H" GATE_ROUTES_DIR="$R" GATE_SSH_DRYRUN=1 bash ./gate-ssh.sh "$@"; }
guard=$(printf "%s\n" -o BatchMode=yes -o IdentitiesOnly=yes -o IdentityAgent=none)
want=$(printf '/usr/bin/ssh\n%s\n-i\n%s\n-n\n-o\nConnectTimeout=10\nvm-a\ntrue\n' "$guard" "$H/k/vm_key")
got=$(gate_ssh ssh -n -o ConnectTimeout=10 vm-a true); [ "$got" = "$want" ] || bad "direct ssh argv: $got"
got=$(gate_ssh rsh vm-a true); want=$(printf '/usr/bin/ssh\n%s\n-i\n%s\nvm-a\ntrue\n' "$guard" "$H/k/vm_key"); [ "$got" = "$want" ] || bad "direct rsh argv: $got"
got=$(gate_ssh scp -q a.txt vm-a:dir/); want=$(printf '/usr/bin/scp\n%s\n-i\n%s\n-q\na.txt\nvm-a:dir/\n' "$guard" "$H/k/vm_key"); [ "$got" = "$want" ] || bad "direct scp argv: $got"
got=$(gate_ssh ssh -n vm-b 'echo "a b"; ls'); want=$(printf '/usr/bin/ssh\n%s\n-i\n%s\n-n\nops@hop-host\n' "$guard" "$H/k/hop_key")
[ "$(echo "$got" | sed '$d')" = "$want" ] || bad "hop ssh argv: $got"
last=$(echo "$got" | tail -n 1); b64=$(echo "$last" | sed -n "s/.*printf %s \([A-Za-z0-9+\/=]*\) | base64 -d.*/\1/p")
[ "$(printf %s "$b64" | base64 -d)" = 'echo "a b"; ls' ] || bad "hop ssh command does not round-trip: $last"
case $last in "ssh -o BatchMode=yes vm-b "*) ;; *) bad "hop ssh second hop: $last" ;; esac
printf x > "$T/up.txt"
got=$(gate_ssh scp "$T/up.txt" vm-b:dir/ | grep -c '^/usr/bin/ssh$'); [ "$got" = 1 ] || bad "hop scp upload is not one ssh: $got"
gate_ssh scp vm-b:dir/f.txt "$T/down.txt"; [ "$(head -n 1 "$T/down.txt")" = /usr/bin/ssh ] || bad "hop scp download: $(cat "$T/down.txt")"
for c in "ssh vm-nope true" "ssh vm-def true" "ssh vm-mixed true" "ssh vm-extra true" "ssh vm-loose true" "ssh a/b true" "ssh user@vm-a true" "scp x vm-nope:y" "scp x vm-a:y vm-b:z"; do
  gate_ssh $c > /dev/null 2>&1; [ $? = 92 ] || bad "a refused route does not exit 92: $c"
done
for c in "ssh -i /x vm-a true" "ssh -F /dev/null vm-a true" "ssh -J h vm-a true" "ssh -A vm-a true" "ssh -o IdentityAgent=x vm-a true" "ssh -o ProxyCommand=x vm-a true" "ssh" "scp -r a vm-a:b" "scp -P 1 a vm-a:b" "scp a b" "bogus"; do
  gate_ssh $c > /dev/null 2>&1; [ $? = 2 ] || bad "an unrouted option is accepted: $c"
done
env -i PATH="$PATH" HOME="$H" GATE_SSH_DRYRUN=1 bash ./gate-ssh.sh ssh vm-a true > /dev/null 2>&1; [ $? = 92 ] || bad "no route folder does not exit 92"
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

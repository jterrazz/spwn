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
sb() { local d=$T/$1; rm -rf "$d"; mkdir -p "$d/bin" "$d/home" "$d/run" "$d/r/scripts"; cp -R . "$d/r/scripts/gates"
  [ -n "${NOFOLDER:-}" ] || mkdir -p "$d/home/spwn-gates"
  cat > "$d/bin/ssh" <<'EOF'
#!/bin/bash
while [ $# -gt 0 ]; do case $1 in -o | -i) shift 2 ;; -*) shift ;; *) break ;; esac; done
shift
[ -e "$HOME/down" ] && exit 255
echo "$*" >> "$HOME/ssh.calls"
exec bash -c "$*"
EOF
  cat > "$d/bin/scp" <<'EOF'
#!/bin/bash
a=(); while [ $# -gt 0 ]; do case $1 in -o | -i) shift 2 ;; -*) shift ;; *) a+=("$1"); shift ;; esac; done
[ -e "$HOME/down" ] && exit 255
m() { case $1 in *:*) case ${1#*:} in /*) echo "${1#*:}" ;; *) echo "$HOME/${1#*:}" ;; esac ;; *) echo "$1" ;; esac; }
dst=${a[${#a[@]}-1]}; unset 'a[${#a[@]}-1]'
for s in "${a[@]}"; do cp "$(m "$s")" "$(m "$dst")" || exit 1; done
EOF
  printf '#!/bin/bash\nexec /bin/sleep 0.2\n' > "$d/bin/sleep"; chmod +x "$d/bin/ssh" "$d/bin/scp" "$d/bin/sleep"
  git init -q -b b "$d/w"; local g=(-c user.name=t -c user.email=t@t); git -C "$d/w" "${g[@]}" commit -q --allow-empty -m c0; git -C "$d/w" "${g[@]}" commit -q --allow-empty -m c1
}
in_sb() { local d=$T/$1; shift; ( cd "$d/r/scripts/gates" && HOME=$d/home PATH=$d/bin:$PATH GATE_RUN_DIR=$d/run GATE_VM_LINUX=stubvm GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_SYSTEM=/dev/null "$@" ); }
dp_case() { local n=$1 want_rc=$2 want=$3; shift 3
  sb "$n"; printf '%s\n' "$@" > "$T/$n/home/dp-stages"
  printf '#!/bin/bash\ncat "$HOME/dp-stages"\n' > "$T/$n/r/scripts/gates/desktop-proof-remote.sh"
  out=$(in_sb "$n" bash desktop-proof.sh "$T/$n/w" lab b 2>&1); rc=$?
  [ $rc = "$want_rc" ] || bad "desktop-proof $n: exit $rc, wanted $want_rc"
  case $out in *"VERDICT lab $(git -C "$T/$n/w" rev-parse --short=9 b) $want"*) ;; *) bad "desktop-proof $n: no 'VERDICT lab <sha> $want' in: $(echo "$out" | tail -2)" ;; esac
}
dp_case dp-green 0 "GREEN steps=-" NPMCI=0 TSC=0 VITEST=0 PROOF=0 WEBSITE=0 TSCHECK=0 ARCH=0 MAKELINT=0 DT-DONE
dp_case dp-red 1 "RED steps=WEBSITE" NPMCI=0 TSC=0 VITEST=0 PROOF=0 WEBSITE=1 TSCHECK=0 ARCH=0 MAKELINT=0 DT-DONE
dp_case dp-two 1 "RED steps=TSC,MAKELINT" NPMCI=0 TSC=2 VITEST=0 PROOF=0 WEBSITE=0 TSCHECK=0 ARCH=0 MAKELINT=1 DT-DONE
dp_case dp-cut 1 "RED steps=WEBSITE,TSCHECK,ARCH,MAKELINT,DT-DONE" NPMCI=0 TSC=0 VITEST=0 PROOF=0
[ $fail = 0 ] && echo "gate scripts: ok"
exit $fail

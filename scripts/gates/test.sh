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
  printf '#!/bin/bash\nexec /bin/sleep 0.2\n' > "$d/bin/sleep"
  cat > "$d/bin/df" <<'EOF'
#!/bin/bash
for p; do :; done
case $p in "$HOME"/ram*) k=ram; s=ramfs ;; *) k=disk; s=vda1 ;; esac
v=$(cat "$HOME/df.$k" 2>/dev/null || echo 10)
case $* in *--output=source) printf "Filesystem\n%s\n" "$s" ;; *--output=pcent*) printf 'Use%%\n %s%%\n' "$v" ;; *--output=source,pcent*) printf 'Filesystem Use%%\n%s %s%%\n' "$s" "$v" ;; *) echo "$s 1G 1G 1G $v% $p" ;; esac
EOF
  chmod +x "$d/bin/ssh" "$d/bin/scp" "$d/bin/sleep" "$d/bin/df"
  git init -q -b b "$d/w"; local g=(-c user.name=t -c user.email=t@t); git -C "$d/w" "${g[@]}" commit -q --allow-empty -m c0; git -C "$d/w" "${g[@]}" commit -q --allow-empty -m c1
}
in_sb() { local d=$T/$1; shift; ( cd "$d/r/scripts/gates" && HOME=$d/home PATH=$d/bin:$PATH GATE_RUN_DIR=$d/run GATE_VM_LINUX=stubvm GATE_VM_RAM=$d/home/ram GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_SYSTEM=/dev/null "$@" ); }
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
tools_sb() { local d=$T/$1 t; mkdir -p "$d/home/.local/bin"
  for t in mise npm npx go pnpm flock; do printf '#!/bin/bash\nexit 0\n' > "$d/bin/$t"; done
  printf '#!/bin/bash\nexit $(cat "$HOME/make-$1.rc" 2>/dev/null || echo 0)\n' > "$d/bin/make"
  printf '#!/bin/bash\nshift\nexec "$@"\n' > "$d/home/.local/bin/heavy-run-excl"
  chmod +x "$d/bin/"* "$d/home/.local/bin/heavy-run-excl"
}
cold_case() { local n=$1 want_rc=$2; shift 2
  sb "$n"; tools_sb "$n"; for k in "$@"; do echo 1 > "$T/$n/home/make-$k.rc"; done
  in_sb "$n" bash cold-gate.sh "$T/$n/w" b lab > /dev/null 2>&1; rc=$?
  [ $rc = "$want_rc" ] || bad "cold-gate $n: exit $rc, wanted $want_rc: $(tail -3 "$T/$n/run/station-lab-cold.log" 2>/dev/null)"
}
cold_case cold-green 0
cold_case cold-generate 1 generate
cold_case cold-check 1 check
cold_case cold-lint 1 lint
tip_case() { local n=$1 want_rc=$2 want=$3 mb=$4; sb "$n"; local w=$T/$n/w g=(-c user.name=t -c user.email=t@t)
  mkdir -p "$w/machines/specs/machine/menubar"; printf '#!/bin/bash\n[ "$2" = --update-goldens ] && exit %s\nexit 0\n' "$mb" > "$w/machines/specs/machine/menubar/run.sh"
  printf '#!/bin/bash\nexit 0\n' > "$T/$n/r/scripts/gates/darwin-build.sh"; chmod +x "$w/machines/specs/machine/menubar/run.sh"
  git -C "$w" add -A; git -C "$w" "${g[@]}" commit -q -m m; out=$(in_sb "$n" env GATE_SLOT_MAX_LOAD=100000 bash tip-mac.sh "$w" "$(git -C "$w" rev-parse --short=9 HEAD)" lab 2>&1); rc=$?
  [ $rc = "$want_rc" ] || bad "tip-mac $n: exit $rc, wanted $want_rc"
  case $out in *"VERDICT lab $(git -C "$w" rev-parse --short=9 HEAD) $want"*) ;; *) bad "tip-mac $n: no 'VERDICT lab <sha> $want' in: $(echo "$out" | tail -2)" ;; esac
}
tip_case tip-green 0 "GREEN steps=-" 0
tip_case tip-red 1 "RED steps=MB-UPDATE1,MB-UPDATE2" 1
regen_case() { local n=$1 want_rc=$2 mode=$3 rr=$4; sb "$n"; local w=$T/$n/w s; s=$(git -C "$w" rev-parse HEAD)
  printf '#!/bin/bash\ncd "$CC_HOME" && tar -cf "$1-regen-r.tar" -T /dev/null && : > "$1-regen-changes.txt"\nexit %s\n' "$rr" > "$T/$n/r/scripts/gates/regen-remote.sh"
  if [ $mode = one ]; then in_sb "$n" bash regen.sh one "$w" "$s" lab > /dev/null 2>&1; rc=$?
  else echo "1 $s" > "$T/$n/map"; in_sb "$n" bash regen.sh all "$w" "$T/$n/map" lab "1" "" > /dev/null 2>&1; rc=$?; fi
  [ $rc = "$want_rc" ] || bad "regen $mode $n: exit $rc, wanted $want_rc"
}
regen_case regen-one-green 0 one 0
regen_case regen-one-red 3 one 3
regen_case regen-all-green 0 all 0
regen_case regen-all-red 1 all 3
cc_sb() { sb "$1"
  cat > "$T/$1/r/scripts/gates/cc-remote.sh" <<'EOF'
#!/bin/bash
echo "CHEAP-SEEN[$CHEAP]" >> "$HOME/cheap.seen"; echo "$CC_WORK" > "$HOME/work.seen"
echo "GOFMT[]"; echo VET-OK; echo SPECSVET-OK; echo LINT=$(cat "$HOME/lint.rc" 2>/dev/null || echo 0); echo "CC-DONE $3"
EOF
}
cc_run() { local n=$1; shift; in_sb "$n" timeout 120 "$@" bash cc.sh "$T/$n/w" "$(git -C "$T/$n/w" rev-parse b~1)" lab 2>&1; }
cheap_case() { local v=$1 want_rc=$2 tier=$3 seen=$4 n=cheap-$1; cc_sb "$n"
  if [ "$v" = unset ]; then out=$(cc_run "$n" env -u CHEAP); else out=$(cc_run "$n" env CHEAP="$v"); fi; rc=$?
  [ $rc = "$want_rc" ] || bad "cc.sh CHEAP=$v: exit $rc, wanted $want_rc"
  case $tier in
    refused) case $out in *"bad CHEAP=$v"*) ;; *) bad "cc.sh CHEAP=$v: no refusal sentence in: $out" ;; esac; [ ! -e "$T/$n/home/cheap.seen" ] || bad "cc.sh CHEAP=$v: refused but ran" ;;
    *) case $out in *"launched lab on stubvm"*": tier=$tier commits"*) ;; *) bad "cc.sh CHEAP=$v: launch line has no tier=$tier: $out" ;; esac
       case $out in *"VERDICT lab 1 "*" tier=$tier"*) ;; *) bad "cc.sh CHEAP=$v: VERDICT line has no tier=$tier" ;; esac
       [ "$(cat "$T/$n/home/cheap.seen" 2>/dev/null)" = "CHEAP-SEEN[$seen]" ] || bad "cc.sh CHEAP=$v: the loop saw $(cat "$T/$n/home/cheap.seen" 2>/dev/null)" ;;
  esac
}
cheap_case 1 0 cheap 1
cheap_case true 0 cheap 1
cheap_case yes 0 cheap 1
cheap_case 0 0 heavy ""
cheap_case false 0 heavy ""
cheap_case no 0 heavy ""
cheap_case "" 0 heavy ""
cheap_case unset 0 heavy ""
cheap_case maybe 2 refused ""
cheap_case 2 2 refused ""
NOFOLDER=1 dp_case dp-nofolder 0 "GREEN steps=-" NPMCI=0 TSC=0 VITEST=0 PROOF=0 WEBSITE=0 TSCHECK=0 ARCH=0 MAKELINT=0 DT-DONE
NOFOLDER=1 cheap_case 0 0 heavy ""
NOFOLDER=1 cold_case cold-nofolder 0
NOFOLDER=1 regen_case regen-nofolder 0 one 0
loop_case() { local n=$1 mode=$2 want_rc=$3 want=$4; cc_sb "$n"; echo "$mode" > "$T/$n/home/loop.mode"
  cat > "$T/$n/r/scripts/gates/cc-loop.sh" <<'EOF'
#!/bin/bash
L=$1
case $(cat "$HOME/loop.mode") in
  down) : > "$HOME/down"; exit 0 ;;
  nosummary) echo "VERDICT $L 1 abc GREEN - took=1s tier=heavy" ;;
  short) echo "VERDICT $L 1 abc GREEN - took=1s tier=heavy"; echo "SUMMARY $L: 2 commits, 1 GREEN, 0 FLAKE, 0 RED" ;;
  red) echo "VERDICT $L 1 abc RED steps=LINT took=1s tier=heavy"; echo "SUMMARY $L: 1 commits, 0 GREEN, 0 FLAKE, 1 RED" ;;
  flake) echo "VERDICT $L 1 abc FLAKE(x) rerun=green took=1s tier=heavy"; echo "SUMMARY $L: 1 commits, 0 GREEN, 1 FLAKE, 0 RED" ;;
esac
echo END > "$CC_HOME/$L-cc.end"
EOF
  out=$(cc_run "$n" env -u CHEAP); rc=$?
  [ $rc = "$want_rc" ] || bad "cc.sh loop $mode: exit $rc, wanted $want_rc"
  case $out in *"$want"*) ;; *) bad "cc.sh loop $mode: no '$want' in: $(echo "$out" | tail -2)" ;; esac
}
loop_case loop-red red 1 "SUMMARY lab: 1 commits, 0 GREEN, 0 FLAKE, 1 RED"
loop_case loop-flake flake 0 "1 FLAKE"
loop_case loop-nosummary nosummary 94 "INCOMPLETE lab"
loop_case loop-short short 94 "INCOMPLETE lab"
loop_case loop-down down 93 "ATTACH=1 bash"
cc_sb attach; mkdir -p "$T/attach/home/spwn-gates"
printf '%s\n' "VERDICT lab 1 abc GREEN - took=1s tier=heavy" "SUMMARY lab: 1 commits, 1 GREEN, 0 FLAKE, 0 RED" > "$T/attach/home/spwn-gates/lab-cc.log"; echo END > "$T/attach/home/spwn-gates/lab-cc.end"
out=$(in_sb attach timeout 120 env ATTACH=1 bash cc.sh "$T/attach/w" "$(git -C "$T/attach/w" rev-parse b~1)" lab 2>&1); rc=$?
[ $rc = 0 ] || bad "cc.sh ATTACH=1 on a finished loop: exit $rc"
ram_case() { local n=$1 ram=$2 disk=$3 want_rc=$4 want_work=$5 want=$6; cc_sb "$n"; echo "$ram" > "$T/$n/home/df.ram"; echo "$disk" > "$T/$n/home/df.disk"
  out=$(cc_run "$n" env -u CHEAP); rc=$?
  [ $rc = "$want_rc" ] || bad "cc.sh ram $ram disk $disk: exit $rc, wanted $want_rc"
  [ "$(cat "$T/$n/home/work.seen" 2>/dev/null)" = "${want_work/#@/$T/$n/home/}" ] || bad "cc.sh ram $ram disk $disk: clones under $(cat "$T/$n/home/work.seen" 2>/dev/null), wanted $want_work"
  case $out in *"$want"*) ;; *) bad "cc.sh ram $ram disk $disk: no '$want' in: $(echo "$out" | head -4)" ;; esac
}
ram_case ram-ok 50 10 0 @ram "VERDICT lab 1"
ram_case ram-full 97 10 0 @spwn-gates/clones "RAM-DISK"
ram_case ram-disk-full 97 95 86 "" "DISK stubvm at 95%"
[ $fail = 0 ] && echo "gate scripts: ok"
exit $fail

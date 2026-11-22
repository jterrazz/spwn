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
[ $fail = 0 ] && echo "gate scripts: ok"
exit $fail

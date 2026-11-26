#!/bin/bash
# invariant: a waiter never takes the slot while a live waiter of a higher priority waits: each waiter marks itself in $RUN/gate-slots/wanted/<pid> with its priority on the first line.
. "$(dirname "$0")/env.sh"
D=$RUN/gate-slots
mkdir -p "$D/wanted"
rank() { case $1 in landing) echo 3 ;; low) echo 1 ;; normal | "") echo 2 ;; *) echo "gate-slot: unknown priority $1 (landing, normal or low)" >&2; return 1 ;; esac; }
prio=${GATE_SLOT_PRIORITY:-normal}; me=$(rank "$prio") || exit 2
[ -z "${GATE_SLOT_VM:-}" ] || gate_route "$GATE_SLOT_VM"
{ echo "$prio"; echo "$*" | cut -c1-200; } > "$D/wanted/$$"
trap 'rm -f "$D/wanted/$$"' EXIT
load() {
  if [ -r /proc/loadavg ]; then l=$(awk '{print int($1)}' /proc/loadavg); else l=$(sysctl -n vm.loadavg | awk '{print int($2)}'); fi
  if [ -n "${GATE_SLOT_VM:-}" ]; then
    v=$("$GATE_SSH" ssh -n -o ConnectTimeout=10 "$GATE_SLOT_VM" cat /proc/loadavg 2>/dev/null | awk '{print int($1)}')
    [ "${v:-0}" -gt "$l" ] && l=$v
  fi
  echo "$l"
}
higher_waiting() {
  for w in "$D"/wanted/*; do
    [ -e "$w" ] || continue
    p=${w##*/}
    if kill -0 "$p" 2>/dev/null; then
      [ "$(rank "$(head -n1 "$w")" 2>/dev/null || echo 0)" -gt "$me" ] && return 0
    else rm -f "$w"; fi
  done
  return 1
}
# value: the RAM cap is 50 GB, the Linux VM (42) plus one other VM: a command that starts a tart rig (menubar check, darwin build) waits while any other VM runs.
rig_room() {
  case "$*" in *darwin-build*|*menubar/run.sh*) ;; *) return 0 ;; esac
  command -v tart >/dev/null || return 0
  [ -z "$(tart list 2>/dev/null | awk -v vm="$VM_LINUX" 'NR>1 && $NF=="running" && $2!=vm')" ]
}
n=0
while :; do
  for s in 1; do
    l=$(load)
    if higher_waiting; then
      :
    elif ! rig_room "$@"; then
      :
    elif [ "$l" -lt "${GATE_SLOT_MAX_LOAD:-20}" ] && mkdir "$D/$s" 2>/dev/null; then
      rm -f "$D/wanted/$$"
      echo $$ > "$D/$s/pid"; date +%H:%M > "$D/$s/since"; echo "$*" | cut -c1-300 > "$D/$s/cmd"
      trap 'rm -rf "$D/$s"; rm -f "$D/wanted/$$"' EXIT
      echo "gate-slot $s taken $(date +%H:%M)"; "$@"; exit $?
    fi
    p=$(cat "$D/$s/pid" 2>/dev/null); [ -n "$p" ] && ! kill -0 "$p" 2>/dev/null && rm -rf "$D/$s"
    [ $((n % 20)) -eq 0 ] && echo "gate-slot waiting $(date +%H:%M) (priority $prio): held since $(cat "$D/$s/since" 2>/dev/null) by $(cat "$D/$s/cmd" 2>/dev/null)"
  done
  n=$((n + 1))
  command sleep "${GATE_SLOT_POLL:-30}"
done

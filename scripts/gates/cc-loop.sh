#!/bin/bash
# invariant: <label>-cc.end is written last, whatever ends the loop, and <label>-cc.pid exists only while it runs.
export PATH=$HOME/.local/bin:$PATH
L=$1; B=$2; shift 2
R=${CC_HOME:?CC_HOME is the folder of the files}; W=${CC_WORK:?CC_WORK is the folder of the clones}; PAR=${PAR:-$([ -n "$CHEAP" ] && echo 4 || echo 1)}
items=$R/$L-cc.items; pidf=$R/$L-cc.pid
o=$(cat "$pidf" 2>/dev/null); if [ -n "$o" ] && kill -0 "$o" 2>/dev/null; then echo "refused: $L already runs loop pid $o"; exit 88; fi
rm -f "$R/$L-cc.end"; echo $$ > "$pidf"
trap 'kill $(jobs -p) 2>/dev/null; rm -rf "${W:?}/$L-node"; rm -f "$pidf"; echo "END $(date +%T)" > "$R/$L-cc.end"' EXIT
trap 'exit 143' TERM HUP INT
disk_ok(){ local d u; for d in ~ "$W"; do mkdir -p "$d"; u=$(df --output=pcent "$d" | tail -1 | tr -dc 0-9); [ "${u:-100}" -lt 90 ] || { echo "DISK $d at ${u}%: free it (old *-node, clones, logs) before rerunning"; exit 86; }; done; }
disk_ok
one(){ local n=$1 sha=$2 step=$3 suffix=$4 f=$R/$L-cc$1 t0=$SECONDS; shift 4
  rm -f "$f$suffix.out" "$f$suffix.rc"; [ -z "$suffix" ] && rm -f "$R/$L-drift$n.patch"
  CC_HOME=$R CC_WORK=$W CC_SHARED=1 CHEAP=$CHEAP ONLY=$step UPD=$UPD bash "$R/$L-cc-remote.sh" "$L" "$B" "$sha" "$n" "$@" > "$f$suffix.out" 2>&1; local rc=$?; echo $((SECONDS-t0)) > "$f$suffix.secs"; echo $rc > "$f$suffix.rc"
}
off=0
flush(){ local sz; sz=$(stat -c %s "$1" 2>/dev/null || echo 0); [ "$sz" -gt "$off" ] && { tail -c +$((off+1)) "$1" | head -c $((sz-off)); off=$sz; }; return 0; }
wait_rc(){ local f=$1 n=$2 t=0; off=0
  until [ -e "$f.rc" ]; do flush "$f.out"; sleep 5; t=$((t+5)); [ $((t % 300)) = 0 ] && echo "... $L commit $n running $(date +%T)"; done
  flush "$f.out"
}
mapfile -t N < <(awk '{print $1}' "$items"); mapfile -t S < <(awk '{print $2}' "$items")
total=${#N[@]}; declare -a pid; next=0; emitted=0; g=0; fl=0; rd=0
while [ $emitted -lt $total ]; do
  while [ $next -lt $total ] && [ $((next-emitted)) -lt "$PAR" ]; do
    disk_ok; one "${N[$next]}" "${S[$next]}" "$ONLY" "" "$@" & pid[$next]=$!; next=$((next+1))
  done
  i=$emitted; n=${N[$i]}; sha=${S[$i]}; f=$R/$L-cc$n
  echo "=== $n $sha"
  wait_rc "$f" "$n"; wait "${pid[$i]}" 2>/dev/null
  rc=$(cat "$f.rc"); echo "REMOTE-RC=$rc commit $n"
  { cat "$f.out"; echo "REMOTE-RC=$rc commit $n"; } > "$f.blk"
  v=$(python3 -I "$R/$L-cc-verdict.py" --block "$f.blk" --patch "$R/$L-drift$n.patch" ${ONLY:+--only "$ONLY"}); vv=${v%% *}; why=${v#* }
  case $vv in
  FLAKE*)
    st=; case $why in *DRIFT*) st=GEN ;; esac; case $why in *STORIES*) st=${st:+$st,}STORIES ;; esac
    echo "--- $L commit $n: $vv, rerun of $st alone"
    one "$n" "$sha" "$st" 2 "$@"; wait_rc "${f}2" "$n"; rc2=$(cat "${f}2.rc")
    { cat "${f}2.out"; echo "REMOTE-RC=$rc2 commit $n"; } > "${f}2.blk"
    v2=$(python3 -I "$R/$L-cc-verdict.py" --block "${f}2.blk" --patch "$R/$L-drift$n.patch" --only "$st"); vv2=${v2%% *}
    if [ "$vv2" = GREEN ]; then v="$vv $why rerun=green"; fl=$((fl+1)); else v="RED $why rerun=$vv2 ($v2)"; rd=$((rd+1)); fi ;;
  GREEN) g=$((g+1)) ;;
  *) rd=$((rd+1)) ;;
  esac
  echo "VERDICT $L $n ${sha:0:9} $v took=$(cat "$f.secs" 2>/dev/null)s"
  rm -f "$f.out" "$f.rc" "$f.secs" "$f.blk" "${f}2.out" "${f}2.rc" "${f}2.blk"
  emitted=$((emitted+1))
done
echo "SUMMARY $L: $total commits, $g GREEN, $fl FLAKE, $rd RED"
rm -f "$R/$L-cc.bundle" "$R/$L-cc-remote.sh" "$R/$L-cc-verdict.py" "$items"

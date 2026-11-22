#!/bin/bash
[ $# -eq 2 ] || { echo "usage: ratchet.sh <worktree> <base>" >&2; exit 2; }
W=$1; B=$2
for c in $(git -C "$W" rev-list --reverse "$B..HEAD"); do
  bl=$(git -C "$W" show "$c:machines/.studio/baseline.json" 2>/dev/null | python3 -c 'import json,sys; print(json.load(sys.stdin).get("test",{}).get("pixel-goldens/bytes","none"))')
  sum=$(git -C "$W" ls-tree -r -l "$c" machines | awk '$5 ~ /__expected__\/.*\.png$/ {s+=$4} END{print s+0}')
  { [ "$bl" = "$sum" ] || { [ "$bl" = none ] && [ "$sum" = 0 ]; }; } && v=EQ || v=DIFF; echo "$(git -C "$W" log -1 --format=%h "$c") bl=$bl png=$sum $v"
done

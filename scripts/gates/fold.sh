#!/bin/bash
. "$(dirname "$0")/env.sh"
[ $# -eq 2 ] || { echo "usage: fold.sh <old sha> <label>" >&2; exit 2; }
OLD=$1; O=$RUN/$2-regen-out; SKIP='(^|/)(go\.mod|go\.sum|go\.work\.sum)$'
[ -f $O/changes.txt ] || { echo "fold $2: no changes.txt"; exit 1; }
GEN=${GEN_DIRS:-"machines/apps/desktop/src/fixtures machines/packages/api"}
git restore --source=$OLD --staged --worktree -- $GEN || exit 1
git clean -fdq -- $GEN || exit 1
n=0
while IFS=$'\t' read -r st f; do
  echo "$f" | grep -qE "$SKIP" && continue
  case $st in
    A|M) mkdir -p "$(dirname "$f")"; cp "$O/$f" "$f" || exit 1; git add -- "$f" || exit 1 ;;
    D) git rm -q -- "$f" || exit 1 ;;
    R*) a=${f%%$'\t'*}; b=${f#*$'\t'}; git rm -q -- "$a" || exit 1; mkdir -p "$(dirname "$b")"; cp "$O/$b" "$b" || exit 1; git add -- "$b" || exit 1; n=$((n+1)) ;;
    *) echo "fold $2: status $st for $f"; exit 1 ;;
  esac
  n=$((n+1))
done < $O/changes.txt
git commit -q --amend --no-edit || exit 1
d=$(git diff --name-only $OLD HEAD -- $GEN | wc -l | tr -d ' '); o=$(git diff --name-only $OLD HEAD -- . $(for g in $GEN; do echo ":!$g"; done) | tr "\n" " ")
echo "FOLD $2 $OLD -> $(git rev-parse --short=9 HEAD): $d files differ from old, regen applied $n; outside generated: [$o]"
g=$(grep -cE "	(machines/apps/desktop/src/fixtures|machines/packages/api)/" $O/changes.txt); [ "$d" = "$g" ] || exit 1

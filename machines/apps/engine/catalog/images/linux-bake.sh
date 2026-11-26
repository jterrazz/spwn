#!/bin/bash
# invariant: The image phase: run as VM_USER (bash -s), its sudo passwordless, once the
# plumbing ran and the build put the Machinefile's chain and lock into the user's
# folder. Its input is MACHINEFILE, the template's name; it prints the lock the run
# wrote after the line `--- station lock`, or, when the apply failed, each line it
# marked failed after the line `--- station failed`, its cells without the mark.
set -euo pipefail

page=$(mktemp)
status=0
/usr/local/bin/station machinefile apply --machinefile "$MACHINEFILE" --phase image </dev/null >"$page" || status=$?
cat "$page"
if [ "$status" -ne 0 ]; then
    echo '--- station failed'
    grep "^✗"$'\t' "$page" | cut -f2- || true
fi
rm -f "$page"
[ "$status" -eq 0 ] || exit "$status"
echo '--- station lock'
cat "$HOME/.config/station/machinefiles/$MACHINEFILE.lock" 2>/dev/null || true

#!/bin/bash
# invariant: The image phase: run as VM_USER (bash -s), its sudo passwordless, once the
# plumbing ran and the build put the Machinefile's chain and lock into the user's
# folder. Its input is MACHINEFILE, the template's name; it prints the lock the run
# wrote after the line `--- station lock`.
set -euo pipefail

/usr/local/bin/station apply --machinefile "$MACHINEFILE" --phase image </dev/null
echo '--- station lock'
cat "$HOME/.config/station/machinefiles/$MACHINEFILE.lock" 2>/dev/null || true

#!/bin/bash
GATE_ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
GATE_DIR=$GATE_ROOT/scripts/gates
GATE_SSH=$GATE_DIR/gate-ssh.sh
RUN=${GATE_RUN_DIR:-$GATE_ROOT/.artifacts/gates}
VM_LINUX=${GATE_VM_LINUX:-development-linux}
VM_DIR=${GATE_VM_DIR:-spwn-gates}
VM_RAM=${GATE_VM_RAM:-/gates/$VM_DIR}
mkdir -p "$RUN"
gate_route() { GATE_SSH_DRYRUN=1 "$GATE_SSH" ssh "$1" true > /dev/null || exit $?; }

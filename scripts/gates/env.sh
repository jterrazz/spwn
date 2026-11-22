#!/bin/bash
GATE_ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
GATE_DIR=$GATE_ROOT/scripts/gates
RUN=${GATE_RUN_DIR:-$GATE_ROOT/.artifacts/gates}
VM_LINUX=${GATE_VM_LINUX:-development-linux}
VM_DIR=${GATE_VM_DIR:-spwn-gates}
mkdir -p "$RUN"

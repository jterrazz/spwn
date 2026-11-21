#!/bin/bash
# usage: . env.sh  (sourced by every launcher that runs on the workstation; never run alone)
# The one place the gate scripts read their environment. A run's files live in RUN, the VM's names and folders are parameters:
#   GATE_RUN_DIR    where a run writes its logs, end files, locks and holds (default <checkout>/.artifacts/gates)
#   GATE_VM_LINUX   the shared Linux VM, an ssh host (default development-linux)
#   GATE_VM_DIR     the folder of the gates' files in the VM's home (default spwn-gates)
GATE_ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
GATE_DIR=$GATE_ROOT/scripts/gates
RUN=${GATE_RUN_DIR:-$GATE_ROOT/.artifacts/gates}
VM_LINUX=${GATE_VM_LINUX:-development-linux}
VM_DIR=${GATE_VM_DIR:-spwn-gates}
mkdir -p "$RUN"

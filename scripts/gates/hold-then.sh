#!/bin/bash
# usage: hold-then.sh <vm> <command...>  marks the VM held (cc.sh loops wait) only once the gate slot is ours, then runs the command
. "$(dirname "$0")/env.sh"
mkdir -p "$RUN/vm-hold"; touch "$RUN/vm-hold/$1" "$RUN/vm-hold/all"; shift; exec "$@"

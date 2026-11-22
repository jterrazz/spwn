#!/bin/bash
. "$(dirname "$0")/env.sh"
mkdir -p "$RUN/vm-hold"; touch "$RUN/vm-hold/$1" "$RUN/vm-hold/all"; shift; exec "$@"

#!/usr/bin/env bash
# value: usage: wait-for.sh <pid-file> [<seconds> [<log>]]: waits, bounded (500 s by default, under an ssh call's
# value: limit), for the job whose pid a background start wrote (`nohup … & echo $! > <pid-file>`), then prints the
# value: tail of <log>. By pid, never by `pgrep -f`, whose pattern matches the waiting command's own line.
# value: Exit 0 once the job has ended, 124 when it still runs.
set -uo pipefail
pid=$(cat "$1")
for _ in $(seq 1 "${2:-500}"); do
    kill -0 "$pid" 2>/dev/null || break
    sleep 1
done
[ -z "${3:-}" ] || tail -n 20 "$3"
if kill -0 "$pid" 2>/dev/null; then
    echo "pid $pid still runs" >&2
    exit 124
fi

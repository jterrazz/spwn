#!/bin/sh
# value: the stand-in of a VM's run in a launchd job: a parent that starts a child in its
# own process group and waits, the child writing a beat file named for its pid every
# second. Its one argument is the folder the files live in. Tier 3 copies it into the guest.
dir=$1
mkdir -p "$dir"
/bin/sh -c 'while :; do date +%s > "$0/beat.$$"; sleep 1; done' "$dir" &
echo $! > "$dir/child.pid"
wait

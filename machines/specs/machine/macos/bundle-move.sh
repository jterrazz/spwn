#!/bin/bash
# value: tier 3's move of a Mac from the bare jobs to Spwn.app's, played inside the guest:
# mode B placed (the binary on the PATH, its engine kept by launchd, a standalone menu bar
# running), then Spwn.app opened the way its owner opens it, so the window's own engine
# start makes the move (ADR-073). It reads /tmp/station-e2e/{station,Spwn.app,Station.app};
# each `miss <what>` line it prints is an assertion lost, a `setup <what>` line and exit 1
# a step before any assertion.
set -uo pipefail
e2e=/tmp/station-e2e
uid=$(id -u)
agents=$HOME/Library/LaunchAgents
app=$HOME/Applications/Spwn.app
old=$HOME/Applications/Station.app
station=$HOME/.local/bin/station
engine="$app/Contents/MacOS/Spwn Engine"
nested="$app/Contents/Library/LoginItems/Station.app/Contents/MacOS/StationMenuBar"
say() { echo "   $*"; }
miss() { echo "miss $*"; }
setup() {
  echo "setup $*"
  exit 1
}
job_pid() { launchctl print "gui/$uid/$1" 2> /dev/null | awk '$1 == "pid" { print $3; exit }'; }
job_runs() { [ -n "$(job_pid "$1")" ]; }
# value: a process is told by the file it runs, as lsof reads it: launchd names a login
# item's and a BundleProgram's processes by their label and file name, never their path.
exe_of() { lsof -a -p "$1" -d txt -Fn 2> /dev/null | sed -n 's/^n//p' | head -1; }
running() { lsof -d txt -Fn 2> /dev/null | grep -qxF "n$1"; }
within() {
  local seconds=$1 n=0
  shift
  until "$@"; do
    n=$((n + 1))
    [ $n -lt "$seconds" ] || return 1
    sleep 1
  done
}

# value: the kind is pinned in the user's launchd, so the window `open` starts, and the
# engine it starts in turn, play a Mac like every call of the tier.
sudo -n launchctl asuser "$uid" launchctl setenv STATION_MACHINE_KIND mac || setup "pin the kind in the user's launchd"
mkdir -p "$HOME/.local/bin" "$HOME/Applications" || setup "the install folders"
cp $e2e/station "$station" && chmod +x "$station" || setup "place the binary"
"$station" engine start > /dev/null || setup "engine start on a bare Mac"
for job in engine agent; do
  test -e "$agents/com.jterrazz.station.$job.plist" || setup "mode B wrote no com.jterrazz.station.$job.plist"
done
within 30 job_runs com.jterrazz.station.engine || setup "launchd keeps no pid for the bare engine"
rm -rf "$old" && ditto $e2e/Station.app "$old" && open "$old" || setup "open the standalone menu bar"
within 30 running "$old/Contents/MacOS/StationMenuBar" || setup "the standalone menu bar never ran"
say "mode B: the bare engine pid $(job_pid com.jterrazz.station.engine), the standalone menu bar running"

rm -rf "$app" && ditto $e2e/Spwn.app "$app" && open "$app" || setup "open Spwn.app"
opened=$(date +%s)
within 90 job_runs com.jterrazz.spwn.engine || miss "launchd keeps no pid for gui/$uid/com.jterrazz.spwn.engine 90 s after Spwn.app opened"
pid=$(job_pid com.jterrazz.spwn.engine)
if [ -n "$pid" ]; then
  ran=$(exe_of "$pid")
  [ "$ran" = "$engine" ] || miss "the engine's job runs $ran, not the binary inside Spwn.app"
  say "the bundle's engine runs $(($(date +%s) - opened)) s after Spwn.app opened: pid $pid, $ran"
fi
for job in engine agent; do
  test -e "$agents/com.jterrazz.station.$job.plist" && miss "the bare com.jterrazz.station.$job.plist is still there"
  launchctl print "gui/$uid/com.jterrazz.station.$job" > /dev/null 2>&1 && miss "launchd still holds the bare com.jterrazz.station.$job"
done
link=$(readlink "$station")
[ "$link" = "$engine" ] || miss "~/.local/bin/station links to '$link', not into the bundle"
within 30 test ! -e "$old" || miss "~/Applications/Station.app is still there"
running "$old/Contents/MacOS/StationMenuBar" && miss "the standalone menu bar still runs"
within 30 job_runs com.jterrazz.station.menubar || miss "launchd keeps no pid for the bundle's menu bar"
menubar=$(job_pid com.jterrazz.station.menubar)
[ -z "$menubar" ] || [ "$(exe_of "$menubar")" = "$nested" ] || miss "the menu bar's login item runs $(exe_of "$menubar"), not the one inside Spwn.app"
say "after: plists $(ls "$agents" | grep jterrazz | tr '\n' ' '), ~/.local/bin/station -> $link, ~/Applications: $(ls "$HOME/Applications" | tr '\n' ' ')"

osascript -e 'quit app "Spwn"' > /dev/null 2>&1 || true
[ -z "$menubar" ] || kill "$menubar" > /dev/null 2>&1 || true
for job in engine agent; do
  launchctl bootout "gui/$uid/com.jterrazz.spwn.$job" > /dev/null 2>&1 || true
done
sudo -n launchctl asuser "$uid" launchctl unsetenv STATION_MACHINE_KIND > /dev/null 2>&1 || true
exit 0

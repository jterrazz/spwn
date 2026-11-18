#!/bin/bash
# value: the system's own answer for every checked setting, asked of the tool macOS
# answers it with, printed as "<name> <value>" in the words station reads it in.
set -uo pipefail
onoff() { if "$@"; then echo on; else echo off; fi; }
echo "filevault $(onoff sh -c 'fdesetup status | grep -q "is On"')"
echo "firewall $(onoff sh -c '/usr/libexec/ApplicationFirewall/socketfilterfw --getglobalstate | grep -q enabled')"
echo "sip $(onoff sh -c 'csrutil status | grep -q enabled')"
echo "gatekeeper $(onoff sh -c 'spctl --status 2>&1 | grep -q enabled')"
critical=$(defaults read /Library/Preferences/com.apple.SoftwareUpdate CriticalUpdateInstall 2> /dev/null || echo 1)
echo "automatic-updates $([ "$critical" = 0 ] && echo off || echo on)"
echo "remote-login $(onoff sh -c 'sudo -n systemsetup -getremotelogin | grep -q "Remote Login: On"')"
lock=$(sysadminctl -screenLock status 2>&1)
case "$lock" in
  *"screenLock is off"*) echo "screen-lock never locks" ;;
  *"delay is immediate"*) echo "screen-lock locks when the display turns off" ;;
  *) echo "screen-lock $lock" ;;
esac
if tmutil destinationinfo 2>&1 | grep -q "No destinations configured"; then
  echo "time-machine no backup yet"
else
  echo "time-machine $(tmutil latestbackup 2>&1 | tail -1)"
fi
waiting=$(softwareupdate -l 2>&1 | grep -c '^\* Label:')
# value: softwareupdated rewrites its list on disk a few seconds after the check, one
# entry first; the list station reads is waited on, 30 s at most, to settle.
for _ in $(seq 15); do
  [ "$(plutil -extract RecommendedUpdates raw -o - /Library/Preferences/com.apple.SoftwareUpdate.plist 2> /dev/null)" = "$waiting" ] && break
  sleep 2
done
case "$waiting" in
  0) echo "macos-updates up to date" ;;
  1) echo "macos-updates 1 update waiting" ;;
  *) echo "macos-updates $waiting updates waiting" ;;
esac

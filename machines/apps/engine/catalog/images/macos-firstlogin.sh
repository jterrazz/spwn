#!/bin/bash
# invariant: A fetched macOS guest's lockdown: run as root (sudo -n /bin/bash <dir>/lockdown.sh)
# from the private folder station made and put this script and its own binary in, over the
# image's own account by its published default password, or over the VM user by the key when
# its marker is absent. Its values come on stdin, one per line, never in a command line. It is
# idempotent. Once its binary and its VM user are checked, it first closes what the image's login opens (sshd's
# password login, the auto-login, Screen Sharing, the default password, its keys and sudo rule), so a run cut
# after that block leaves no login of the image's (one cut inside it, before the password changes, leaves
# the default password to a local login and su); it makes the key reachable only once all of it is
# checked, then writes its marker last; its stdout is the guest's new ed25519 host key, printed as
# soon as it is made, nothing else.
set -euo pipefail

IFS= read -r VM_USER
IFS= read -r IMAGE_KEY
IFS= read -r LOGIN_USER
IFS= read -r LOGIN_PASSWORD
IFS= read -r BIN_SHA256
exec </dev/null 3>&1 1>&2

dir=$(/usr/bin/dirname "$0")
trap '/bin/rm -rf "$dir"' EXIT
marker=/var/db/station/locked
/bin/rm -f "$marker"

random() {
    local p
    p=$(/usr/bin/openssl rand -base64 33)
    [ -n "$p" ] || { echo "openssl made no password" >&2; exit 1; }
    printf '%s\n' "$p"
}

home_of() { /usr/bin/dscl . -read "/Users/$1" NFSHomeDirectory | /usr/bin/awk '{print $2}'; }

[ "$(/usr/bin/shasum -a 256 "$dir/station" | /usr/bin/awk '{print $1}')" = "$BIN_SHA256" ] || { echo "the station binary in $dir is not the engine's" >&2; exit 1; }
# outside: macOS reads a user name whatever its case (id Admin is admin), so only a lowercase short name is compared.
vm_user_rule='^[a-z]([a-z0-9_.-]{0,30}[a-z0-9_-])?$'
[[ "$VM_USER" =~ $vm_user_rule ]] && [ "$VM_USER" != "$LOGIN_USER" ] && [ "$VM_USER" != root ] || { echo "the VM user '$VM_USER' is no lowercase short name, or the image's own account or root: a fetched macOS guest is made for another user" >&2; exit 1; }
# outside: macOS's own accounts hold a uid under 500 (daemon's 1, the _ ones), but nobody's 4294967294; an account sysadminctl makes starts at 501.
if vm_uid=$(/usr/bin/id -u "$VM_USER" 2>/dev/null) && { [ "$vm_uid" -lt 500 ] || [ "$vm_uid" -gt 60000 ]; }; then echo "the VM user '$VM_USER' is a system account of the image (uid $vm_uid): a fetched macOS guest is made for another user" >&2; exit 1; fi
# outside: sshd keeps the first value it reads, and macOS's sshd_config includes sshd_config.d/* before its own lines, in name order.
/bin/cat > /etc/ssh/sshd_config.d/000-station.conf <<'CONF'
PasswordAuthentication no
KbdInteractiveAuthentication no
PermitRootLogin no
AuthorizedKeysFile .ssh/authorized_keys
AllowTcpForwarding local
ClientAliveInterval 15
ClientAliveCountMax 4
CONF
# outside: the image logs LOGIN_USER in at boot by the password /etc/kcpassword holds.
/usr/bin/defaults delete /Library/Preferences/com.apple.loginwindow autoLoginUser 2>/dev/null || true
/bin/rm -f /etc/kcpassword
# outside: the image serves Screen Sharing (launchd listens on 5900), which takes LOGIN_USER's password, and a VNC password once VNCSettings.txt holds one.
/bin/launchctl disable system/com.apple.screensharing
/bin/launchctl bootout system/com.apple.screensharing 2>/dev/null || true
/bin/rm -f /Library/Preferences/com.apple.VNCSettings.txt
# outside: macOS changes the password of an account holding a secure token only by its old one, read from a terminal or the command line.
if /usr/bin/dscl . -authonly "$LOGIN_USER" "$LOGIN_PASSWORD" 2>/dev/null; then
    p=$(random)
    /usr/bin/dscl . -passwd "/Users/$LOGIN_USER" "$LOGIN_PASSWORD" "$p"
    unset p
fi
login_home=$(home_of "$LOGIN_USER")
if [ -d "$login_home/.ssh" ]; then
    : > "$login_home/.ssh/authorized_keys"
    /bin/rm -f "$login_home/.ssh/authorized_keys2"
fi
/bin/rm -f /var/root/.ssh/authorized_keys /var/root/.ssh/authorized_keys2
/bin/rm -f "/etc/sudoers.d/$LOGIN_USER-nopasswd"


/bin/mkdir -p /usr/local/bin
/usr/sbin/chown root:wheel /usr/local/bin
/bin/chmod 0755 /usr/local/bin
/usr/bin/install -o root -g wheel -m 0755 "$dir/station" /usr/local/bin/station

if ! /usr/bin/id "$VM_USER" >/dev/null 2>&1; then
    p=$(random)
    /usr/sbin/sysadminctl -addUser "$VM_USER" -fullName "$VM_USER" -shell /bin/zsh -password "$p" -admin
fi
p=$(random)
printf '%s\n' "$p" | /usr/bin/dscl . -passwd "/Users/$VM_USER" >/dev/null
unset p
/usr/sbin/createhomedir -c -u "$VM_USER" >/dev/null
home=$(home_of "$VM_USER")
if /usr/bin/dscl . -read /Groups/com.apple.access_ssh >/dev/null 2>&1; then
    /usr/sbin/dseditgroup -o edit -a "$VM_USER" -t user com.apple.access_ssh
fi
/usr/bin/install -d -m 0700 -o "$VM_USER" -g staff "$home/.ssh"
: > "$home/.ssh/authorized_keys"
/usr/sbin/chown "$VM_USER:staff" "$home/.ssh/authorized_keys"
/bin/chmod 0600 "$home/.ssh/authorized_keys"
/bin/rm -f "$home/.ssh/authorized_keys2"
/usr/bin/install -d -m 0700 -o "$VM_USER" -g staff "$home/.config" "$home/.config/station" "$home/.config/station/machinefiles"

# outside: sudo skips a file of /etc/sudoers.d whose name holds a dot.
rule=$(/usr/bin/mktemp /etc/sudoers.d/.60-station-user.XXXXXXXX)
echo "$VM_USER ALL=(ALL) NOPASSWD:ALL" > "$rule"
/usr/sbin/chown root:wheel "$rule"
/bin/chmod 0440 "$rule"
/usr/sbin/visudo -cf "$rule" >/dev/null || { /bin/rm -f "$rule"; echo "visudo refuses the VM user's sudo rule" >&2; exit 1; }
/bin/mv -f "$rule" /etc/sudoers.d/60-station-user
/usr/bin/sudo -u "$VM_USER" /usr/bin/sudo -k -n true || { echo "sudo asks $VM_USER a password: /etc/sudoers.d/60-station-user is not read" >&2; exit 1; }

# outside: the image's Homebrew belongs to LOGIN_USER, and brew runs as the user who owns its prefix.
if [ -d /opt/homebrew ] && [ "$(/usr/bin/stat -f %Su /opt/homebrew)" = "$LOGIN_USER" ]; then
    /usr/sbin/chown -R "$VM_USER" /opt/homebrew
fi

/bin/rm -f /etc/ssh/ssh_host_*
/usr/bin/ssh-keygen -A
[ -s /etc/ssh/ssh_host_ed25519_key.pub ] || { echo "ssh-keygen made no ed25519 host key" >&2; exit 1; }
/bin/cat /etc/ssh/ssh_host_ed25519_key.pub >&3

if /usr/bin/dscl . -authonly "$LOGIN_USER" "$LOGIN_PASSWORD" 2>/dev/null; then
    echo "$LOGIN_USER still logs in by the image's default password" >&2
    exit 1
fi
[ ! -e /etc/kcpassword ] || { echo "/etc/kcpassword is still there" >&2; exit 1; }
! /bin/launchctl print system/com.apple.screensharing >/dev/null 2>&1 || { echo "Screen Sharing still runs" >&2; exit 1; }
[ ! -e "/etc/sudoers.d/$LOGIN_USER-nopasswd" ] || { echo "$LOGIN_USER still has a passwordless sudo" >&2; exit 1; }
sshd=$(/usr/sbin/sshd -T)
for want in 'passwordauthentication no' 'kbdinteractiveauthentication no' 'permitrootlogin no' 'authorizedkeysfile .ssh/authorized_keys'; do
    printf '%s\n' "$sshd" | /usr/bin/grep -qx "$want" || { echo "sshd does not read '$want': /etc/ssh/sshd_config.d/000-station.conf is not read first" >&2; exit 1; }
done
/usr/bin/sudo -u "$VM_USER" /usr/bin/sudo -k -n true || { echo "sudo asks $VM_USER a password" >&2; exit 1; }

printf '%s\n' "$IMAGE_KEY" > "$home/.ssh/authorized_keys"
/usr/bin/install -d -m 0755 /var/db/station
: > "$marker"

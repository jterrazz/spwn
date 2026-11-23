#!/bin/bash
# invariant: A fetched macOS guest's first login: run as root (sudo -n /bin/bash -s) over
# the image's own account, LOGIN_USER by its published default password, once, before
# station's key is in the guest. Station sets VM_USER, IMAGE_KEY, LOGIN_USER and
# LOGIN_PASSWORD above this script and has copied its own binary to /tmp/station. It ends
# with no password of the image's that logs in: password login is off, and the image
# account's password is one made here and never kept.
set -euo pipefail

unknown() { /usr/bin/openssl rand -base64 33; }

if ! /usr/bin/id "$VM_USER" >/dev/null 2>&1; then
    /usr/sbin/sysadminctl -addUser "$VM_USER" -fullName "$VM_USER" -shell /bin/zsh -password "$(unknown)" -admin
fi
/usr/sbin/createhomedir -c -u "$VM_USER" >/dev/null
home=$(/usr/bin/dscl . -read "/Users/$VM_USER" NFSHomeDirectory | /usr/bin/awk '{print $2}')
if /usr/bin/dscl . -read /Groups/com.apple.access_ssh >/dev/null 2>&1; then
    /usr/sbin/dseditgroup -o edit -a "$VM_USER" -t user com.apple.access_ssh
fi

echo "$VM_USER ALL=(ALL) NOPASSWD:ALL" > /etc/sudoers.d/60-station-user
/bin/chmod 0440 /etc/sudoers.d/60-station-user
/usr/sbin/visudo -cf /etc/sudoers.d/60-station-user
/usr/bin/sudo -u "$VM_USER" /usr/bin/sudo -k -n true || { echo "sudo asks $VM_USER a password: /etc/sudoers.d/60-station-user is not read" >&2; exit 1; }

/usr/bin/install -d -m 0700 -o "$VM_USER" -g staff "$home/.ssh"
printf '%s\n' "$IMAGE_KEY" > "$home/.ssh/authorized_keys"
/usr/sbin/chown "$VM_USER:staff" "$home/.ssh/authorized_keys"
/bin/chmod 0600 "$home/.ssh/authorized_keys"
/usr/bin/install -d -m 0700 -o "$VM_USER" -g staff "$home/.config" "$home/.config/station" "$home/.config/station/machinefiles"

/usr/bin/install -d -m 0755 /usr/local/bin
/usr/bin/install -m 0755 /tmp/station /usr/local/bin/station
/bin/rm -f /tmp/station

# invariant: the image password goes before sshd is locked, so a failure past it leaves no password known.
# outside: the image logs LOGIN_USER in at boot by the password /etc/kcpassword holds.
/usr/bin/defaults delete /Library/Preferences/com.apple.loginwindow autoLoginUser 2>/dev/null || true
/bin/rm -f /etc/kcpassword
/usr/bin/dscl . -passwd "/Users/$LOGIN_USER" "$LOGIN_PASSWORD" "$(unknown)"
if /usr/bin/dscl . -authonly "$LOGIN_USER" "$LOGIN_PASSWORD" 2>/dev/null; then
    echo "$LOGIN_USER still logs in by the image's default password" >&2
    exit 1
fi

# outside: sshd keeps the first value it reads, and macOS's sshd_config includes sshd_config.d/* before its own lines, in name order.
/bin/cat > /etc/ssh/sshd_config.d/000-station.conf <<'CONF'
PasswordAuthentication no
KbdInteractiveAuthentication no
AllowTcpForwarding local
ClientAliveInterval 15
ClientAliveCountMax 4
CONF
/usr/sbin/sshd -T | /usr/bin/grep -qx 'passwordauthentication no' || { echo "sshd still takes passwords: /etc/ssh/sshd_config.d/000-station.conf is not read first" >&2; exit 1; }
/usr/sbin/sshd -T | /usr/bin/grep -qx 'kbdinteractiveauthentication no' || { echo "sshd still takes keyboard-interactive logins" >&2; exit 1; }

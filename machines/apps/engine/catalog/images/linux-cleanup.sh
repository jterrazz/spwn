#!/bin/bash
# invariant: The second build session: run as root (sudo bash -s) as VM_USER with the image
# key, once the base's admin session has ended, so its user can be removed.
set -euo pipefail

# outside: the base runs tart-guest-agent as admin, and its process keeps userdel from removing the user.
systemctl disable --now tart-guest-agent.service 2>/dev/null || true
loginctl terminate-user admin 2>/dev/null || true
pkill -KILL -u admin || true
for _ in $(seq 1 30); do
    pgrep -u admin >/dev/null || break
    sleep 1
done
userdel --remove admin
sed -i '/^admin /d' /etc/sudoers.d/90-cloud-init-users 2>/dev/null || true
truncate -s 0 /etc/machine-id
rm -f /etc/ssh/ssh_host_*
apt-get clean
rm -rf /var/lib/apt/lists/*
sync
systemctl --no-block poweroff

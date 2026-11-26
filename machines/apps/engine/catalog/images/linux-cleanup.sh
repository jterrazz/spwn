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

export DEBIAN_FRONTEND=noninteractive
added="$(getent passwd "$SUDO_USER" | cut -d: -f6)/.local/share/station/apt"
purge=()
# outside: the upstream ships two kernels, each with its headers, tools and metapackages, and the firmware of physical devices; a VM boots one kernel on virtual devices. When the image phase installed a kernel (the Machinefile pins one), every other kernel and the firmware go; what that phase installed stays.
if ls "$added" 2>/dev/null | grep -q '^linux-image-'; then
    kernel=$(linux-version list | linux-version sort | tail -1)
    for p in $(dpkg-query -W -f '${db:Status-Abbrev} ${Package}\n' 'linux-*' 'firmware-*' 2>/dev/null | awk '$1 == "ii" {print $2}'); do
        case $p in
            linux-libc-dev | linux-base | *"$kernel"*) ;;
            *) [ -e "$added/$p" ] || purge+=("$p") ;;
        esac
    done
fi
# outside: snapd and its seeds serve no app of the catalog; the eBPF tools and Vulkan's Mesa drivers come with the upstream's server set and nothing a VM runs asks them (xvfb takes Mesa's GL, not its Vulkan).
for p in snapd bpftrace bpfcc-tools python3-bpfcc libbpfcc mesa-vulkan-drivers; do
    if dpkg-query -W -f '${db:Status-Abbrev}' "$p" 2>/dev/null | grep -q '^ii' && [ ! -e "$added/$p" ]; then purge+=("$p"); fi
done
if [ ${#purge[@]} -gt 0 ]; then
    apt-get purge -y "${purge[@]}"
    apt-get autoremove --purge -y
fi
dpkg-query -W snapd >/dev/null 2>&1 || rm -rf /var/lib/snapd /var/cache/snapd /snap
! command -v update-initramfs >/dev/null || update-initramfs -u -k all
apt-get clean
rm -rf /var/lib/apt/lists/*
sync
systemctl --no-block poweroff

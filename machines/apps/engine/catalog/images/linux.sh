#!/bin/bash
# invariant: The linux image's base plumbing: run as root (sudo bash -s) over the build
# VM's NAT session, as the base's admin user. Its inputs are the environment variables
# station sets: VM_USER, IMAGE_KEY (the host's image public key) and the station
# binary already copied to /tmp/station. What the image carries is its Machinefile's
# image phase, which the build runs after it.
set -euo pipefail

if [ "$VM_USER" = admin ]; then echo "the VM user 'admin' is the base's own account: an image is made for another user" >&2; exit 1; fi
# outside: Debian and Ubuntu keep the uids under 1000 and above 60000 (nobody's 65534) to the system's own accounts; the user an image makes gets one from 1000.
if vm_uid=$(id -u "$VM_USER" 2>/dev/null) && { [ "$vm_uid" -lt 1000 ] || [ "$vm_uid" -gt 60000 ]; }; then echo "the VM user '$VM_USER' is a system account of the image (uid $vm_uid): an image is made for another user" >&2; exit 1; fi
# outside: the base ships admin, the build's own login, which linux-cleanup.sh removes; another name of it is admin, by its uid.
if [ -n "${vm_uid:-}" ] && [ "$vm_uid" = "$(id -u admin 2>/dev/null)" ]; then echo "the VM user '$VM_USER' is the base's own account admin by its uid ($vm_uid): an image is made for another user" >&2; exit 1; fi

export DEBIAN_FRONTEND=noninteractive

id "$VM_USER" >/dev/null 2>&1 || useradd --create-home --shell /bin/zsh "$VM_USER"
# hazard: sudo's includedir skips a file whose name holds a dot, and a user's name may hold one: the rule lives in a file of a fixed name.
echo "$VM_USER ALL=(ALL) NOPASSWD:ALL" > /etc/sudoers.d/60-station-user
chmod 0440 /etc/sudoers.d/60-station-user
visudo -cf /etc/sudoers.d/60-station-user
sudo -u "$VM_USER" sudo -k -n true || { echo "sudo asks $VM_USER a password: /etc/sudoers.d/60-station-user is not read" >&2; exit 1; }
install -d -m 0700 -o "$VM_USER" -g "$VM_USER" "/home/$VM_USER/.ssh"
printf '%s\n' "$IMAGE_KEY" > "/home/$VM_USER/.ssh/authorized_keys"
chown "$VM_USER:$VM_USER" "/home/$VM_USER/.ssh/authorized_keys"
chmod 0600 "/home/$VM_USER/.ssh/authorized_keys"
install -d -m 0700 -o "$VM_USER" -g "$VM_USER" "/home/$VM_USER/.config" "/home/$VM_USER/.config/station" "/home/$VM_USER/.config/station/machinefiles"

apt-get update
# outside: zsh is the user's login shell, growpart (cloud-guest-utils) grows the disk at each boot, and node's linux builds link libatomic, which the base lacks.
apt-get install -y zsh curl ca-certificates cloud-guest-utils libatomic1
# outside: the docker group exists before the docker package installs, so the user is in it from the image's first session.
groupadd -f docker
usermod -aG docker "$VM_USER"

cat > /etc/ssh/sshd_config.d/60-station.conf <<'CONF'
AllowTcpForwarding local
ClientAliveInterval 15
ClientAliveCountMax 4
CONF

install -m 0755 /tmp/station /usr/local/bin/station
sudo -u "$VM_USER" -H /usr/local/bin/station init

cat > /etc/systemd/system/station-hostkeys.service <<'UNIT'
[Unit]
Description=Generate the ssh host keys a clone was built without
DefaultDependencies=no
After=local-fs.target
Before=ssh.service ssh.socket
ConditionPathExists=!/etc/ssh/ssh_host_ed25519_key

[Service]
Type=oneshot
ExecStart=/usr/bin/ssh-keygen -A

[Install]
WantedBy=multi-user.target
UNIT

cat > /etc/systemd/system/station-intent.service <<'UNIT'
[Unit]
Description=Tell the host this stop is a poweroff, never a reboot
DefaultDependencies=no
Before=shutdown.target

[Service]
Type=oneshot
ExecStart=/bin/sh -c 'printf "\\nstation-intent=poweroff\\n" >/dev/hvc0'

[Install]
WantedBy=poweroff.target halt.target
UNIT

cat > /etc/systemd/system/station-growroot.service <<'UNIT'
[Unit]
Description=Grow the root partition and its filesystem to the disk the VM was given
After=local-fs.target

[Service]
Type=oneshot
ExecStart=-/usr/bin/growpart /dev/vda 1
ExecStart=-/usr/sbin/resize2fs /dev/vda1

[Install]
WantedBy=multi-user.target
UNIT

systemctl enable station-hostkeys.service station-intent.service station-growroot.service

# outside: a VM is a clone of a disk baked behind its Mac's NAT, never a cloud instance; cloud-init would only render again, at every boot, the network it wrote at the build's (every `en*` link by DHCP, whatever its MAC), which a clone keeps without it.
[ ! -d /etc/cloud ] || touch /etc/cloud/cloud-init.disabled
# outside: the engine waits on sshd, which answers once the address exists; waiting for every link online held docker and the boot's end, never ssh.
systemctl mask systemd-networkd-wait-online.service
# outside: each serves what a VM has not: ModemManager a modem, multipathd a SAN's paths, udisks2 a desktop's disk mounts, fwupd a device's firmware, motd-news Canonical's news for a login banner.
systemctl mask ModemManager.service multipathd.service multipathd.socket udisks2.service fwupd.service fwupd-refresh.timer motd-news.timer

# outside: a guest kernel line reaches the host's console file only when hvc0 is the kernel's console.
install -d /etc/default/grub.d
cat > /etc/default/grub.d/60-station-console.cfg <<'CFG'
GRUB_CMDLINE_LINUX_DEFAULT="$GRUB_CMDLINE_LINUX_DEFAULT console=hvc0"
CFG
update-grub
grep -q 'console=hvc0' /boot/grub/grub.cfg || { echo "the kernel's console is not hvc0" >&2; exit 1; }
# outside: the console is a host file tart reads back as input, so a getty on it answers its own prompts.
systemctl mask --now serial-getty@hvc0.service
# outside: every Mac gives a VM the same virtual devices, so an initramfs needs only the modules this one's root uses; the cleanup builds it again for the kernel the image keeps.
[ ! -d /etc/initramfs-tools/conf.d ] || echo 'MODULES=dep' > /etc/initramfs-tools/conf.d/60-station-modules.conf

#!/bin/bash
# invariant: The linux image recipe: run as root (sudo bash -s) over the build VM's NAT
# session, as the base's admin user. Its inputs are the environment variables
# station sets: VM_USER, IMAGE_KEY (the host's image public key) and the station
# binary already copied to /tmp/station.
set -euo pipefail

export DEBIAN_FRONTEND=noninteractive

id "$VM_USER" >/dev/null 2>&1 || useradd --create-home --shell /bin/zsh "$VM_USER"
echo "$VM_USER ALL=(ALL) NOPASSWD:ALL" > "/etc/sudoers.d/$VM_USER"
chmod 0440 "/etc/sudoers.d/$VM_USER"
install -d -m 0700 -o "$VM_USER" -g "$VM_USER" "/home/$VM_USER/.ssh"
printf '%s\n' "$IMAGE_KEY" > "/home/$VM_USER/.ssh/authorized_keys"
chown "$VM_USER:$VM_USER" "/home/$VM_USER/.ssh/authorized_keys"
chmod 0600 "/home/$VM_USER/.ssh/authorized_keys"

apt-get update
apt-get install -y libatomic1 git zsh curl ca-certificates build-essential gnupg cloud-guest-utils
# outside: the system libraries and fonts of `playwright install-deps chromium` on Ubuntu 24.04, so a headless browser gate runs in a fresh workspace.
apt-get install -y libasound2t64 libatk-bridge2.0-0t64 libatk1.0-0t64 libatspi2.0-0t64 libcairo2 libcups2t64 libdbus-1-3 libdrm2 libgbm1 libglib2.0-0t64 libnspr4 libnss3 libpango-1.0-0 libx11-6 libxcb1 libxcomposite1 libxdamage1 libxext6 libxfixes3 libxkbcommon0 libxrandr2 xvfb fonts-noto-color-emoji fonts-unifont libfontconfig1 libfreetype6 xfonts-cyrillic xfonts-scalable fonts-liberation fonts-ipafont-gothic fonts-wqy-zenhei fonts-tlwg-loma-otf fonts-freefont-ttf

curl -fsSL https://get.docker.com | sh
usermod -aG docker "$VM_USER"
sudo -u "$VM_USER" bash -c 'curl -fsSL https://mise.run | sh'

cat > /etc/ssh/sshd_config.d/60-station.conf <<'CONF'
AllowTcpForwarding local
ClientAliveInterval 15
ClientAliveCountMax 4
CONF

install -m 0755 /tmp/station /usr/local/bin/station
sudo -u "$VM_USER" -H /usr/local/bin/station setup

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

systemctl enable station-hostkeys.service station-intent.service

# outside: a guest kernel line reaches the host's console file only when hvc0 is the kernel's console.
install -d /etc/default/grub.d
cat > /etc/default/grub.d/60-station-console.cfg <<'CFG'
GRUB_CMDLINE_LINUX_DEFAULT="$GRUB_CMDLINE_LINUX_DEFAULT console=hvc0"
CFG
update-grub
grep -q 'console=hvc0' /boot/grub/grub.cfg || { echo "the kernel's console is not hvc0" >&2; exit 1; }
# outside: the console is a host file tart reads back as input, so a getty on it answers its own prompts.
systemctl mask --now serial-getty@hvc0.service

package pages

// value: the lines a running VM's console shows, the oldest first: the
// boot of the station image, then what the VM runs for. A VM without an
// entry has written none.
func consoleOf(vm string) []string {
	return map[string][]string{
		"infra": {
			"[    0.000000] Linux version 6.8.0-52-generic (buildd@lcy02) #53-Ubuntu SMP PREEMPT_DYNAMIC",
			"[    1.412903] virtio_net virtio1 enp0s1: renamed from eth0",
			"[    2.108344] EXT4-fs (vda1): mounted filesystem with ordered data mode",
			"[    3.221087] systemd[1]: Reached target Multi-User System.",
			"[    3.480112] systemd[1]: Started OpenBSD Secure Shell server.",
			"[    3.902667] systemd[1]: Started Tailscale node agent.",
			"[    4.337941] systemd[1]: Started station guest.",
			"[    5.016528] systemd[1]: Started nginx - A high performance web server.",
			"[    5.774190] systemd[1]: Started cloudflared - Cloudflare Tunnel.",
			"[    6.210455] cloudflared[612]: Registered tunnel connection connIndex=0 location=cdg04",
			"[    6.431879] k3s[744]: Running kube-apiserver --secure-port=6443",
			"[    9.883012] nginx[598]: GET /healthz 200 0.036s",
		},
		"linux-1": {
			"[    0.000000] Linux version 6.8.0-52-generic (buildd@lcy02) #53-Ubuntu SMP PREEMPT_DYNAMIC",
			"[    1.388210] virtio_net virtio1 enp0s1: renamed from eth0",
			"[    2.090731] EXT4-fs (vda1): mounted filesystem with ordered data mode",
			"[    3.150962] systemd[1]: Reached target Multi-User System.",
			"[    3.402518] systemd[1]: Started OpenBSD Secure Shell server.",
			"[    3.871204] systemd[1]: Started station guest.",
			"[    4.655033] station-guest[411]: listening for its Mac over the one way in",
			"[    5.120847] station-guest[411]: credentials loaned by macbook-pro: github",
			"[    7.304291] sshd[702]: Accepted publickey for admin from 192.168.64.1 port 52211",
			"[   12.018664] node[1190]: vite v6.0.7 ready in 412 ms, listening on 5173",
			"[   12.540318] node[1203]: astro v5.1.2 dev server listening on 4321",
			"[   31.774052] node[1190]: hmr update /src/pages/index.astro",
		},
	}[vm]
}

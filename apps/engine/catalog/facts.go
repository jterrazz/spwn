package catalog

type Fact string // value: a fact of the machine an Inspector reads, never writes: what a Checked setting and the Overview look at.

const (
	FactFileVault  Fact = "filevault"
	FactFirewall   Fact = "firewall"
	FactSIP        Fact = "sip"
	FactGatekeeper Fact = "gatekeeper"
	// value: macOS installs its security responses by itself: on or off.
	FactAutoUpdates Fact = "automatic-updates"
	// value: the seconds the screen locks after the display turns off, or
	// off; the display's own idle minutes, 0 for never.
	FactScreenLock   Fact = "screen-lock"
	FactDisplaySleep Fact = "display-sleep"
	FactRemoteLogin  Fact = "remote-login"
	// value: when Time Machine last finished a backup (RFC 3339), or never.
	FactTimeMachine Fact = "time-machine"
	// value: whether FileVault's planned restart, which unlocks the disk
	// once, is supported.
	FactAuthRestart Fact = "authrestart"
	// value: whether the operating system has updates waiting.
	FactSystemUpdates Fact = "system-updates"
	// value: the kernel's name and release ("Darwin 25.0.0").
	FactKernel  Fact = "kernel"
	FactLocalIP Fact = "local-ip"
	// value: the resolvers the machine asks, at most two, the local stub left out.
	FactDNS Fact = "dns"
	// value: the owner's global git identity and signing, the keys they sign
	// and push with, and who GitHub's CLI is signed in as.
	FactGitEmail   Fact = "git-email"
	FactGitName    Fact = "git-name"
	FactGitSigning Fact = "git-signing"
	FactGPGKey     Fact = "gpg-key"
	FactGitHub     Fact = "github"
	FactSSHKey     Fact = "ssh-key"
)

package catalog

import (
	"path/filepath"
	"strings"
)

type Level string

const (
	LevelCache    Level = "cache"
	LevelData     Level = "data"
	LevelSettings Level = "settings"
	LevelKept     Level = "kept"
)

var Levels = []Level{LevelCache, LevelData, LevelSettings, LevelKept}

func (Level) Enum() []string {
	out := make([]string, 0, len(Levels))
	for _, l := range Levels {
		out = append(out, string(l))
	}
	return out
}

type Stored struct {
	Path  string `yaml:"path"`
	Level Level  `yaml:"level"`
	// value: the machines the row is for, as the file selects them; On is
	// what the catalog's load makes of it, every machine when empty.
	Placed Selector      `yaml:",inline" json:"-"`
	On     []MachineKind `yaml:"-"`
	Note   string        `yaml:"note,omitempty"`
	Memos  []string      `yaml:"memos,omitempty"`
}

type Action struct {
	Run    []string `yaml:"run,omitempty"`
	Env    []string `yaml:"env,omitempty"`
	Delete []string `yaml:"delete,omitempty"`
	Prune  []string `yaml:"prune,omitempty"`
	// value: the command the action runs with, which must be on the
	// search path for it to run.
	Uses   string        `yaml:"uses,omitempty"`
	Exists string        `yaml:"exists,omitempty"`
	Placed Selector      `yaml:",inline" json:"-"`
	On     []MachineKind `yaml:"-"`
}

type AppFile string

const (
	FileGoEnv  AppFile = "go-env"
	FileNpmrc  AppFile = "npmrc"
	FileBunfig AppFile = "bunfig"
	FileMise   AppFile = "mise settings"
)

func (AppFile) Enum() []string {
	return []string{string(FileGoEnv), string(FileNpmrc), string(FileBunfig), string(FileMise)}
}

func (f AppFile) Tabled() bool {
	// invariant: a TOML file keeps its keys in tables; the others keep one
	// key per line.
	return f == FileBunfig || f == FileMise
}

type Relocation struct {
	File    AppFile `yaml:"file"`
	Key     string  `yaml:"key"`
	Value   string  `yaml:"value"`
	Pending string  `yaml:"pending,omitempty"`
}

type Move struct {
	From string `yaml:"from"`
	To   string `yaml:"to"`
}

type Misplaced struct {
	Path  string   `yaml:"path"`
	Busy  string   `yaml:"busy,omitempty"`
	Move  []Move   `yaml:"move,omitempty"`
	Prune []string `yaml:"prune,omitempty"`
	Then  []string `yaml:"then,omitempty"`
	Free  []Action `yaml:"free,omitempty"`
	// value: the folders under Path whose entries are the owner's own, a VM
	// each: while one holds any entry, Path moves and is never freed.
	Yours []string `yaml:"yours,omitempty"`
}

type SizeMethod string

const (
	SizeDirs       SizeMethod = ""
	SizeContainers SizeMethod = "docker-system-df"
)

func (SizeMethod) Enum() []string { return []string{string(SizeContainers)} }

type Disk struct {
	Paths     []Stored          `yaml:"paths,omitempty"`
	Installed []string          `yaml:"installed,omitempty"`
	Removal   map[string]string `yaml:"removal,omitempty"`
	Size      SizeMethod        `yaml:"size,omitempty"`
	Free      []Action          `yaml:"free,omitempty"`
	Erase     []Action          `yaml:"erase,omitempty"`
	NoErase   string            `yaml:"no-erase,omitempty"`
	Quit      string            `yaml:"quit,omitempty"`
	Relocate  []Relocation      `yaml:"relocate,omitempty"`
	Misplaced []Misplaced       `yaml:"misplaced,omitempty"`
}

func (s Disk) Declared() bool {
	return len(s.Paths) > 0 || len(s.Installed) > 0 || len(s.Free) > 0 || len(s.Misplaced) > 0 || s.Size != SizeDirs
}

type Exception struct{ Path, Reason string }

func (s Stored) PlacedOn(k MachineKind) bool { return placedOn(s.On, k) }

func (a Action) PlacedOn(k MachineKind) bool { return placedOn(a.On, k) }

func placedOn(on []MachineKind, k MachineKind) bool {
	if len(on) == 0 {
		return true
	}
	for _, kind := range on {
		if kind == k {
			return true
		}
	}
	return false
}

func cache(path string, on ...MachineKind) Stored {
	return Stored{Path: path, Level: LevelCache, On: on}
}
func data(path string, on ...MachineKind) Stored { return Stored{Path: path, Level: LevelData, On: on} }
func conf(path string, on ...MachineKind) Stored {
	return Stored{Path: path, Level: LevelSettings, On: on}
}
func kept(path string, on ...MachineKind) Stored { return Stored{Path: path, Level: LevelKept, On: on} }

func noted(s Stored, note string) Stored { s.Note = note; return s }

func memos(s Stored, patterns ...string) Stored { s.Memos = patterns; return s }

func remove(patterns ...string) Action { return Action{Delete: patterns} }

const (
	stationNoErase = "Machines' data are VMs and images: station machines delete <vm>, station images delete <image>"
	appsNoErase    = "nothing to erase: keys and logins are kept"
)

var StationDisk = Disk{
	Paths: []Stored{
		memos(noted(cache("{cache}/station"), "downloaded station binaries (`station-linux-arm64`) and the last-known sizes a read shows until its walk answers"), "sizes.json"),
		noted(cache("{cache}/station/release.json"), "the latest release as GitHub last gave it, and the last ask it refused: `disk free` removes it, and the next ask writes it again"),
		noted(cache(StationLogs), "`engine.log`, each VM's run `<vm>.log`, `station-build-<image>.log`, each app's install `<app>.log`"),
		noted(cache("{state}/station/images/askpass"), "0700, the base image's admin password an earlier build left: no build writes it now, `disk free` removes it"),
		noted(cache("{state}/station/dns/quad9-dns.mobileconfig"), "regenerated by the dns setting"),
		noted(cache("{cache}/station/sources/*"), "the shared source's clone, at its default branch: cloned again when it is gone"),
		noted(data(StationVMs), "every VM and image in Machines' home, whoever made it: counted in Machines' kinds (machine disks, images, other VMs, leftovers), never a Disk row; `station machines delete`, `station images delete`"),
		noted(data("{state}/station/images/*/*.json"), "image records, paired with their images"),
		noted(data("{state}/station/builds/*.json"), "each running build's step and its process, `<image>.json`: cleared when the build ends"),
		noted(data("{state}/station/known_hosts/*"), "pins: deleting them downgrades to trust-on-first-use, never cleaned"),
		noted(data("{state}/station/consoles/*"), "live while a VM or a build runs: `consoles/<vm>`, `consoles/station-build-<image>`"),
		noted(data("{state}/station/engine.sock"), "the engine's socket, live while it runs"),
		noted(data("{state}/station/agent.sock", Mac), "the key agent's socket, served by the engine while it runs: the IdentityAgent of every machine's block on this Mac"),
		noted(data("{state}/station/device-key.pub", Mac), "this Mac's device key, its public half, written at each engine start: the private half never leaves the Secure Enclave"),
		noted(data("{state}/station/device-key.enclave", Mac), "this Mac's device key as its Secure Enclave wrapped it, mode 0600, made at the engine's first start: only this Mac's Enclave can use it, it signs only for you at the unlocked Mac, and deleting it loses the key: the next start makes another"),
		noted(data("{state}/station/device-key-unattended.enclave", Mac), "this Mac's device key instead of `device-key.enclave` while Never sleeps is applied, mode 0600: it signs with nobody at the Mac from its first unlock on, so a Mac run alone reaches its VMs with its screen locked; each engine start keeps the one file of the two that matches the setting and deletes the other"),
		noted(data("{state}/station/engine.json"), "the binary the engine last started from and its version, rewritten at each engine start: the window finds station by it"),
		noted(data("{state}/station/installed.json"), "the binary the public installer last installed and its version, written by the installer at the end of an install that installed a station binary, never deleted: the window finds station by it before any engine has started"),
		noted(data("{state}/station/children/*"), "the engine's `tailscale` calls in flight, `<pid>` holding its start: the next engine start kills what a killed one left"),
		noted(data("{state}/station/runs/*"), "each running VM's `<vm>.pid` and `<vm>.exit`"),
		noted(data("{state}/station/applied/*.json"), "one record for each setting Machines applied: when, and a fingerprint of what it wrote, so a setting changed since reads as such; deleting them leaves each setting read from what is there"),
		noted(data("{state}/station/engine-stopped.json"), "when and by whom the owner last stopped the engine, shown while it is stopped: deleting it only forgets who stopped it"),
		noted(data("{state}/station/acts.json"), "the long acts the engine guides (a create), each one's rail, step and end, kept 10 minutes after it ends: deleting it forgets them"),
		noted(data("{state}/station/uptime.json"), "each server's last 30 days, written by its Mac's engine: deleting it starts every record again"),
		noted(data("{state}/station/journal.jsonl", Mac), "what was done on this Mac and its VMs, the last 30 days: deleting it forgets that history"),
		noted(data("{state}/station/journal.jsonl.lock", Mac), "empty: the lock every write of the journal takes"),
		noted(data("{state}/station/vault.json", Mac), "the vault's book: each password's label, program, kind and grants, never a value (the values are Machines' Keychain items)"),
		noted(data("{state}/station/vault.json.lock", Mac), "empty: the lock every change of the vault's book takes"),
		noted(data("{state}/station/devices.json", Mac), "the device book, mode 0600: the other devices this Mac lets its VMs trust, each sealed by its device key, its generation and digest anchored in the login keychain's `station-devices` item; deleting it, or writing back an older copy, lets none in until each is added again"),
		noted(data("{state}/station/devices.json.lock", Mac), "empty: the lock every change of the device book takes"),
		noted(data("{state}/station/vms/.records.lock"), "empty: the lock a create and the engine take in turn to write a VM record"),
		noted(data("{state}/station/use/*"), "each terminal's machine (`station use`), forgotten after 7 days unused"),
		noted(conf("~/Library/LaunchAgents/com.jterrazz.station.engine.plist", Mac), "the one engine's launchd agent, on a Mac without Spwn.app: with the app, its job is the bundle's, and moving to it deletes this file"),
		noted(data("~/.local/state/station/"+JobStartFile, Mac), "on a Mac with Spwn.app, what its engine's job starts with, which its definition in the sealed bundle cannot hold: the environment `station engine start` ran in and the job's log; written by `station engine start`, always under the home, since a job starts with no XDG variable"),
		noted(data("~/.local/state/station/"+MenuBarOfferedFile, Mac), "empty, on a Mac with Spwn.app: the bundle's first start showed its menu bar at every login, so no later start shows it again; written by `station engine start`, and deleting it shows the menu bar at the next start"),
		noted(conf("{config}/station/preferences.json"), "here, hosts, vm_user, the ports policy"),
		noted(conf("{config}/station/machinefiles/*.lock"), "each Machinefile's lock, written by `station machinefile apply` and `station machinefile update`: deleting one makes the next apply write it from what is there"),
		noted(conf("{config}/station/grants/*.env", Linux, MacOS), "mode 0600, the token of a program granted to it that reads one from its environment, sourced by the zsh block"),
		noted(conf("{config}/station/services.env", Linux, MacOS), "its ports' exports, written by the guest"),
		noted(data("{state}/station/guest.json", Linux, MacOS), "the Mac that runs it"),
		noted(data("{state}/station/guest.sock", Linux, MacOS), "the guest's socket, live while it runs"),
		noted(conf("~/.zshrc"), "Machines' main block, and one block per package that declares shell"),
		noted(conf("~/.ssh/config"), "one block per machine"),
		noted(conf(SpwnSkillFiles[0]), "the machine's `spwn` skill, for an AI agent: where it is, the apps present and how to drive spwn, written whole at the end of each `station machinefile apply` and `update`: deleting it, the next one writes it again"),
		noted(conf(SpwnSkillFiles[1]), "the same `spwn` skill, in Claude Code's skill folder"),
		noted(kept("{state}/station/vms/*.json"), "VM records: do not delete"),
		noted(kept("{state}/station/images/key*"), "the only way into a fresh VM: do not delete"),
		noted(kept("{state}/station/leases.json"), "the reserved ports of its Mac: deleting it changes every URL"),
		noted(kept("{state}/station/ledger.json"), "what `station machinefile apply` installed and the `~/.zshrc` blocks it wrote, the only ones `apply --remove` takes back, and what the last night's upgrade did: deleting it makes station remove none"),
		noted(data("{state}/station/ledger.json.lock"), "empty: the lock two applies at once take in turn to change the ledger"),
		noted(data("{state}/station/runs.lock"), "empty: the turn an apply, an upgrade and the night's upgrade take, so one runs at a time"),
	},
	Installed: []string{InstallDir + "/" + StationBinary},
	Free: []Action{remove(
		"{cache}/station/*",
		StationLogs,
		"{state}/station/images/askpass",
		"{state}/station/dns/quad9-dns.mobileconfig",
	)},
	NoErase: stationNoErase,
}

const KeysRow = "ssh"

const StationLogs = "{state}/station/logs/*.log"

const StationVMs = "{data}/station/vms/vms/*"

var SSHDisk = Disk{
	// value: the owner's keys and ssh config: a Disk row of Never touched
	// alone, so `disk erase ssh` is told what station never erases.
	Paths:   []Stored{noted(kept("~/.ssh"), "the owner's keys and ssh_config")},
	NoErase: appsNoErase,
}

var HomeExceptions = []Exception{
	{Path: ".ssh", Reason: "the owner's keys and ssh_config"},
	{Path: "Applications", Reason: "macOS's folder for one user's apps: Spwn.app lives there"},
	{Path: ".gnupg", Reason: "the owner's GPG keys"},
	{Path: ".cargo", Reason: "rust-analyzer started by a GUI IDE would lose the toolchain"},
	{Path: ".rustup", Reason: "rust-analyzer started by a GUI IDE would lose the toolchain"},
	{Path: ".claude", Reason: "Claude Code reads no settings file for its home"},
	{Path: ".codex", Reason: "Codex reads no settings file for its home"},
	{Path: ".zshrc", Reason: "a dotfile"},
	{Path: ".npmrc", Reason: "a dotfile: npm's own settings"},
	{Path: ".gitconfig", Reason: "a dotfile"},
	{Path: ".bunfig.toml", Reason: "a dotfile: bun's global settings"},
	{Path: ".tmux.conf", Reason: "a dotfile"},
	{Path: ".hushlogin", Reason: "a dotfile"},
	{Path: ".claude.json", Reason: "a dotfile"},
	{Path: ".zsh_history", Reason: "a dotfile"},
	{Path: ".zcompdump*", Reason: "a dotfile: zsh's completion dump"},
	{Path: ".oh-my-zsh", Reason: "relocated by ZSH only, a shell variable (open question 48)"},
	{Path: ".gemini", Reason: "no settings file relocates it (open question 48)"},
	{Path: ".pulumi", Reason: "relocated by PULUMI_HOME only (open question 48)"},
	{Path: ".terraform.d", Reason: "no settings file relocates it (open question 48)"},
	{Path: ".ansible", Reason: "relocated by ANSIBLE_HOME only (open question 48)"},
	{Path: ".kube", Reason: "relocated by KUBECONFIG only (open question 48)"},
	{Path: ".expo", Reason: "no settings file relocates it (open question 48)"},
	{Path: ".agents", Reason: "the skills CLI's home, no settings file relocates it (open question 48)"},
	{Path: ".cocoapods", Reason: "relocated by CP_HOME_DIR only (open question 48)"},
	{Path: ".gradle", Reason: "relocated by GRADLE_USER_HOME only (open question 48)"},
	{Path: ".android", Reason: "relocated by ANDROID_USER_HOME only (open question 48)"},
	{Path: ".ollama", Reason: "relocated by OLLAMA_MODELS only (open question 48)"},
	{Path: ".orbstack", Reason: "OrbStack's own home (open question 48)"},
	{Path: ".docker", Reason: "relocated by DOCKER_CONFIG only (open question 48)"},
	{Path: ".agent-browser", Reason: "no settings file relocates it (open question 48)"},
	{Path: ".hermes", Reason: "no settings file relocates it (open question 48)"},
}

func (f AppFile) Label() string {
	if f == FileGoEnv {
		return "go env"
	}
	return string(f)
}

func (f AppFile) KeyLabel(key string) string {
	switch f {
	case FileNpmrc:
		return "npm " + key
	case FileMise:
		return "mise " + strings.TrimPrefix(key, "settings.")
	case FileGoEnv, FileBunfig:
	}
	return key
}

func AppFilePaths(home, prefsDir string, kind MachineKind, vars map[string]string) map[AppFile]string {
	// value: where each tool keeps the file station relocates in: Go's env file
	// where `go env GOENV` says ($GOENV when absolute, "off" writes nothing, else
	// the user config dir: ~/Library/Application Support on darwin, the XDG
	// config root on linux), bun's ~/.bunfig.toml ($XDG_CONFIG_HOME first),
	// mise's config.toml ($MISE_GLOBAL_CONFIG_FILE, then $MISE_CONFIG_DIR, then
	// the config root).
	goenv := filepath.Join(home, "Library", "Application Support", "go", "env")
	if kind == Linux {
		goenv = filepath.Join(prefsDir, "go", "env")
	}
	switch v := vars["GOENV"]; {
	case v == "off":
		goenv = ""
	case filepath.IsAbs(v):
		goenv = v
	}
	bunfig := filepath.Join(home, ".bunfig.toml")
	if filepath.IsAbs(vars["XDG_CONFIG_HOME"]) {
		bunfig = filepath.Join(vars["XDG_CONFIG_HOME"], ".bunfig.toml")
	}
	mise := filepath.Join(prefsDir, "mise", "config.toml")
	if dir := vars["MISE_CONFIG_DIR"]; filepath.IsAbs(dir) {
		mise = filepath.Join(dir, "config.toml")
	}
	if file := vars["MISE_GLOBAL_CONFIG_FILE"]; filepath.IsAbs(file) {
		mise = file
	}
	return map[AppFile]string{FileGoEnv: goenv, FileNpmrc: filepath.Join(home, ".npmrc"), FileBunfig: bunfig, FileMise: mise}
}

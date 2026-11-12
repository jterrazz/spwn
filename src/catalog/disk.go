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
	stationNoErase = "station's data are VMs and images: station machines delete <vm>, station images delete <image>"
	appsNoErase    = "nothing to erase: keys and logins are kept"
)

var StationDisk = Disk{
	Paths: []Stored{
		memos(noted(cache("{cache}/station"), "downloaded station binaries (`station-linux-arm64`) and the last-known sizes a read shows until its walk answers"), "sizes.json"),
		noted(cache("{state}/station/logs/*.log"), "`engine.log`, each VM's run `<vm>.log`, `station-build-<image>.log`, `setup.log`, each app's install `<app>.log`, `accounts.log`"),
		noted(cache("{state}/station/images/askpass"), "0700, the base image's admin password an earlier build left: no build writes it now, `disk free` removes it"),
		noted(cache("{state}/station/dns/quad9-dns.mobileconfig"), "regenerated by the dns setting"),
		noted(cache("{cache}/station/sources/*"), "the shared source's clone, at its default branch: cloned again when it is gone"),
		noted(data("{data}/station/vms/vms/*"), "station's VMs and images: `station machines delete`, `station images delete`"),
		noted(data("{state}/station/images/*/*.json"), "image records, paired with their images"),
		noted(data("{state}/station/known_hosts/*"), "pins: deleting them downgrades to trust-on-first-use, never cleaned"),
		noted(data("{state}/station/consoles/*"), "live while a VM or a build runs: `consoles/<vm>`, `consoles/station-build-<image>`"),
		noted(data("{state}/station/engine.sock"), "the engine's socket, live while it runs"),
		noted(data("{state}/station/children/*"), "the engine's `tailscale` calls in flight, `<pid>` holding its start: the next engine start kills what a killed one left"),
		noted(data("{state}/station/runs/*"), "each running VM's `<vm>.pid` and `<vm>.exit`"),
		noted(data("{state}/station/uptime.json"), "each server's last 30 days, written by its Mac's engine: deleting it starts every record again"),
		noted(data("{state}/station/vms/.records.lock"), "empty: the lock a create and the engine take in turn to write a VM record"),
		noted(data("{state}/station/use/*"), "each terminal's machine (`station use`), forgotten after 7 days unused"),
		noted(conf("~/Library/LaunchAgents/com.jterrazz.station.engine.plist", Mac), "the one engine's launchd agent"),
		noted(conf("{config}/station/preferences.json"), "here, hosts, vm_user, the share list, the ports policy"),
		noted(conf("{config}/station/machinefiles/*.lock"), "each Machinefile's lock, written by `station apply` and `station upgrade`: deleting one makes the next apply write it from what is there"),
		noted(conf("{config}/station/services.env", Linux, MacOS), "its ports' exports, written by the guest"),
		noted(data("{state}/station/guest.json", Linux, MacOS), "the Mac that runs it"),
		noted(data("{state}/station/guest.sock", Linux, MacOS), "the guest's socket, live while it runs"),
		noted(data("{state}/station/agents/*.json", Linux, MacOS), "its agents' journal, one file per session, 0600: each agent's name, task, state, steps and the scrubbed words of its last message, folded from its hooks"),
		noted(conf("~/.zshrc"), "station's main block, and one block per package that declares shell"),
		noted(conf("~/.ssh/config"), "one block per machine"),
		noted(kept("{state}/station/vms/*.json"), "VM records: do not delete"),
		noted(kept("{state}/station/images/key*"), "the only way into a fresh VM: do not delete"),
		noted(kept("{state}/station/leases.json"), "the reserved ports of its Mac: deleting it changes every URL"),
		noted(kept("{state}/station/ledger.json"), "what `station apply` installed and the `~/.zshrc` blocks it wrote, the only ones `apply --remove` takes back, and what the last night's upgrade did: deleting it makes station remove none"),
		noted(data("{state}/station/ledger.json.lock"), "empty: the lock two applies at once take in turn to change the ledger"),
		noted(data("{state}/station/runs.lock"), "empty: the turn an apply, an upgrade and the night's upgrade take, so one runs at a time"),
	},
	Installed: []string{"~/.local/bin/station"},
	Free: []Action{remove(
		"{cache}/station/*",
		"{state}/station/logs/*.log",
		"{state}/station/images/askpass",
		"{state}/station/dns/quad9-dns.mobileconfig",
	)},
	NoErase: stationNoErase,
}

const KeysRow = "ssh"

var SSHDisk = Disk{
	// value: the owner's keys and ssh config: a Disk row of Never touched
	// alone, so `disk erase ssh` is told what station never erases.
	Paths:   []Stored{noted(kept("~/.ssh"), "the owner's keys and ssh_config")},
	NoErase: appsNoErase,
}

const StationRetiredNote = "The keychain entry `station` holds nothing since round 5: `station init` removes the three tailnet accounts the earlier rounds kept there, and `station engine start` retires round 3's per-VM launchd agents, `~/Library/LaunchAgents/com.jterrazz.station.vm.*.plist`: booted out and removed, the one engine runs every VM."

var HomeExceptions = []Exception{
	{Path: ".ssh", Reason: "the owner's keys and ssh_config"},
	{Path: "Applications", Reason: "macOS's folder for one user's apps: station-menubar puts Station.app there"},
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

package world

import (
	"context"
	"encoding/json"
	"maps"
	"slices"
	"sync"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/activity"
	"github.com/jterrazz/studio-station/apps/engine/agents"
	"github.com/jterrazz/studio-station/apps/engine/automations"
	"github.com/jterrazz/studio-station/apps/engine/awake"
	"github.com/jterrazz/studio-station/apps/engine/background"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/journal"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/engine"
	"github.com/jterrazz/studio-station/apps/engine/images"
	"github.com/jterrazz/studio-station/apps/engine/machinefile"
	"github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/ports"
	"github.com/jterrazz/studio-station/apps/engine/settings"
	"github.com/jterrazz/studio-station/apps/engine/teams"
)

type World struct {
	// value: each method's manager as its real adapter names it; set by
	// the wiring, never read from a file.
	Managers map[catalog.Method]Named `json:"-"`
	// invariant: every fake over the world holds this lock while it reads
	// or writes it, as the engine's goroutines and a command share it.
	mu sync.Mutex
	// value: what a held call does before it waits: the run writes its world
	// back and lets go of it, so a command run meanwhile reads what the held
	// one has made so far; nil in a run that writes nothing back.
	paused func()
	// value: closes once the world's file no longer declares the hold, so a
	// world unhold run as another command lets go of the call it keeps; nil
	// in a run that writes nothing back.
	unheld func(context.Context, Fault) <-chan struct{}

	Now time.Time `json:"now"`
	// value: the Mac the binary runs on, and its owner's login.
	ThisMac string          `json:"this_mac"`
	User    string          `json:"user,omitempty"`
	Macs    map[string]*Mac `json:"macs"`
	// value: what the engine told the owner, in the order told.
	Told []engine.Notification `json:"told,omitempty"`
	// value: each image's latest version, as its registry answers it.
	Registry map[string]string `json:"registry,omitempty"`
	// value: the one account every image pulled from its publisher ships
	// with; nil when the world's pulled images take station's key.
	UpstreamLogin *ImageLogin `json:"upstream_login,omitempty"`
	// value: how each next lockdown run in a guest is cut, in order: before
	// it closes the image's login, before its key goes in, after it and
	// before its marker, never ending, or its connection dropped once it
	// printed its new host key, before its key goes in.
	LockdownCuts []string `json:"lockdown_cuts,omitempty"`
	// value: the failures injected into the fakes, in the order scripted.
	Script []Fault `json:"script,omitempty"`
	// value: the holds declared for a VM whose call waits now: written with
	// the world its held call lets go of, gone once that call goes on.
	Held []Fault `json:"held,omitempty"`
	// value: every command station asked a Mac of the world to run as root,
	// each refused (SC-03: station never runs sudo).
	AsRoot []string `json:"as_root,omitempty"`
	// value: the commands that follow run with admin rights, on This Mac's
	// own terminal.
	Admin bool `json:"admin,omitempty"`
	// value: the repositories a shared source is read from, by URL.
	Sources map[string]*Repository `json:"sources,omitempty"`
	// value: the names a health check asks that answer no more, a site
	// down whichever machine serves it: example.com.
	SitesDown []string `json:"sites_down,omitempty"`
	// value: station's releases on GitHub, in the order made: the
	// last is the latest.
	Releases []Release `json:"releases,omitempty"`
}

type Release struct {
	Tag    string   `json:"tag"`
	URL    string   `json:"url"`
	Assets []string `json:"assets,omitempty"`
}

// value: a repository's files at each of its commits, by slash path,
// the commit its default branch is at, and its history.

type Repository struct {
	Commits map[string]map[string]string `json:"commits"`
	Head    string                       `json:"head"`
	// value: each commit's parent, so a history is read: a commit with
	// none written starts one.
	Parents map[string]string `json:"parents,omitempty"`
}

func (r *Repository) Descends(commit, from string) bool {
	// invariant: a commit descends from itself and from every commit its
	// parents lead back to.
	for seen := 0; commit != "" && seen <= len(r.Commits); seen++ {
		if commit == from {
			return true
		}
		commit = r.Parents[commit]
	}
	return false
}

type Fault struct {
	// value: the method that fails, "<Contract>.<Method>": "Installer.Install".
	Call string `json:"call"`
	// value: how many more calls fail; zero fails every call.
	Times int `json:"times,omitempty"`
	// value: the next call does its first step, then waits until it is
	// interrupted, as a long install caught by ctrl-c.
	Hold bool `json:"hold,omitempty"`
	// value: the one VM of one Mac a hold the world file declares keeps:
	// every call of it waits, for the world's life, so its create stays at
	// that step and every read meanwhile sees it there; empty, a hold of
	// the next call, whichever VM it is of.
	Mac string `json:"mac,omitempty"`
	VM  string `json:"vm,omitempty"`
}

type Mac struct {
	// value: a run of this Mac holds its turn (an apply, an upgrade, the
	// night's), within one process: a run in another reads it free.
	turn bool
	// value: the engine of this Mac; nil when none was ever set up.
	Engine *Engine `json:"engine,omitempty"`
	// value: FileVault waits for its password at pre-boot: nothing of the
	// Mac runs, its engine neither.
	Locked bool `json:"locked,omitempty"`
	// value: out of the tailnet's reach.
	Offline bool `json:"offline,omitempty"`
	// value: the owner runs station as root on this Mac (sudo ~/.local/bin/station), as a
	// setting that needs root asks; station itself never runs sudo.
	Root bool `json:"root,omitempty"`
	// value: a line run as root asks no password on this machine, as in a
	// VM station made, whose admin it is; the owner's Mac asks one.
	Passwordless bool `json:"passwordless,omitempty"`
	// value: a Mac that restarts does not answer until then.
	Back time.Time `json:"back,omitzero"`
	// value: the terminals the owner's terminal was put on, each
	// "<machine>/<terminal>".
	Attached []string `json:"attached,omitempty"`
	// value: the hypervisor this Mac has: empty, the one station is built
	// with; none, no hypervisor installed; another name, a second fake that
	// stands for another adapter (vz).
	Hypervisor string `json:"hypervisor,omitempty"`
	// value: the programs a need of station is provided by that this Mac has
	// not installed (tailscale, studio); its hypervisor is Hypervisor's.
	Absent []string `json:"absent,omitempty"`
	// value: the lines station wrote in this Mac's engine log, the oldest
	// first, read in place of the file of the run's own state folder.
	EngineLog []string `json:"engine_log,omitempty"`
	// value: the station version this Mac runs; empty, the spec binary's.
	Station string `json:"station,omitempty"`
	// value: the calls that older station does not serve, until an upgrade.
	Lacks []string `json:"lacks,omitempty"`
	// value: what station keeps in this Mac's preferences, by group.
	Preferences map[string]json.RawMessage `json:"preferences,omitempty"`
	// value: station's own files as text, by name, while no store has read
	// them: a file edited by hand, a file station v2 left.
	Files map[string]string `json:"files,omitempty"`
	// value: the commit each shared source's clone is at, by the source's
	// url.
	Clones map[string]string `json:"clones,omitempty"`
	// value: the images the hypervisor holds, by reference.
	Images []string `json:"images,omitempty"`
	// value: the VMs station's hypervisor adapter cloned here and has not
	// deleted since, by name, each with the Born of the disk it cloned: its
	// ledger, which a VM made or renamed by hand never enters.
	Cloned map[string]int `json:"cloned,omitempty"`
	// value: the image builds this Mac is at when the world opens, each at
	// its step.
	Building []images.Making `json:"building,omitempty"`
	// value: a VM freshly cloned from a fetched image, before anything ran
	// in it: its size and the guest it answers with; nil clones a disk whose
	// guest is station's image's.
	Fresh *VM            `json:"fresh,omitempty"`
	VMs   map[string]*VM `json:"vms,omitempty"`
	// value: the processes of station that run on this Mac besides its
	// engine, a create a command runs: a reboot ends them.
	Processes []machines.Process `json:"processes,omitempty"`

	Jobs map[string]*Job `json:"jobs,omitempty"`
	// value: the job manager's overrides by label: enabled or not.
	Overrides map[string]bool `json:"overrides,omitempty"`
	// value: what station keeps in this Mac's vault, by name.
	Vault map[string][]byte `json:"vault,omitempty"`
	// value: what this machine's programs, sign-ins, passwords and grants are.
	Accounts *Accounts `json:"accounts,omitempty"`
	// value: the facts this Mac answers, by fact; a fact absent here is one
	// it cannot look at.
	Facts map[catalog.Fact]settings.Fact `json:"facts,omitempty"`
	// value: what each setting a procedure writes holds now, by setting:
	// "station's" as station applied it, or the owner's own text.
	Settings map[string]string `json:"settings,omitempty"`
	// value: what Self reads; nil reads a Mac on macOS.
	Self *Self `json:"self,omitempty"`
	// value: what the power hardware reads now, its capabilities with it (a
	// battery, a lid); nil on a machine with no power management.
	Power *Power `json:"power,omitempty"`
	// value: what listens on this Mac, one row per listening address.
	Listening []ports.Listener `json:"listening,omitempty"`
	// value: this Mac's node on the tailnet; nil when it is signed out.
	Tailnet *Node `json:"tailnet,omitempty"`
	// value: each installer's state, by method; a method absent here has its
	// manager installed and holds nothing.
	Apps map[catalog.Method]*Manager `json:"apps,omitempty"`
	// value: the apps never opened on this Mac: a command one of them brings
	// at its first launch (Herdr.app's hc) is absent until then.
	Unopened []string `json:"unopened,omitempty"`
	// value: the counters that move, read by the meter.
	Meter *Meter `json:"meter,omitempty"`
	// value: the volume and the files on it, by absolute path.
	Disk *Disk `json:"disk,omitempty"`
	// value: the ssh client config's managed Host blocks, by name, and the
	// names defined outside them.
	SSH *SSH `json:"ssh,omitempty"`
	// value: station's own records kept beside the preferences, by store.
	Stores *Stores `json:"stores,omitempty"`
	// value: the hardware address each Wake-on-LAN packet this Mac sent
	// named, in order.
	WakeSent []string `json:"wake_sent,omitempty"`
	// value: how this Mac stands with macOS's Local Network approval of
	// station; empty, a read that cannot tell. While not approved, its
	// engine's dial to any of its VMs is refused.
	LocalNetwork machines.LocalNetwork `json:"local_network,omitempty"`
	AgentHeld    *AgentHolder          `json:"agent_held,omitempty"`
}

type AgentHolder struct {
	PID        int    `json:"pid"`
	Executable string `json:"executable,omitempty"`
	Label      string `json:"label,omitempty"`
}

type Self struct {
	Kind     machine.Kind     `json:"kind,omitempty"`
	System   machine.System   `json:"system"`
	Hardware machine.Hardware `json:"hardware"`
}

type Power struct {
	awake.Reading
	// value: the awake holds taken now, by name, and whether the lid is held.
	Holds     []string `json:"holds,omitempty"`
	Clamshell bool     `json:"clamshell,omitempty"`
	// value: how many times the display was put to sleep.
	DisplaySleeps int `json:"display_sleeps,omitempty"`
	// value: macOS told this Mac it is going to sleep, and it has not woken.
	Sleeping bool `json:"sleeping,omitempty"`
}

type Node struct {
	Self   ports.Self    `json:"self"`
	Shares []ports.Share `json:"shares,omitempty"`
	// value: the host's certificate is asked for and not issued yet.
	CertificatePending bool `json:"certificate_pending,omitempty"`
	// value: the ports its Funnel config opens to the internet.
	Funnel []int `json:"funnel,omitempty"`
}

// value: a method's manager as its real adapter names it: its word in a
// row's detail, and the catalogue app that brings it; a method that runs
// no manager names neither.

type Named struct {
	Name string
	App  string
	// value: what its real adapter can do.
	Install, Update, Remove, UpdateAll bool
}

type Manager struct {
	// value: the manager itself is not installed.
	Absent    bool              `json:"absent,omitempty"`
	Installed map[string]string `json:"installed,omitempty"`
	// value: the keys a newer version waits for, with that version.
	Behind map[string]string `json:"behind,omitempty"`
	// value: the version an install of a key brings.
	Offers map[string]string `json:"offers,omitempty"`
	// value: the other versions the manager offers of a key, any order: an
	// install within a range takes the newest of them it holds.
	Versions map[string][]string `json:"versions,omitempty"`
	// value: the manager reads the newest version it offers of a key,
	// whatever range holds it, as its real adapter does (mise's `latest`).
	ReadsNewest bool `json:"reads_newest,omitempty"`
	// value: the methods an install of a key brings, their manager present
	// from then on: Homebrew's script lands brew and cask.
	Lands map[string][]catalog.Method `json:"lands,omitempty"`
	// value: for a method that holds what it adds by app (apt), the apps
	// whose installs added each key; a key it lists none of is the owner's.
	Holders map[string][]string `json:"holders,omitempty"`
}

type Meter struct {
	Sample activity.Sample `json:"sample"`
	// value: how much each counter of Sample grows a second from Sample's
	// time, so two reads apart in the world's time give a rate.
	PerSecond *activity.Sample `json:"per_second,omitempty"`
	// value: a busy share that varies with the world's clock: CPUBusy grows
	// by CPUTotal's rate at a level that changes every slot, so two reads
	// apart differ and two runs of one world agree.
	Load *Load            `json:"load,omitempty"`
	Top  activity.Busiest `json:"top"`
	// value: the processes that run, by pid.
	Processes map[int]activity.Usage `json:"processes,omitempty"`
}

// value: a machine's busy share as a seeded curve of the world's clock: one
// level a slot, in thousandths of CPUTotal's rate, between Floor and Ceil,
// drawn from Seed and the slot's number alone; a zero slot is a minute.

type Load struct {
	Seed        uint64 `json:"seed"`
	Floor       int    `json:"floor"`
	Ceil        int    `json:"ceil"`
	SlotSeconds int    `json:"slot_seconds,omitempty"`
}

type Disk struct {
	Total int64 `json:"total,omitempty"`
	Free  int64 `json:"free,omitempty"`
	// value: what its APFS container's other volumes hold; 0 for a volume
	// that does not tell its system apart.
	System int64            `json:"system,omitempty"`
	Files  map[string]int64 `json:"files,omitempty"`
	// value: the processes that run, by name.
	Running []string `json:"running,omitempty"`
	// value: how late each size answers.
	Late time.Duration `json:"late,omitempty"`
}

type SSH struct {
	Blocks  map[string]SSHBlock `json:"blocks,omitempty"`
	Foreign []string            `json:"foreign,omitempty"`
}

type SSHBlock struct {
	User    string `json:"user,omitempty"`
	Key     string `json:"key,omitempty"`
	Behind  string `json:"behind,omitempty"`
	ThisMac bool   `json:"this_mac,omitempty"`
}

type Stores struct {
	Images       []images.Record             `json:"images,omitempty"`
	Reservations []ports.Reservation         `json:"reservations,omitempty"`
	Declarations []ports.Declaration         `json:"declarations,omitempty"`
	Applied      map[string]settings.Applied `json:"applied,omitempty"`
	// value: each server's uptime record, by name.
	Uptime map[string]machines.Uptime `json:"uptime,omitempty"`
	// value: what station installed for a Machinefile, by package.
	Ledger []machinefile.Installed `json:"ledger,omitempty"`
	// value: the last night the engine's upgrade recorded.
	Night *machinefile.Night `json:"night,omitempty"`
	// value: when each Machinefile was last applied, by its name.
	Applies map[string]time.Time `json:"applies,omitempty"`
	// value: the blocks of ~/.zshrc apply wrote, by id.
	Blocks []machinefile.Written `json:"blocks,omitempty"`
	// value: what the engine keeps of its last asks of the latest release.
	Release *engine.ReleaseKept `json:"release,omitempty"`
	// value: what was done on this Mac and its VMs, oldest first.
	Journal []journal.Line `json:"journal,omitempty"`
	// value: the agents' Minds kept on this machine, by the project root
	// they belong to ("" for the user's own), each list by name.
	Agents map[string][]agents.Mind `json:"agents,omitempty"`
	// value: the agents' archives written on this machine, by path.
	AgentArchives map[string]agents.Mind `json:"agent_archives,omitempty"`
	// value: the teams declared for this machine's user, by slug; their
	// members are read from Agents, never kept here.
	Teams []teams.Team `json:"teams,omitempty"`
	// value: the automations a project declares, by the project root they
	// belong to; what their runs say is read from AutomationRuns.
	Automations map[string][]automations.Automation `json:"automations,omitempty"`
	// value: the runs written for a project, by its root, oldest first.
	AutomationRuns map[string][]automations.Run `json:"automation_runs,omitempty"`
}

func (m *Mac) Store() *Stores {
	// invariant: the stores of a Mac, made empty the first time they are asked.
	if m.Stores == nil {
		m.Stores = &Stores{}
	}
	return m.Stores
}

type Engine struct {
	// value: the version the engine runs; empty is the binary's own.
	Version string `json:"version,omitempty"`
	// value: the abilities of its own calls that older engine does not
	// serve, until it starts again.
	Lacks []string `json:"lacks,omitempty"`
	// value: when the engine that runs now started.
	StartedAt time.Time `json:"started_at,omitzero"`
	// value: set up, and stopped by the owner: when, and by what.
	Stopped   bool      `json:"stopped,omitempty"`
	StoppedAt time.Time `json:"stopped_at,omitzero"`
	StoppedBy string    `json:"stopped_by,omitempty"`
	// value: the binary the engine last started from, as it kept it beside
	// its socket.
	Recorded *engine.Record `json:"recorded,omitempty"`
	// value: what the engine knows of its VMs between two commands.
	Memory engine.Memory `json:"memory,omitempty"`
	// value: what the engine holds of Awake between two commands.
	Awake *engine.AwakeKept `json:"awake,omitempty"`
	// value: what the exposure job knows between two commands.
	Exposure *engine.ExposureKept `json:"exposure,omitempty"`
	// value: the activity the engine holds in its memory between two
	// commands, by machine; its successor after a restart holds none.
	Activity map[string]activity.Kept `json:"activity,omitempty"`
	// value: the activity the engine kept before the world's now, by
	// machine, as profiles: unfolded into Activity when the world is read.
	Past map[string]activity.Profile `json:"past,omitempty"`
	// value: the last settings read of each workspace the engine holds in
	// its memory between two commands; its successor after a restart holds
	// none.
	Settings engine.SettingsKept `json:"settings,omitempty"`
	// value: what the engine copied of the other Macs between two commands.
	Held engine.Held `json:"held,omitempty"`
	// value: the menu bar runs against this engine between two commands,
	// "closed" or "open": closed it holds its page and events watches,
	// open it watches This Mac's activity too.
	MenuBar string `json:"menubar,omitempty"`
	// value: the machines whose Activity the menu bar shows between two
	// commands.
	Watching []string `json:"watching,omitempty"`
}

func (m *Mac) EngineRuns() bool {
	// value: whether this Mac's engine answers now.
	return m.Engine != nil && !m.Engine.Stopped && !m.Locked
}

type VM struct {
	// value: what station keeps of the VM; nil for a disk it has no record of.
	Kept *Kept `json:"kept,omitempty"`
	// value: the hypervisor holds its disk.
	Disk bool `json:"disk,omitempty"`
	// value: which disk it is, as its folder's identity tells one from
	// another: a clone gives each a number of the world's own, and a disk
	// made by hand has none (0), so one deleted and made again under the
	// name is never the disk the ledger kept.
	Born int `json:"born,omitempty"`
	// value: its record's file does not parse: the store sees a file of
	// that name and never a record.
	Unreadable bool `json:"unreadable,omitempty"`
	// value: a create cloned it from an image in this world: once its
	// guest serves station's API it is a machine of its own, a workspace or
	// a server on Linux, sized as it was made.
	Made bool          `json:"made,omitempty"`
	Size machines.Size `json:"size"`
	// value: the pid of its run; zero when it does not run.
	PID int `json:"pid,omitempty"`
	// value: the hypervisor holds it suspended while it does not run.
	Suspended bool `json:"suspended,omitempty"`
	// value: how its last run ended, kept until a run is read.
	Ended *machines.RunEnd `json:"ended,omitempty"`
	// value: when its last run began and when its last run ended, on the
	// world's clock: what a wait between the two took in the world.
	UpAt   time.Time `json:"up_at,omitzero"`
	DownAt time.Time `json:"down_at,omitzero"`
	// value: what its last boot was asked: its network, and the folders of
	// its Mac shared into it as tart's --dir says them, only the vm.mounts
	// of the Machinefile it was made from (SC-06: nothing else).
	Booted  *Boot  `json:"booted,omitempty"`
	Address string `json:"address,omitempty"`
	// value: the lines its runs wrote on its console, the oldest first.
	Console []string `json:"console,omitempty"`
	// value: the lines its hypervisor's own output wrote, the oldest first,
	// read after the console's as the hypervisor's adapter reads them.
	Hypervisor []string `json:"hypervisor,omitempty"`
	// value: how long its health check takes to answer; zero is the world's usual.
	CheckTakes time.Duration `json:"check_takes,omitempty"`
	// value: its guest, answering over the one way into a VM; nil when no
	// guest answers.
	Guest *Guest `json:"guest,omitempty"`
	// value: what listens inside the VM.
	Listening []ports.Listener `json:"listening,omitempty"`
	// value: the containers run inside the VM, each with the ports
	// it publishes.
	Containers []Container `json:"containers,omitempty"`
}

type Container struct {
	Name   string `json:"name"`
	Shared []int  `json:"ports"`
}

type Boot struct {
	Bridged bool     `json:"bridged"`
	Folders []string `json:"folders"`
}

type Kept struct {
	Image     string            `json:"image,omitempty"`
	Version   string            `json:"version,omitempty"`
	Lifetime  machines.Lifetime `json:"lifetime"`
	Want      string            `json:"want,omitempty"`
	Network   string            `json:"network,omitempty"`
	Check     string            `json:"check,omitempty"`
	Env       map[string]string `json:"env,omitempty"`
	CreatedAt time.Time         `json:"created_at,omitzero"`
	Birth     *machines.Birth   `json:"birth,omitempty"`
	// value: the process its create runs in, while it runs.
	Owner *machines.Process `json:"owner,omitempty"`
	// value: the Machinefile it was made from.
	Template string `json:"template,omitempty"`
	// value: the folders its last start shared.
	Mounts []machines.Mount `json:"mounts,omitempty"`
	// value: what it is for; the session that made it, empty for the owner.
	Purpose string `json:"purpose,omitempty"`
	Creator string `json:"creator,omitempty"`
}

func (v *VM) Running() bool { return v.PID != 0 }

func (v *VM) Down(now time.Time, end *machines.RunEnd) {
	if v.Running() {
		v.DownAt = now
	}
	v.PID, v.Ended = 0, end
}

type Guest struct {
	// value: what the guest answers to each command line it is given; a
	// line ending in * answers every line it begins.
	Answers map[string]Answer `json:"answers,omitempty"`
	// value: the VM's own tooling replaced station's guest: its ssh answers,
	// station's session does not.
	Replaced bool `json:"replaced,omitempty"`
	// value: made from station's image by a create: a command no answer
	// scripts succeeds, as the image's first boot does.
	Fresh bool `json:"fresh,omitempty"`
	// value: every command line station ran in the VM, apart from its own
	// session and the liveness probe, in the order run: what it set up inside.
	Ran []string `json:"ran,omitempty"`
	// value: the one login it takes: a clone of an image as its publisher
	// ships it, before station's key is in it; nil takes station's key.
	Login *ImageLogin `json:"login,omitempty"`
	// value: takes station's key while its lockdown's marker is absent.
	Unlocked bool `json:"unlocked,omitempty"`
	// value: how many host keys its lockdowns made; it serves the last.
	HostKeys int `json:"host_keys,omitempty"`
	// value: takes no login at all: its lockdown closed the image's login
	// and never put station's key in.
	Closed bool `json:"closed,omitempty"`
}

const (
	CutBeforeLock      = "before-lock"
	CutBeforeKey       = "before-key"
	CutAfterKey        = "after-key"
	CutNeverEnds       = "never-ends"
	CutDropsAfterPrint = "drops-after-print"
)

type ImageLogin struct {
	User     string `json:"user"`
	Password string `json:"password"`
}

type Answer struct {
	Code   int    `json:"code,omitempty"`
	Stdout string `json:"stdout,omitempty"`
	Stderr string `json:"stderr,omitempty"`
	// value: the line powers the VM off once it is answered.
	Poweroff bool `json:"poweroff,omitempty"`
}

type Job struct {
	background.Job
	// value: the definition as it was written, so writing it again is seen.
	Definition string `json:"definition,omitempty"`
}

func (w *World) Lock()   { w.mu.Lock() }
func (w *World) Unlock() { w.mu.Unlock() }

func Decode(data []byte) (*World, error) {
	w := &World{}
	if err := json.Unmarshal(data, w); err != nil {
		return nil, err
	}
	w.unfold()
	return w, nil
}

func (w *World) Encode() ([]byte, error) {
	data, err := json.MarshalIndent(w, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func (w *World) UseHypervisor(hypervisor string) {
	// invariant: AR-02: the same world over another adapter. Every Mac that
	// has the hypervisor station is built with gets the one named instead;
	// a Mac with none installed, or one named already, keeps its own.
	if hypervisor == "" {
		return
	}
	for _, m := range w.Macs {
		if m.Hypervisor == "" {
			m.Hypervisor = hypervisor
		}
	}
}

func (m *Mac) Reads(f catalog.Fact, value string) {
	// value: the Mac answers fact f with value from now on.
	if m.Facts == nil {
		m.Facts = map[catalog.Fact]settings.Fact{}
	}
	m.Facts[f] = settings.Fact{State: catalog.StateOK, Value: value}
}

func (w *World) Mac(name string) *Mac {
	// invariant: Mac answers the Mac of that name, made empty the first time it
	// is asked, so a fake writes its slice without checking it exists.
	if w.Macs == nil {
		w.Macs = map[string]*Mac{}
	}
	m := w.Macs[name]
	if m == nil {
		m = &Mac{}
		w.Macs[name] = m
	}
	if m.VMs == nil {
		m.VMs = map[string]*VM{}
	}
	return m
}

func (m *Mac) TakeTurn() bool {
	// invariant: called under the world's lock; true when the turn was free
	// and is now held.
	if m.turn {
		return false
	}
	m.turn = true
	return true
}

func (m *Mac) GiveTurn() { m.turn = false }

func (w *World) Fails(call string) bool {
	// invariant: a scripted fault fails its call and is spent; one with no
	// count fails every call.
	for i, f := range w.Script {
		if f.Call != call || f.Hold {
			continue
		}
		switch f.Times {
		case 0:
		case 1:
			w.Script = append(w.Script[:i], w.Script[i+1:]...)
		default:
			w.Script[i].Times--
		}
		return true
	}
	return false
}

func (w *World) Holds(call string) bool {
	// invariant: a scripted hold is spent by the one call it holds.
	for i, f := range w.Script {
		if f.Call == call && f.Hold && f.VM == "" {
			w.Script = append(w.Script[:i], w.Script[i+1:]...)
			return true
		}
	}
	return false
}

func (w *World) Keeps(call, mac, vm string) bool {
	// invariant: a hold declared for one VM of one Mac is never spent.
	return slices.ContainsFunc(w.Script, func(f Fault) bool {
		return f.Call == call && f.Hold && f.VM == vm && f.Mac == mac
	})
}

func (w *World) Unheld(ctx context.Context, call, mac, vm string) <-chan struct{} {
	// value: closes once a world unhold lets go of the hold the world
	// declares for vm of mac; nil, so never, for a hold of the next call.
	if w.unheld == nil || !w.Keeps(call, mac, vm) {
		return nil
	}
	return w.unheld(ctx, Fault{Call: call, Hold: true, Mac: mac, VM: vm})
}

func (w *World) OnUnhold(f func(context.Context, Fault) <-chan struct{}) { w.unheld = f }

func (w *World) Unhold(mac, vm string) {
	// value: every hold the script declares or keeps for the next call is
	// gone; given a VM of a Mac, only the holds declared for it.
	w.Script = slices.DeleteFunc(w.Script, func(f Fault) bool {
		return f.Hold && (mac == "" || f.Mac == mac && f.VM == vm)
	})
}

func (w *World) Reach(f Fault) {
	// value: f, a hold declared for a VM, is reached: its call waits now.
	w.Held = append(w.Held, f)
}

func (w *World) Leave(f Fault) {
	// value: the call f held goes on: it waits no more.
	if i := slices.Index(w.Held, f); i >= 0 {
		w.Held = slices.Delete(w.Held, i, i+1)
	}
	if len(w.Held) == 0 {
		w.Held = nil
	}
}

func (w *World) Pause() {
	// invariant: called under the world's lock, by a fake about to wait on
	// a held call.
	if w.paused != nil {
		w.paused()
	}
}

func (w *World) OnPause(f func()) { w.paused = f }

func (w *World) Holder(vm string) string {
	// value: the Mac that holds vm, empty when none does; of two Macs that
	// each hold a VM of that name, the first by name, so a run never
	// depends on the map's order.
	for _, name := range slices.Sorted(maps.Keys(w.Macs)) {
		if _, ok := w.Macs[name].VMs[vm]; ok {
			return name
		}
	}
	return ""
}

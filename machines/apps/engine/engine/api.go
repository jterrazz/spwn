package engine

import (
	"time"

	"github.com/jterrazz/studio-station/apps/engine/activity"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/settings"
)

// value: the names of the engine's own calls: what a process of this Mac
// asks of the engine's live state (its runs, its leases, its guest
// connections, its Power), served beside the registry; enginecalls.Owns
// says the types each reads and answers.
const (
	CallHello     = "engine.hello"
	CallVMs       = "engine.vms"
	CallPorts     = "engine.ports"
	CallReserve   = "engine.reserve"
	CallRelease   = "engine.release"
	CallGuestExec = "engine.guest-exec"
	CallStopAll   = "engine.stop-all"
	CallAwakeView = "engine.awake"
	CallKeepAwake = "engine.awake-keep"
	CallAwakeOff  = "engine.awake-off"
	CallRefresh   = "engine.refresh"
	// value: the activity this engine keeps, of This Mac and of each VM it
	// sees, in its memory alone.
	CallActivity = "engine.activity"
	// value: the last settings read of a workspace, kept by the engine in its
	// memory alone, and asked back while it is stopped.
	CallKeepSettings = "engine.settings-keep"
	CallLastSettings = "engine.settings-last"
	// value: the watch of the engine's pages, and of what it tells.
	CallPage   = "page"
	CallEvents = EventsStream
	// value: the watch a surface holds while it shows a machine's activity:
	// that machine is sampled every 2 s until it leaves.
	CallWatch = "engine.watch"
)

const (
	// value: what the engine's own calls read of an input that an older
	// engine decodes as absent, each named `<call>.<field>`: its hello lists
	// those it serves, and a caller asks one only of an engine that lists it.
	AbilityReserveTo = CallReserve + ".to"
)

func Abilities() []string { return []string{AbilityReserveTo} }

// value: what a VM's verb asks of the engine, by the verb's name.
const (
	VMRun     = "run"
	VMStart   = "start"
	VMStop    = "stop"
	VMRestart = "restart"
	VMDelete  = "delete"
	VMForget  = "forget"
	VMReady   = "ready"
)

func VMVerbs() []string {
	// value: the verbs of the engine's calls on one VM, engine.vm-<verb>.
	return []string{VMRun, VMStart, VMStop, VMRestart, VMDelete, VMForget, VMReady}
}

func CallVM(verb string) string { return "engine.vm-" + verb }

type KeepSettingsIn struct {
	VM   string            `json:"vm"`
	Read settings.LastRead `json:"read"`
}

type LastSettingsOut struct {
	Read settings.LastRead `json:"read"`
	Kept bool              `json:"kept"`
}

type VMIn struct {
	VM string `json:"vm"`
}

type ReserveIn struct {
	VM        string `json:"vm"`
	GuestPort int    `json:"guest_port"`
	// value: the audience the owner chose; empty keeps the reservation's.
	To catalog.Audience `json:"to,omitempty"`
}

type ExecIn struct {
	VM   string `json:"vm"`
	Line string `json:"line"`
	// invariant: what a command inside a VM reads, a secret included: it travels
	// in the input alone, never in Line.
	Stdin string `json:"stdin,omitempty"`
	// value: the call of the guest's own API the exchange carries, when
	// Line runs it; the engine reads a VM's drift again after one of
	// Options.Changes.
	Call string `json:"call,omitempty"`
}

type ExecOut struct {
	Code   int    `json:"code"`
	Stdout string `json:"stdout"`
	Stderr string `json:"stderr"`
}

type ActivityIn struct {
	// value: the machine whose history is read, and what its reader reads
	// of it (activity.Want): the window its graphs draw, the 24 h.
	Machine string        `json:"machine"`
	Graph   time.Duration `json:"graph,omitempty"`
	History bool          `json:"history,omitempty"`
}

type Kept struct {
	Last activity.Reading `json:"last"`
	// value: the part of its tiers (ADR-107) the reader reads, each the
	// buckets it holds of it.
	Series activity.Series `json:"series"`
}

type ActivityOut struct {
	// value: when the engine started: every window begins there; Kept none
	// when it keeps nothing of the machine.
	Since time.Time `json:"since"`
	Kept  *Kept     `json:"kept,omitempty"`
}

type WatchIn struct {
	// value: the machines watched by name; none is This Mac.
	Machines []string `json:"machines,omitempty"`
}

type PagesIn struct {
	Pages []string `json:"pages"`
}

type PageWatchIn struct {
	Pages []string `json:"pages"`
	// value: the revision of `station.page/v1` the client draws, judged by
	// FitOf in the watch's opening.
	Revision int `json:"revision"`
}

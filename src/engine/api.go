package engine

import (
	"time"

	"github.com/jterrazz/studio-station/src/activity"
	"github.com/jterrazz/studio-station/src/settings"
)

// value: the names of the engine's own calls: what a process of this Mac
// asks of the engine's live state (its runs, its leases, its guest
// connections, its Power), served beside the registry and never in calls.json.
const (
	CallHello     = "engine.hello"
	CallVMs       = "engine.vms"
	CallPorts     = "engine.ports"
	CallReserve   = "engine.reserve"
	CallRelease   = "engine.release"
	CallGuestExec = "engine.guest-exec"
	CallAccounts  = "engine.accounts"
	CallStopAll   = "engine.stop-all"
	CallAwakeView = "engine.awake"
	CallKeepAwake = "engine.awake-keep"
	CallAwakeOff  = "engine.awake-off"
	CallRefresh   = "engine.refresh"
	// value: the names every machine's agents and workspaces hold, which a
	// workspace asks before it names an agent.
	CallAgentsHeld = "engine.agents-held"
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
}

type ExecIn struct {
	VM   string `json:"vm"`
	Line string `json:"line"`
	// invariant: what a command inside a VM reads, a secret included: it travels
	// in the input alone, never in Line.
	Stdin string `json:"stdin,omitempty"`
}

type ExecOut struct {
	Code   int    `json:"code"`
	Stdout string `json:"stdout"`
	Stderr string `json:"stderr"`
}

type Kept struct {
	Machine string            `json:"machine"`
	Last    activity.Reading  `json:"last"`
	Fine    []activity.Sample `json:"fine"`
	Coarse  []activity.Sample `json:"coarse"`
}

type ActivityOut struct {
	// value: when the engine started: every window begins there.
	Since    time.Time `json:"since"`
	Machines []Kept    `json:"machines"`
}

type WatchIn struct {
	// value: the machines watched by name; none is This Mac.
	Machines []string `json:"machines,omitempty"`
}

type PagesIn struct {
	Pages []string `json:"pages"`
}

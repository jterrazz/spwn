package engine

type Work struct {
	Answering bool
	VMs       []Stopped
	// value: the servers that run: a stop skips them, and the engine keeps
	// running for them (SPEC §5.11).
	Servers []string
}

func (w Work) Empty() bool { return len(w.VMs) == 0 }

type StopReport struct {
	VMs []Stopped
	// value: "ended" (the engine ended the Awake session), "reset" (the CLI
	// set a dead engine's clamshell back to normal), or "" (nothing held, or
	// no keep-awake here).
	Awake string
	// value: the engine stopped sharing the ports of the VMs it stopped.
	Sharing bool
	// value: "stopped" (booted out now), "already stopped", "kept" (a VM
	// would not stop, so the engine still supervises it), or "for servers"
	// (a server runs, which the engine keeps running).
	Engine string
	// value: the servers it skipped.
	Servers []string
}

const (
	AwakeEnded    = "ended"
	AwakeReleased = "released"
)

const (
	EngineHalted  = "stopped"
	EngineAlready = "already stopped"
	EngineKept    = "kept"
	EngineServers = "for servers"
)

type StartReport struct {
	Already bool
	Version string
	PID     int
	// value: what the start did to the engine's job before it ran.
	Installed Installed
}

type Installed struct {
	// value: what bringing the engine's job to this station changed: written
	// for the first time (or changed), moved from v2's label, restarted, and
	// the version it ran before ("" when it did not answer).
	Written, Moved, Restarted bool
	From                      string
}

type StartFailedError struct {
	// value: the last line the engine's log holds, or the job manager's
	// answer when the log holds none.
	Last  string
	Cause error
}

func (e *StartFailedError) Error() string { return "the engine did not start" }

func (e *StartFailedError) Unwrap() error { return e.Cause }

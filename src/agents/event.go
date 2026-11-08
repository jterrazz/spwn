package agents

import "time"

type Kind string

const (
	Began       Kind = "began"
	Prompted    Kind = "prompted"
	UseStarting Kind = "use-starting"
	UseDone     Kind = "use-done"
	Notified    Kind = "notified"
	Stopped     Kind = "stopped"
	Ended       Kind = "ended"
	// value: an app that reports no tool use finished one turn and waits.
	TurnDone Kind = "turn-done"
)

type Act string

const (
	ActEdit     Act = "edit"
	ActRead     Act = "read"
	ActRun      Act = "run"
	ActWeb      Act = "web"
	ActDelegate Act = "delegate"
	// value: a use that makes no step.
	ActOther Act = "other"
)

type Event struct {
	// value: one event an app's hook reports, read by its Hooks adapter
	// into words no app owns. Prompt, Command, Output and Message are
	// read, then dropped: Judge keeps only what Scrub lets through, a
	// command's class and counts.
	Kind    Kind   `json:"kind"`
	App     App    `json:"app"`
	Session string `json:"session"`
	// value: the terminal of the VM's terminal host the hook ran in.
	Terminal string    `json:"terminal,omitempty"`
	Folder   string    `json:"folder,omitempty"`
	At       time.Time `json:"at"`
	Act      Act       `json:"act,omitempty"`
	Path     string    `json:"path,omitempty"`
	Command  string    `json:"command,omitempty"`
	Output   string    `json:"output,omitempty"`
	// value: an edit's lines, those it adds and those it takes out.
	Added   int    `json:"added,omitempty"`
	Removed int    `json:"removed,omitempty"`
	Prompt  string `json:"prompt,omitempty"`
	Message string `json:"message,omitempty"`
	Reason  string `json:"reason,omitempty"`
	// value: the change of the agent's folder against its base, read when it
	// stops.
	Change *Change `json:"change,omitempty"`
}

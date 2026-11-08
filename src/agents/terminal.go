package agents

import "slices"

type Seen string

const (
	// value: what an agent says of itself through the multiplexer, in the
	// multiplexer's own words; empty for a program that says nothing.
	SeenWorking Seen = "working"
	SeenBlocked Seen = "blocked"
	SeenIdle    Seen = "idle"
	SeenDone    Seen = "done"
	SeenError   Seen = "error"
)

type Terminal struct {
	// value: a terminal station opened on the VM's multiplexer for an
	// agent, by its address there.
	Address string `json:"address"`
	// value: the name of the agent station started in it, from its label.
	Name   string `json:"name,omitempty"`
	App    App    `json:"app,omitempty"`
	Folder string `json:"folder,omitempty"`
	Seen   Seen   `json:"seen,omitempty"`
	// value: the program's own words with its status, the title of what it
	// waits for.
	Message string `json:"message,omitempty"`
}

func Detected(kept []Agent, live []Terminal) []Agent {
	// invariant: the hooks win: an agent the journal keeps is said as its
	// hook said it. A live terminal no kept agent runs in is an agent the
	// multiplexer alone sees: working, waiting with the program's own words, or
	// running when it says nothing; never more than the multiplexer told.
	out := slices.Clone(kept)
	for _, t := range live {
		if !PlainSession(t.Address) || slices.ContainsFunc(kept, func(a Agent) bool { return a.Terminal == t.Address || a.Session == t.Address }) {
			continue
		}
		a := Agent{Name: t.Name, App: t.App, Session: t.Address, Terminal: t.Address, Folder: t.Folder, State: Running}
		switch t.Seen {
		case SeenWorking:
			a.State = Working
		case SeenError:
		case SeenBlocked, SeenIdle, SeenDone:
			a.State = Waiting
			if title, detail := Scrub(t.Message); title != "" {
				a.Ask = &Ask{Title: title, Detail: detail}
			}
		}
		out = append(out, a)
	}
	return out
}

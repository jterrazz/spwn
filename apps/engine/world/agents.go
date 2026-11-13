package world

import (
	"fmt"
	"strconv"

	"github.com/jterrazz/studio-station/apps/engine/agents"
)

// value: the app a world's agent runs, and the folder it works in when the
// event names none.
const agentApp agents.App = "claudecode"

const agentFolder = "/home/admin/app"

// invariant: an agent event of the world is its hook's: the VM's journal
// folds it as station's hook would have written it, at the world's time.
func agentEvent(kind agents.Kind, build func(e *agents.Event, h *Agents, args []string) error) func(w *World, args []string) error {
	return func(w *World, args []string) error {
		v, err := w.vm(args[0])
		if err != nil {
			return err
		}
		if !agents.PlainSession(args[1]) {
			return fmt.Errorf("world %s: %q is no session name", kind, args[1])
		}
		held := v.Agentic()
		if held.Journal == nil {
			held.Journal = map[string]agents.Agent{}
		}
		a := held.Journal[args[1]]
		e := agents.Event{Kind: kind, App: agentApp, Session: args[1], Folder: agentFolder, At: w.Now}
		if build != nil {
			if err := build(&e, held, args[2:]); err != nil {
				return err
			}
		}
		if a.Terminal == "" {
			e.Terminal = freeTerminal(held)
		}
		held.Journal[args[1]] = agents.Judge(a, e)
		return nil
	}
}

// value: the first terminal of the multiplexer no kept agent runs in: the one a
// hook of a new session reports from.
func freeTerminal(h *Agents) string {
	for _, t := range h.Terminals {
		taken := false
		for _, a := range h.Journal {
			taken = taken || a.Terminal == t.Address
		}
		if !taken {
			return t.Address
		}
	}
	return ""
}

func agentPrompt(e *agents.Event, h *Agents, args []string) error {
	// invariant: an agent prompted runs in a terminal of the VM's multiplexer.
	e.Prompt = args[0]
	if len(args) > 1 {
		e.Folder = args[1]
	}
	if freeTerminal(h) == "" {
		h.Opened = append(h.Opened, e.Folder+" "+string(e.App))
		h.Terminals = append(h.Terminals, Terminal{Terminal: agents.Terminal{Address: "t" + strconv.Itoa(len(h.Terminals)+1), App: e.App, Folder: e.Folder}})
	}
	return nil
}

func agentStep(e *agents.Event, h *Agents, args []string) error {
	e.Kind, e.Act = agents.UseStarting, agents.Act(args[0])
	switch e.Act {
	case agents.ActEdit, agents.ActRead:
		e.Path = agentFolder + "/" + args[1]
	case agents.ActRun:
		e.Command = args[1]
	case agents.ActWeb, agents.ActDelegate, agents.ActOther:
	default:
		return fmt.Errorf("world agent-step: %q is not edit, read, run, web or delegate", args[0])
	}
	if len(args) > 2 {
		e.Output = args[2]
		if pr, ok := agents.PullRequest(e.Output); ok {
			if h.PRs == nil {
				h.PRs = map[int]string{}
			}
			h.PRs[pr.Number] = "open"
		}
	}
	return nil
}

// invariant: a step is told by its two events, its start and its end.
func agentUse(w *World, args []string) error {
	if err := agentEvent(agents.UseStarting, agentStep)(w, args); err != nil {
		return err
	}
	return agentEvent(agents.UseDone, func(e *agents.Event, _ *Agents, rest []string) error {
		if len(rest) > 2 {
			e.Output = rest[2]
		}
		return nil
	})(w, args)
}

func agentMessage(e *agents.Event, h *Agents, args []string) error {
	e.Message = args[0]
	if e.Kind == agents.Stopped {
		if c, ok := h.Changes[e.Folder]; ok {
			e.Change = &c
		}
	}
	return nil
}

func agentEnd(e *agents.Event, _ *Agents, args []string) error {
	if len(args) > 0 {
		e.Reason = args[0]
	}
	return nil
}

// invariant: an agent no hook reports runs in a terminal of the VM's
// multiplexer, which says of it what its program said, "-" for nothing.
func agentTerminal(w *World, args []string) error {
	v, err := w.vm(args[0])
	if err != nil {
		return err
	}
	seen := agents.Seen(args[1])
	if args[1] == "-" {
		seen = ""
	}
	t := Terminal{Terminal: agents.Terminal{App: agentApp, Folder: agentFolder, Seen: seen}}
	if len(args) > 2 {
		t.Message = args[2]
	}
	h := v.Agentic()
	t.Address = "t" + strconv.Itoa(len(h.Terminals)+1)
	h.Opened = append(h.Opened, agentFolder+" "+string(agentApp))
	h.Terminals = append(h.Terminals, t)
	return nil
}

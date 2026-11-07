package world

import (
	"fmt"

	"github.com/jterrazz/studio-station/src/agents"
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
		held.Journal[args[1]] = agents.Judge(a, e)
		return nil
	}
}

func agentPrompt(e *agents.Event, h *Agents, args []string) error {
	// invariant: an agent prompted runs in the VM's terminal session.
	e.Prompt = args[0]
	if len(args) > 1 {
		e.Folder = args[1]
	}
	if len(h.Opened) == 0 {
		h.Opened = append(h.Opened, e.Folder+" "+string(e.App))
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

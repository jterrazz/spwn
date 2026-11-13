package agents

import (
	"strconv"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/agents"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/page"
	"github.com/jterrazz/studio-station/apps/engine/pages"
)

func columns(everywhere bool) []page.Column {
	cols := []page.Column{
		{Key: "agent", Kind: page.KindText, Keep: 100},
		{Key: "app", Kind: page.KindText, Keep: 40},
		{Key: "state", Kind: page.KindList, Keep: 90},
		{Key: "task", Kind: page.KindText, Keep: 60},
	}
	if everywhere {
		cols = append(cols, page.Column{Key: "vm", Kind: page.KindText, Keep: 70}, page.Column{Key: "mac", Kind: page.KindText, Keep: 50}, page.Column{Key: "since", Kind: page.KindAge, Keep: 30})
	}
	return cols
}

func List(kept []agents.Agent, thisMachine page.Context) page.Page {
	// invariant: SPEC B5.1: each agent of the VM as its hook said it: the
	// waiting ones first, with what they wait on, then the working ones with
	// their step, then the finished.
	return list(kept, thisMachine, false, time.Time{})
}

func AllMachines(kept []agents.Agent, thisMachine page.Context, now time.Time) page.Page {
	// invariant: SPEC B5.22: every agent of every machine, in the same
	// order, each with its VM, its Mac and since when.
	return list(kept, thisMachine, true, now)
}

func list(kept []agents.Agent, thisMachine page.Context, everywhere bool, now time.Time) page.Page {
	items := []page.Item{}
	for _, state := range []agents.State{agents.Waiting, agents.Working, agents.Starting, agents.Running, agents.Finished} {
		for _, a := range kept {
			if a.State == state {
				r := Row(a)
				if everywhere {
					since := page.None()
					if !a.Since.IsZero() {
						since = page.Age(a.Since, now)
					}
					r.Cells = append(r.Cells, page.Text(a.VM), page.Text(a.Mac), since)
				}
				items = append(items, page.Item{Row: &r})
			}
		}
	}
	if len(items) == 0 {
		items = append(items, page.Item{Empty: &page.Empty{Noun: "agents"}})
	}
	return page.Seal(page.Page{ID: "agents", Command: "agents", Context: thisMachine,
		Sections: []page.Section{{ID: "agents", Title: "Agents", Columns: columns(everywhere), Items: items, PipeCounts: true}}})
}

func State(a agents.Agent) page.Value {
	// value: what an agent is at, in the words of every surface: what it
	// waits on, the step it does, how it finished.
	switch a.State {
	case agents.Waiting:
		if a.Ask != nil && a.Ask.Title != "" {
			return page.Texts("waiting", a.Ask.Title)
		}
	case agents.Working:
		return page.Texts(append([]string{"working"}, agents.Words(a)...)...)
	case agents.Finished:
		if a.Reason != "" {
			return page.Texts("finished", a.Reason)
		}
	case agents.Starting, agents.Running:
	}
	return page.Text(string(a.State))
}

func markOf(s agents.State) page.Mark {
	switch s {
	case agents.Waiting:
		return page.Attention
	case agents.Working, agents.Running, agents.Starting:
		return page.OK
	case agents.Finished:
	}
	return page.Absent
}

func Row(a agents.Agent) page.Row {
	r := page.Row{ID: a.Name, Label: a.Name, Mark: markOf(a.State),
		Cells: []page.Value{page.Text(string(a.App)), State(a), page.Text(a.Task)}}
	if a.PR != nil {
		r.Detail = page.Texts("PR #"+strconv.Itoa(a.PR.Number), a.PR.URL)
	}
	if a.State == agents.Waiting {
		r.Reason = page.Partial
		r.Hint = page.Cmd("station agents reply " + a.Name + " <text>")
		if a.PR != nil {
			r.Hint = page.Cmd("station agents merge " + a.Name)
		}
	}
	return r
}

func Replied(a agents.Agent, thisMachine page.Context) page.Page {
	r := pages.Done(a.Name, catalog.StateOK, "")
	r.Detail = page.Texts("sent", "it works on it")
	return pages.Outcome("agents reply", thisMachine, nil, []page.Row{r}, "agents")
}

func Merged(a agents.Agent, thisMachine page.Context) page.Page {
	r := pages.Done(a.Name, catalog.StateOK, "finished")
	if a.Reason != "" {
		r.Detail = page.Texts(a.Reason, "finished")
	}
	return pages.Outcome("agents merge", thisMachine, nil, []page.Row{r}, "agents")
}

func Started(a agents.Agent, onNew bool, thisMachine page.Context) page.Page {
	// invariant: an agent started is said by its name, the ephemeral workspace made
	// for it, and its app, then its task, or that it waits for a first
	// prompt.
	said := []string{"starts", string(a.App)}
	if onNew {
		said[0] = "starts on " + a.VM + ", a new ephemeral workspace"
	}
	if a.Folder != "" {
		said = append(said, "in "+a.Folder)
	}
	if a.Task != "" {
		said = append(said, a.Task)
	} else {
		said = append(said, "waits for a prompt")
	}
	r := pages.Done(a.Name, catalog.StateOK, "")
	r.Detail = page.Texts(said...)
	return pages.Outcome("agents start", thisMachine, nil, []page.Row{r}, "agents")
}

func Stopped(a agents.Agent, thisMachine page.Context) page.Page {
	r := pages.Done(a.Name, catalog.StateOK, "")
	r.Detail = page.Texts("stopped", "its workspace keeps running")
	return pages.Outcome("agents stop", thisMachine, nil, []page.Row{r}, "agents")
}

func Forgotten(a agents.Agent, thisMachine page.Context) page.Page {
	r := pages.Done(a.Name, catalog.StateOK, "")
	r.Detail = page.Texts("forgotten", "its workspace stays")
	return pages.Outcome("agents forget", thisMachine, nil, []page.Row{r}, "agents")
}

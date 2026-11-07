package agents

import (
	"strconv"

	"github.com/jterrazz/studio-station/src/agents"
	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/page"
	"github.com/jterrazz/studio-station/src/pages"
)

func columns() []page.Column {
	return []page.Column{
		{Key: "session", Kind: page.KindText, Keep: 100},
		{Key: "app", Kind: page.KindText, Keep: 40},
		{Key: "state", Kind: page.KindList, Keep: 90},
		{Key: "task", Kind: page.KindText, Keep: 60},
	}
}

func List(kept []agents.Agent, thisMachine page.Context) page.Page {
	// invariant: SPEC B5.1: each agent of the VM as its hook said it: the
	// waiting ones first, with what they wait on, then the working ones with
	// their step, then the finished.
	items := []page.Item{}
	for _, state := range []agents.State{agents.Waiting, agents.Working, agents.Starting, agents.Running, agents.Finished} {
		for _, a := range kept {
			if a.State == state {
				r := Row(a)
				items = append(items, page.Item{Row: &r})
			}
		}
	}
	if len(items) == 0 {
		items = append(items, page.Item{Empty: &page.Empty{Noun: "agents"}})
	}
	return page.Seal(page.Page{ID: "agents", Command: "agents", Context: thisMachine,
		Sections: []page.Section{{ID: "agents", Title: "Agents", Columns: columns(), Items: items, PipeCounts: true}}})
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
	case agents.Working, agents.Running:
		return page.OK
	case agents.Starting:
		return page.Awaiting
	case agents.Finished:
	}
	return page.Absent
}

func Row(a agents.Agent) page.Row {
	r := page.Row{ID: a.Session, Label: a.Session, Mark: markOf(a.State),
		Cells: []page.Value{page.Text(string(a.App)), State(a), page.Text(a.Task)}}
	if a.PR != nil {
		r.Detail = page.Texts("PR #"+strconv.Itoa(a.PR.Number), a.PR.URL)
	}
	if a.State == agents.Waiting {
		r.Reason = page.Partial
		r.Hint = page.Cmd("station agents reply " + a.Session + " <text>")
		if a.PR != nil {
			r.Hint = page.Cmd("station agents merge-pr " + a.Session)
		}
	}
	return r
}

func Replied(a agents.Agent, thisMachine page.Context) page.Page {
	r := pages.Done(a.Session, catalog.StateOK, "")
	r.Detail = page.Texts("sent", "it works on it")
	return pages.Outcome("agents reply", thisMachine, nil, []page.Row{r}, "agents")
}

func Merged(a agents.Agent, thisMachine page.Context) page.Page {
	r := pages.Done(a.Session, catalog.StateOK, "finished")
	if a.Reason != "" {
		r.Detail = page.Texts(a.Reason, "finished")
	}
	return pages.Outcome("agents merge-pr", thisMachine, nil, []page.Row{r}, "agents")
}

func Started(app agents.App, folder, task string, thisMachine page.Context) page.Page {
	said := "started"
	if folder != "" {
		said += " in " + folder
	}
	r := pages.Done(string(app), catalog.StateOK, said)
	if task != "" {
		r.Detail = page.Texts(said, task)
	}
	return pages.Outcome("agents start", thisMachine, nil, []page.Row{r}, "agents")
}

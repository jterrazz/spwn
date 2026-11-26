package automations

import (
	"strconv"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/automations"
	"github.com/jterrazz/studio-station/apps/engine/page"
)

func columns() []page.Column {
	return []page.Column{
		{Key: "name", Kind: page.KindText, Keep: 100},
		{Key: "agent", Label: "agent", Kind: page.KindText, Keep: 70},
		{Key: "trigger", Label: "trigger", Kind: page.KindText, Keep: 60},
		{Key: "last-fired", Label: "last fired", Kind: page.KindText, Keep: 40},
		{Key: "runs", Label: "runs", Kind: page.KindText, Keep: 30},
	}
}

func trigger(t automations.Trigger) string {
	if t.Kind == "" {
		return "none"
	}
	return t.Kind + " " + t.Spec
}

func lastFired(a automations.Automation) string {
	if a.LastFired.IsZero() {
		return "never"
	}
	return a.LastFired.UTC().Format(time.DateTime)
}

func runs(a automations.Automation) string {
	switch {
	case a.Fires == 0:
		return "none"
	case a.Failed == 0:
		return strconv.Itoa(a.Fires)
	}
	return strconv.Itoa(a.Fires) + ", " + strconv.Itoa(a.Failed) + " failed"
}

func row(a automations.Automation) page.Row {
	r := page.Row{ID: a.Name, Label: a.Name, Mark: page.OK,
		Cells: []page.Value{page.Text(a.Agent), page.Text(trigger(a.Trigger)), page.Text(lastFired(a)), page.Text(runs(a))}}
	if automations.Reserved(a.Name) {
		r.Mark = page.Attention
		r.Detail = page.Text("its name is a command's: station automations log " + a.Name + " reads its runs, and no detail can be asked for it")
	}
	return r
}

func List(all []automations.Automation, thisMac page.Context) page.Page {
	// invariant: one row per automation by name; a project with none says
	// where one is declared.
	s := page.Section{ID: "automations", Title: "Automations", Columns: columns()}
	for _, a := range all {
		r := row(a)
		s.Items = append(s.Items, page.Item{Row: &r})
	}
	if len(all) == 0 {
		s.Items = []page.Item{{Empty: &page.Empty{Text: "No automation declared: docs/guides/automations.md says how one is."}}}
	}
	return page.Seal(page.Page{ID: "automations", Command: "automations", Context: thisMac, Sections: []page.Section{s}})
}

func Show(a automations.Automation, thisMac page.Context) page.Page {
	// invariant: the automation alone, what its runs say of it on its row;
	// its runs are those of `automations log <automation>`.
	r := row(a)
	head := page.Section{ID: a.Name, Title: a.Name, Role: page.Banner, Uncounted: true, Columns: columns(), Items: []page.Item{{Row: &r}}}
	return page.Seal(page.Page{ID: "automations-show", Command: "automations", Context: thisMac, Sections: []page.Section{head}})
}

func Log(all []automations.Run, thisMac page.Context) page.Page {
	// invariant: one row per run, oldest first, a failed one marked broken
	// with its error; none says where a run is written.
	s := page.Section{ID: "runs", Title: "Runs", Columns: []page.Column{
		{Key: "at", Kind: page.KindText, Keep: 100},
		{Key: "automation", Label: "automation", Kind: page.KindText, Keep: 70},
		{Key: "outcome", Label: "outcome", Kind: page.KindText, Keep: 60},
	}}
	for i, run := range all {
		mark, outcome := page.OK, "ok"
		if !run.OK {
			mark, outcome = page.Broken, "failed: "+oneLine(run.Error)
		}
		r := page.Row{ID: strconv.Itoa(i), Label: run.At.UTC().Format(time.DateTime), Mark: mark,
			Cells: []page.Value{page.Text(run.Name), page.Text(outcome)}, Detail: page.Text(oneLine(run.Output))}
		s.Items = append(s.Items, page.Item{Row: &r})
	}
	if len(all) == 0 {
		s.Items = []page.Item{{Empty: &page.Empty{Text: "No run yet: each run of an automation is written to .spwn/runs.jsonl of the project."}}}
	}
	return page.Seal(page.Page{ID: "automations-log", Command: "automations log", Context: thisMac, Sections: []page.Section{s}})
}

func oneLine(s string) string {
	first, _, _ := strings.Cut(strings.TrimSpace(s), "\n")
	return first
}

type TriggerJSON struct {
	Kind string `json:"kind"`
	Spec string `json:"spec"`
}

type AutomationJSON struct {
	Name      string      `json:"name"`
	Agent     string      `json:"agent"`
	Trigger   TriggerJSON `json:"trigger"`
	LastFired *time.Time  `json:"last_fired"`
	Fires     int         `json:"fires"`
	Failed    int         `json:"failed"`
}

type RunJSON struct {
	Automation string     `json:"automation"`
	Scheduled  *time.Time `json:"scheduled"`
	At         time.Time  `json:"at"`
	OK         bool       `json:"ok"`
	Error      string     `json:"error"`
	Output     string     `json:"output"`
}

type ListJSON struct {
	Schema      string           `json:"schema"`
	Automations []AutomationJSON `json:"automations"`
}

type ShowJSON struct {
	Schema string `json:"schema"`
	AutomationJSON
}

type LogJSON struct {
	Schema string    `json:"schema"`
	Runs   []RunJSON `json:"runs"`
}

func instant(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	utc := t.UTC()
	return &utc
}

func automationJSON(a automations.Automation) AutomationJSON {
	return AutomationJSON{Name: a.Name, Agent: a.Agent, Trigger: TriggerJSON{Kind: a.Trigger.Kind, Spec: a.Trigger.Spec},
		LastFired: instant(a.LastFired), Fires: a.Fires, Failed: a.Failed}
}

func ListContract(schema string, all []automations.Automation) ListJSON {
	out := ListJSON{Schema: schema, Automations: make([]AutomationJSON, 0, len(all))}
	for _, a := range all {
		out.Automations = append(out.Automations, automationJSON(a))
	}
	return out
}

func ShowContract(schema string, a automations.Automation) ShowJSON {
	return ShowJSON{Schema: schema, AutomationJSON: automationJSON(a)}
}

func LogContract(schema string, all []automations.Run) LogJSON {
	out := LogJSON{Schema: schema, Runs: make([]RunJSON, 0, len(all))}
	for _, r := range all {
		out.Runs = append(out.Runs, RunJSON{Automation: r.Name, Scheduled: instant(r.Scheduled), At: r.At.UTC(), OK: r.OK, Error: r.Error, Output: r.Output})
	}
	return out
}

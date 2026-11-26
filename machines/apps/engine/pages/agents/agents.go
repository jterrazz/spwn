package agents

import (
	"strconv"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/agents"
	"github.com/jterrazz/studio-station/apps/engine/page"
)

func columns() []page.Column {
	return []page.Column{
		{Key: "name", Kind: page.KindText, Keep: 100},
		{Key: "team", Label: "team", Kind: page.KindText, Keep: 50},
		{Key: "harness", Label: "harness", Kind: page.KindText, Keep: 60},
	}
}

func row(a agents.Agent) page.Row {
	return page.Row{ID: a.Name, Label: a.Name, Mark: page.OK, Detail: page.Text(a.Description),
		Cells: []page.Value{page.Text(a.Team), page.Text(a.Harness)}}
}

func List(all []agents.Agent, thisMac page.Context) page.Page {
	// invariant: one row per agent by name; a project with none says how
	// one is made.
	s := page.Section{ID: "agents", Title: "Agents", Columns: columns()}
	for _, a := range all {
		r := row(a)
		s.Items = append(s.Items, page.Item{Row: &r})
	}
	if len(all) == 0 {
		s.Items = []page.Item{{Empty: &page.Empty{Text: "No agent yet: station agents create <name> makes one."}}}
	}
	return page.Seal(page.Page{ID: "agents", Command: "agents", Context: thisMac, Sections: []page.Section{s}})
}

func Show(m agents.Mind, thisMac page.Context) page.Page {
	// invariant: the agent, then its layers and their files, then its
	// newest journal entries.
	r := row(m.Agent)
	r.Cells = append(r.Cells, page.Text(m.Path))
	head := page.Section{ID: m.Name, Title: m.Name, Role: page.Banner, Uncounted: true,
		Columns: append(columns(), page.Column{Key: "path", Kind: page.KindText, Keep: 40}), Items: []page.Item{{Row: &r}}}
	layers := page.Section{ID: "layers", Title: "Layers", Columns: []page.Column{{Key: "name", Kind: page.KindText, Keep: 100}, {Key: "files", Label: "files", Kind: page.KindText, Keep: 60}}}
	for _, l := range m.Layers {
		lr := page.Row{ID: l.Name, Label: l.Name, Mark: page.OK, Cells: []page.Value{page.Text(files(l.Files))}}
		layers.Items = append(layers.Items, page.Item{Row: &lr})
	}
	sections := []page.Section{head, layers}
	if len(m.Journal) > 0 {
		j := page.Section{ID: "journal", Title: "Journal", Columns: []page.Column{{Key: "when", Kind: page.KindText, Keep: 100}, {Key: "outcome", Label: "outcome", Kind: page.KindText, Keep: 60}, {Key: "took", Label: "took", Kind: page.KindText, Keep: 40}}}
		for i, e := range m.Journal {
			mark := page.OK
			if e.Code != 0 {
				mark = page.Broken
			}
			jr := page.Row{ID: strconv.Itoa(i), Label: e.At.UTC().Format(time.DateTime), Mark: mark, Cells: []page.Value{page.Text(e.Outcome), page.Text(e.Took.String())}}
			j.Items = append(j.Items, page.Item{Row: &jr})
		}
		sections = append(sections, j)
	}
	return page.Seal(page.Page{ID: "agents-show", Command: "agents", Context: thisMac, Sections: sections})
}

func files(all []string) string {
	if len(all) == 0 {
		return "empty"
	}
	return strings.Join(all, ", ")
}

func Done(id page.ID, command, name, outcome, where string, thisMac page.Context) page.Page {
	// value: the one banner every act of an agent answers with: the agent,
	// what was done, and what it left where.
	r := page.Row{ID: name, Label: name, Mark: page.OK, Cells: []page.Value{page.Text(outcome), page.Text(where)}}
	cols := []page.Column{{Key: "name", Kind: page.KindText, Keep: 100}, {Key: "outcome", Kind: page.KindText, Keep: 90}, {Key: "where", Kind: page.KindText, Keep: 40}}
	return page.Seal(page.Page{ID: id, Command: command, Context: thisMac,
		Sections: []page.Section{{ID: string(id), Title: "Done", Role: page.Banner, Uncounted: true, Columns: cols, Items: []page.Item{{Row: &r}}}}})
}

func Dreamed(d agents.Dreamed, thisMac page.Context) page.Page {
	if d.Skipped != "" {
		return Done("agents-dream", "agents dream", d.Agent, "no dream: "+d.Skipped, "", thisMac)
	}
	said := strconv.Itoa(d.Entries) + " entries, " + strconv.Itoa(int(d.Rate*100+0.5)) + " % completed"
	return Done("agents-dream", "agents dream", d.Agent, said, d.Written, thisMac)
}

func Slept(s agents.Slept, thisMac page.Context) page.Page {
	said := strconv.Itoa(s.Archived) + " playbooks archived, " + strconv.Itoa(s.Pruned) + " sessions pruned"
	return Done("agents-sleep", "agents sleep", s.Agent, said, "", thisMac)
}

type AgentJSON struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Team        string `json:"team"`
	Harness     string `json:"harness"`
}

type LayerJSON struct {
	Name  string   `json:"name"`
	Files []string `json:"files"`
}

type EntryJSON struct {
	At      time.Time `json:"at"`
	Outcome string    `json:"outcome"`
	Code    int       `json:"code"`
	TookMS  int64     `json:"took_ms"`
}

type ListJSON struct {
	Schema string      `json:"schema"`
	Agents []AgentJSON `json:"agents"`
}

type ShowJSON struct {
	Schema string `json:"schema"`
	AgentJSON
	Path    string      `json:"path"`
	Layers  []LayerJSON `json:"layers"`
	Journal []EntryJSON `json:"journal"`
}

func agentJSON(a agents.Agent) AgentJSON {
	return AgentJSON{Name: a.Name, Description: a.Description, Team: a.Team, Harness: a.Harness}
}

func ListContract(schema string, all []agents.Agent) ListJSON {
	out := ListJSON{Schema: schema, Agents: make([]AgentJSON, 0, len(all))}
	for _, a := range all {
		out.Agents = append(out.Agents, agentJSON(a))
	}
	return out
}

func ShowContract(schema string, m agents.Mind) ShowJSON {
	out := ShowJSON{Schema: schema, AgentJSON: agentJSON(m.Agent), Path: m.Path, Layers: make([]LayerJSON, 0, len(m.Layers)), Journal: make([]EntryJSON, 0, len(m.Journal))}
	for _, l := range m.Layers {
		files := l.Files
		if files == nil {
			files = []string{}
		}
		out.Layers = append(out.Layers, LayerJSON{Name: l.Name, Files: files})
	}
	for _, e := range m.Journal {
		out.Journal = append(out.Journal, EntryJSON{At: e.At, Outcome: e.Outcome, Code: e.Code, TookMS: e.Took.Milliseconds()})
	}
	return out
}

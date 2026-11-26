package project

import (
	"strconv"

	"github.com/jterrazz/studio-station/apps/engine/page"
	"github.com/jterrazz/studio-station/apps/engine/project"
)

func banner(id, label string, mark page.Mark, cells ...string) page.Section {
	r := page.Row{ID: label, Label: label, Mark: mark}
	cols := make([]page.Column, 0, 1+len(cells))
	cols = append(cols, page.Column{Key: "name", Kind: page.KindText, Keep: 100})
	for i, c := range cells {
		r.Cells = append(r.Cells, page.Text(c))
		cols = append(cols, page.Column{Key: "cell" + strconv.Itoa(i), Kind: page.KindText, Keep: 40})
	}
	return page.Section{ID: id, Title: label, Role: page.Banner, Uncounted: true, Columns: cols, Items: []page.Item{{Row: &r}}}
}

func Show(p project.Project, thisMac page.Context) page.Page {
	// invariant: the project and the file it is, then its agents by name; a
	// project with none says how one is made.
	agents := page.Section{ID: "agents", Title: "Agents", Columns: []page.Column{{Key: "name", Kind: page.KindText, Keep: 100}}}
	for _, a := range p.Agents {
		r := page.Row{ID: a, Label: a, Mark: page.OK}
		agents.Items = append(agents.Items, page.Item{Row: &r})
	}
	if len(p.Agents) == 0 {
		agents.Items = []page.Item{{Empty: &page.Empty{Text: "No agent yet: station agents create <name> makes one."}}}
	}
	return page.Seal(page.Page{ID: "project", Command: "project", Context: thisMac, Sections: []page.Section{banner(p.Name, p.Name, page.OK, p.Manifest), agents}})
}

func markOf(l project.Level) page.Mark {
	switch l {
	case project.Error:
		return page.Broken
	case project.Warning:
		return page.Attention
	case project.Info:
		return page.Unjudged
	}
	return page.Unjudged
}

func issues(id, title, none string, all []project.Issue) page.Section {
	s := page.Section{ID: id, Title: title, Columns: []page.Column{{Key: "path", Kind: page.KindText, Keep: 100}}}
	for i, issue := range all {
		r := page.Row{ID: strconv.Itoa(i), Label: issue.Path, Mark: markOf(issue.Level), Detail: page.Text(issue.Message)}
		if issue.Hint != "" {
			r.Hint = &page.Hint{Text: issue.Hint}
		}
		s.Items = append(s.Items, page.Item{Row: &r})
	}
	if len(all) == 0 {
		s.Items = []page.Item{{Empty: &page.Empty{Text: none}}}
	}
	return s
}

func verdict(p project.Plan) (mark page.Mark, said string) {
	n := func(count int, one, many string) string {
		if count == 1 {
			return "1 " + one
		}
		return strconv.Itoa(count) + " " + many
	}
	counts := n(p.Summary.Errors, "error", "errors") + ", " + n(p.Summary.Warnings, "warning", "warnings")
	if !p.Valid {
		return page.Broken, counts
	}
	if p.Summary.Warnings > 0 {
		return page.Attention, "valid, " + counts
	}
	return page.OK, "valid"
}

func Plan(p project.Plan, thisMac page.Context) page.Page {
	// invariant: the verdict first, then each issue, the errors first.
	mark, said := verdict(p)
	return page.Seal(page.Page{ID: "project-plan", Command: "project plan", Context: thisMac, Sections: []page.Section{
		banner("plan", p.Manifest, mark, said),
		issues("issues", "Issues", "Nothing to fix: the project reads as it is declared.", p.Issues),
	}})
}

func Drift(d project.Drift, thisMac page.Context) page.Page {
	// invariant: one row per way the project drifted; none says so.
	mark, said := page.OK, "as declared"
	if len(d.Rows) > 0 {
		mark, said = page.Attention, strconv.Itoa(len(d.Rows))+" drifted"
	}
	return page.Seal(page.Page{ID: "project-drift", Command: "project drift", Context: thisMac, Sections: []page.Section{
		banner("drift", d.Manifest, mark, said),
		issues("rows", "Drift", "Nothing drifted: the lockfile records every reference.", d.Rows),
	}})
}

func Created(p project.Project, thisMac page.Context) page.Page {
	// value: the banner a create answers with: the project, and the folder
	// it was made in.
	return page.Seal(page.Page{ID: "project-create", Command: "project create", Context: thisMac,
		Sections: []page.Section{banner("project-create", p.Name, page.OK, "created", p.Root)}})
}

type ProjectJSON struct {
	Schema   string   `json:"schema"`
	Name     string   `json:"name"`
	Root     string   `json:"root"`
	Manifest string   `json:"manifest"`
	Agents   []string `json:"agents"`
}

func ShowContract(schema string, p project.Project) ProjectJSON {
	agents := p.Agents
	if agents == nil {
		agents = []string{}
	}
	return ProjectJSON{Schema: schema, Name: p.Name, Root: p.Root, Manifest: p.Manifest, Agents: agents}
}

type IssueJSON struct {
	Level   string `json:"level"`
	Path    string `json:"path"`
	Message string `json:"message"`
	Hint    string `json:"hint,omitempty"`
	Source  string `json:"source"`
}

type SummaryJSON struct {
	Errors   int `json:"errors"`
	Warnings int `json:"warnings"`
	Info     int `json:"info"`
}

type PlanJSON struct {
	Schema   string      `json:"schema"`
	Valid    bool        `json:"valid"`
	Manifest string      `json:"manifest"`
	Summary  SummaryJSON `json:"summary"`
	Issues   []IssueJSON `json:"issues"`
}

type DriftJSON struct {
	Schema   string      `json:"schema"`
	Manifest string      `json:"manifest"`
	Rows     []IssueJSON `json:"rows"`
}

func issuesJSON(all []project.Issue) []IssueJSON {
	out := make([]IssueJSON, 0, len(all))
	for _, i := range all {
		out = append(out, IssueJSON{Level: string(i.Level), Path: i.Path, Message: i.Message, Hint: i.Hint, Source: i.Source})
	}
	return out
}

func PlanContract(schema string, p project.Plan) PlanJSON {
	return PlanJSON{Schema: schema, Valid: p.Valid, Manifest: p.Manifest,
		Summary: SummaryJSON{Errors: p.Summary.Errors, Warnings: p.Summary.Warnings, Info: p.Summary.Info}, Issues: issuesJSON(p.Issues)}
}

func DriftContract(schema string, d project.Drift) DriftJSON {
	return DriftJSON{Schema: schema, Manifest: d.Manifest, Rows: issuesJSON(d.Rows)}
}

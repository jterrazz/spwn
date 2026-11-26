package teams

import (
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/page"
	"github.com/jterrazz/studio-station/apps/engine/teams"
)

func columns() []page.Column {
	return []page.Column{
		{Key: "slug", Kind: page.KindText, Keep: 100},
		{Key: "name", Label: "name", Kind: page.KindText, Keep: 60},
		{Key: "agents", Label: "agents", Kind: page.KindText, Keep: 50},
	}
}

func members(t teams.Team) string {
	if len(t.Members) == 0 {
		return "none"
	}
	return strings.Join(t.Members, ", ")
}

func row(t teams.Team) page.Row {
	return page.Row{ID: t.Slug, Label: t.Slug, Mark: page.OK, Detail: page.Text(t.Description),
		Cells: []page.Value{page.Text(t.Name), page.Text(members(t))}}
}

func List(all []teams.Team, thisMac page.Context) page.Page {
	// invariant: one row per team by slug; none declared says where a team
	// is declared.
	s := page.Section{ID: "teams", Title: "Teams", Columns: columns()}
	for _, t := range all {
		r := row(t)
		s.Items = append(s.Items, page.Item{Row: &r})
	}
	if len(all) == 0 {
		s.Items = []page.Item{{Empty: &page.Empty{Text: "No team declared: a team is a file, ~/.spwn/teams/<team>.yaml."}}}
	}
	return page.Seal(page.Page{ID: "teams", Command: "teams", Context: thisMac, Sections: []page.Section{s}})
}

func Show(t teams.Team, thisMac page.Context) page.Page {
	// invariant: the team, then each agent of the project that names it.
	r := row(t)
	head := page.Section{ID: t.Slug, Title: t.Slug, Role: page.Banner, Uncounted: true, Columns: columns(), Items: []page.Item{{Row: &r}}}
	people := page.Section{ID: "members", Title: "Members", Columns: []page.Column{{Key: "name", Kind: page.KindText, Keep: 100}}}
	for _, name := range t.Members {
		mr := page.Row{ID: name, Label: name, Mark: page.OK}
		people.Items = append(people.Items, page.Item{Row: &mr})
	}
	if len(t.Members) == 0 {
		people.Items = []page.Item{{Empty: &page.Empty{Text: "No agent names this team: station agents edit <agent> --team " + t.Slug + " puts one in it."}}}
	}
	return page.Seal(page.Page{ID: "teams-show", Command: "teams", Context: thisMac, Sections: []page.Section{head, people}})
}

type TeamJSON struct {
	Slug        string   `json:"slug"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Members     []string `json:"members"`
}

type ListJSON struct {
	Schema string     `json:"schema"`
	Teams  []TeamJSON `json:"teams"`
}

type ShowJSON struct {
	Schema string `json:"schema"`
	TeamJSON
}

func teamJSON(t teams.Team) TeamJSON {
	members := t.Members
	if members == nil {
		members = []string{}
	}
	return TeamJSON{Slug: t.Slug, Name: t.Name, Description: t.Description, Members: members}
}

func ListContract(schema string, all []teams.Team) ListJSON {
	out := ListJSON{Schema: schema, Teams: make([]TeamJSON, 0, len(all))}
	for _, t := range all {
		out.Teams = append(out.Teams, teamJSON(t))
	}
	return out
}

func ShowContract(schema string, t teams.Team) ShowJSON {
	return ShowJSON{Schema: schema, TeamJSON: teamJSON(t)}
}

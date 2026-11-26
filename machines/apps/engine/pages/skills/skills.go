package skills

import (
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/page"
	"github.com/jterrazz/studio-station/apps/engine/skills"
)

func columns() []page.Column {
	return []page.Column{{Key: "name", Kind: page.KindText, Keep: 100}}
}

func row(s skills.Skill) page.Row {
	return page.Row{ID: s.Name, Label: s.Name, Mark: page.OK, Detail: page.Text(s.Description)}
}

func List(all []skills.Skill, thisMac page.Context) page.Page {
	// invariant: one row per skill by name; a project with none says how
	// one is made.
	s := page.Section{ID: "skills", Title: "Skills", Columns: columns()}
	for _, k := range all {
		r := row(k)
		s.Items = append(s.Items, page.Item{Row: &r})
	}
	if len(all) == 0 {
		s.Items = []page.Item{{Empty: &page.Empty{Text: "No skill yet: station skills create <name> makes one."}}}
	}
	return page.Seal(page.Page{ID: "skills", Command: "skills", Context: thisMac, Sections: []page.Section{s}})
}

func Show(k skills.Skill, thisMac page.Context) page.Page {
	// invariant: the skill and the file it is, then its text as it is
	// written, line by line.
	r := row(k)
	r.Cells = []page.Value{page.Text(k.Path)}
	cols := append(columns(), page.Column{Key: "path", Kind: page.KindText, Keep: 40})
	head := page.Section{ID: k.Name, Title: k.Name, Role: page.Banner, Uncounted: true, Columns: cols, Items: []page.Item{{Row: &r}}}
	text := page.Section{ID: "text", Title: "Text"}
	for _, line := range strings.Split(strings.TrimSuffix(k.Body, "\n"), "\n") {
		text.Items = append(text.Items, page.Item{Note: &page.Note{Text: line}})
	}
	return page.Seal(page.Page{ID: "skills-show", Command: "skills", Context: thisMac, Sections: []page.Section{head, text}})
}

func Done(id page.ID, command, name, outcome, where string, thisMac page.Context) page.Page {
	// value: the one banner every act of a skill answers with: the skill,
	// what was done, and the file it left or took.
	r := page.Row{ID: name, Label: name, Mark: page.OK, Cells: []page.Value{page.Text(outcome), page.Text(where)}}
	cols := []page.Column{{Key: "name", Kind: page.KindText, Keep: 100}, {Key: "outcome", Kind: page.KindText, Keep: 90}, {Key: "where", Kind: page.KindText, Keep: 40}}
	return page.Seal(page.Page{ID: id, Command: command, Context: thisMac,
		Sections: []page.Section{{ID: string(id), Title: "Done", Role: page.Banner, Uncounted: true, Columns: cols, Items: []page.Item{{Row: &r}}}}})
}

type SkillJSON struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Description string `json:"description"`
}

type ListJSON struct {
	Schema string      `json:"schema"`
	Skills []SkillJSON `json:"skills"`
}

type ShowJSON struct {
	Schema string `json:"schema"`
	SkillJSON
	Body string `json:"body"`
}

func skillJSON(s skills.Skill) SkillJSON {
	return SkillJSON{Name: s.Name, Path: s.Path, Description: s.Description}
}

func ListContract(schema string, all []skills.Skill) ListJSON {
	out := ListJSON{Schema: schema, Skills: make([]SkillJSON, 0, len(all))}
	for _, s := range all {
		out.Skills = append(out.Skills, skillJSON(s))
	}
	return out
}

func ShowContract(schema string, s skills.Skill) ShowJSON {
	return ShowJSON{Schema: schema, SkillJSON: skillJSON(s), Body: s.Body}
}

package pages

import (
	pageskills "github.com/jterrazz/studio-station/apps/engine/pages/skills"
	"github.com/jterrazz/studio-station/apps/engine/skills"
)

// value: the skill files of a project on This Mac: one with its description,
// one without frontmatter.
func skillList() []skills.Skill {
	return []skills.Skill{
		{Name: "paper-reading", Path: "~/Developer/site/skills/paper-reading.md", Description: "How to read a paper and keep its claims."},
		{Name: "refactoring", Path: "~/Developer/site/skills/refactoring.md"},
	}
}

func skillPages() []Fixture {
	return states("skills",
		pending(pageskills.List(skillList(), thisMacContext())),
		pageskills.List(nil, thisMacContext()),
		failure("skills", "skills", "the skills of ~/Developer/site cannot be read", "station skills"),
		pageskills.List(skillList(), thisMacContext()),
	)
}

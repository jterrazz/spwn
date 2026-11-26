package pages

import (
	pageproject "github.com/jterrazz/studio-station/apps/engine/pages/project"
	"github.com/jterrazz/studio-station/apps/engine/project"
)

// value: a project on This Mac with two agents, and the same project before
// its first agent.
func siteProject(agents ...string) project.Project {
	return project.Project{Name: "site", Root: "~/Developer/site", Manifest: "~/Developer/site/spwn.yaml", Agents: agents}
}

func projectPages() []Fixture {
	return states("project",
		pending(pageproject.Show(siteProject("neo", "trinity"), thisMacContext())),
		pageproject.Show(siteProject(), thisMacContext()),
		failure("project", "project", "the project of ~/Developer/site does not read", "station project plan"),
		pageproject.Show(siteProject("neo", "trinity"), thisMacContext()),
	)
}

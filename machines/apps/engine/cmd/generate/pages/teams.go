package pages

import (
	pageteams "github.com/jterrazz/studio-station/apps/engine/pages/teams"
	"github.com/jterrazz/studio-station/apps/engine/teams"
)

// value: the teams declared for the user on This Mac: one with two agents of
// the project, one with none.
func teamList() []teams.Team {
	return []teams.Team{
		{Slug: "core", Name: "Core", Description: "Keeps the platform standing."},
		{Slug: "web", Name: "Web Team", Description: "Ships the site.", Members: []string{"neo", "trinity"}},
	}
}

func teamPages() []Fixture {
	return states("teams",
		pending(pageteams.List(teamList(), thisMacContext())),
		pageteams.List(nil, thisMacContext()),
		failure("teams", "teams", "the teams of ~/.spwn cannot be read", "station teams"),
		pageteams.List(teamList(), thisMacContext()),
	)
}

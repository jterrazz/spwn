package teams

import "errors"

type Team struct {
	// value: the file name of the team, which `teams <team>` takes.
	Slug string
	// value: how the team is called, as its file says it.
	Name        string
	Description string
	// value: the agents of the asked project whose agent.yaml names this
	// team, by name.
	Members []string
}

var ErrNotFound = errors.New("no team of that name")

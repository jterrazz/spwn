package catalog

type AgentApp struct {
	// value: the app's id as its hook adapter names it, and its name as
	// the owner knows it.
	ID, Name string
}

var AgentApps = []AgentApp{{ID: "claudecode", Name: "Claude"}, {ID: "codex", Name: "Codex"}}

func AgentAppName(id string) string {
	// value: an agent app's name as the owner knows it; its id when no
	// entry names it.
	for _, a := range AgentApps {
		if a.ID == id {
			return a.Name
		}
	}
	return id
}

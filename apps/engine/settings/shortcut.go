package settings

type ContainersOutcome struct {
	Shortcut string
	Removals []ContainersRemoval
}

type ContainersRemoval struct {
	What  string
	Count int
}

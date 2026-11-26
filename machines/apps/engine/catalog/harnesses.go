package catalog

func Harnesses() []string {
	// value: the programs an agent runs on, by the name `--harness` takes,
	// the default first; each is installed by the catalog's app of that
	// program.
	return []string{"claude-code", "codex"}
}

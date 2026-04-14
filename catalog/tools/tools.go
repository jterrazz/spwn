package tools

import (
	"fmt"

	ib "spwn.sh/packages/imagebuilder"

	"spwn.sh/catalog/tools/architect"
	"spwn.sh/catalog/tools/build"
	"spwn.sh/catalog/tools/claude_code"
	"spwn.sh/catalog/tools/codex"
	"spwn.sh/catalog/tools/docker_cli"
	"spwn.sh/catalog/tools/git"
	"spwn.sh/catalog/tools/node"
	"spwn.sh/catalog/tools/python"
	"spwn.sh/catalog/tools/qmd"
	"spwn.sh/catalog/tools/spwn_cli"
	"spwn.sh/catalog/tools/unix"
)

// All is the list of every built-in tool.
// Adding a new tool? Import it and add it here.
var All = []ib.Tool{
	unix.Tool,
	git.Tool,
	node.Tool,
	python.Tool,
	build.Tool,
	docker_cli.Tool,
	claude_code.Tool,
	codex.Tool,
	spwn_cli.Tool,
	architect.Tool,
	qmd.Tool,
}

// RegisterDefaults registers all built-in tools into the given registry.
// Returns an error if any tool fails to register (typically a naming
// collision - indicates a programmer error in the catalog).
func RegisterDefaults(r *ib.Registry) error {
	for _, t := range All {
		if err := r.Register(t); err != nil {
			return fmt.Errorf("register built-in tool %q: %w", t.Name(), err)
		}
	}
	return nil
}

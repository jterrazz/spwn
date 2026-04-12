package catalog

import (
	"fmt"

	ib "spwn.sh/core/imagebuilder"

	"spwn.sh/core/imagebuilder/catalog/architect"
	"spwn.sh/core/imagebuilder/catalog/build"
	"spwn.sh/core/imagebuilder/catalog/claude_code"
	"spwn.sh/core/imagebuilder/catalog/codex"
	"spwn.sh/core/imagebuilder/catalog/docker_cli"
	"spwn.sh/core/imagebuilder/catalog/git"
	"spwn.sh/core/imagebuilder/catalog/node"
	"spwn.sh/core/imagebuilder/catalog/python"
	"spwn.sh/core/imagebuilder/catalog/qmd"
	"spwn.sh/core/imagebuilder/catalog/spwn_cli"
	"spwn.sh/core/imagebuilder/catalog/unix"
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
// collision — indicates a programmer error in the catalog).
func RegisterDefaults(r *ib.Registry) error {
	for _, t := range All {
		if err := r.Register(t); err != nil {
			return fmt.Errorf("register built-in tool %q: %w", t.Name(), err)
		}
	}
	return nil
}

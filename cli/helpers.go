package cli

import (
	"fmt"

	"github.com/jterrazz/universe/internal/architect"
	"github.com/jterrazz/universe/internal/backend"
	"github.com/jterrazz/universe/internal/state"
)

func newArchitect() (*architect.Architect, error) {
	docker, err := backend.NewDocker()
	if err != nil {
		return nil, fmt.Errorf("error: cannot connect to Docker.\n%w", err)
	}

	store, err := state.NewStore()
	if err != nil {
		return nil, fmt.Errorf("error: cannot initialize state store.\n%w", err)
	}

	return architect.New(docker, store), nil
}

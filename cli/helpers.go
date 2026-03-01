package cli

import (
	"fmt"

	"github.com/jterrazz/universe/internal/architect"
	"github.com/jterrazz/universe/internal/backend"
	"github.com/jterrazz/universe/internal/state"
)

// newArchitect creates an Architect with Docker backend and state store.
func newArchitect() (*architect.Architect, error) {
	docker, err := backend.NewDocker()
	if err != nil {
		return nil, fmt.Errorf("error: cannot connect to Docker.\nIs Docker running? Try: docker info")
	}

	store, err := state.NewStore()
	if err != nil {
		return nil, err
	}

	return architect.New(docker, store), nil
}

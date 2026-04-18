package resolver

import (
	"errors"
	"fmt"
)

var (
	// ErrToolNotFound is returned when a requested tool is not in the registry.
	ErrToolNotFound = errors.New("tool not found")

	// ErrDependencyCycle is returned when tool dependencies form a cycle.
	ErrDependencyCycle = errors.New("dependency cycle detected")

	// ErrDuplicateTool is returned when registering a tool with a name that already exists.
	ErrDuplicateTool = errors.New("duplicate tool name")
)

// MissingDependencyError is returned when a tool depends on an unregistered tool.
type MissingDependencyError struct {
	Tool       string
	Dependency string
}

func (e *MissingDependencyError) Error() string {
	return fmt.Sprintf("tool %s depends on %s, which is not registered", e.Tool, e.Dependency)
}

package imagebuilder

import (
	"errors"
	"fmt"
	"strings"
)

var (
	// ErrToolNotFound is returned when a requested tool is not in the registry.
	ErrToolNotFound = errors.New("tool not found")

	// ErrDependencyCycle is returned when tool dependencies form a cycle.
	ErrDependencyCycle = errors.New("dependency cycle detected")

	// ErrDuplicateTool is returned when registering a tool with a name that already exists.
	ErrDuplicateTool = errors.New("duplicate tool name")
)

// VerifyError is returned when post-build verification fails.
type VerifyError struct {
	Tool    string
	Command string
	Output  string
}

func (e *VerifyError) Error() string {
	return fmt.Sprintf("verify failed for %s: command %q returned: %s", e.Tool, e.Command, e.Output)
}

// MissingDependencyError is returned when a tool depends on an unregistered tool.
type MissingDependencyError struct {
	Tool       string
	Dependency string
}

func (e *MissingDependencyError) Error() string {
	return fmt.Sprintf("tool %s depends on %s, which is not registered", e.Tool, e.Dependency)
}

// BuildError wraps errors from the Docker build process.
type BuildError struct {
	Tag    string
	Cause  error
	Output string
}

func (e *BuildError) Error() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "build failed for %s: %s", e.Tag, e.Cause)
	if e.Output != "" {
		fmt.Fprintf(&sb, "\noutput: %s", e.Output)
	}
	return sb.String()
}

func (e *BuildError) Unwrap() error {
	return e.Cause
}

package compile

import (
	"fmt"
	"strings"
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

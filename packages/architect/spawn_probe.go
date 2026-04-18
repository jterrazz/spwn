package architect

import (
	"context"
	"fmt"
	"strings"

	"spwn.sh/packages/dependency"
)

// probeTools runs each resolved tool's Verify() commands inside
// the container. The catalog is the single source of truth — this
// used to fall back to a static list of binaries which drifted as
// new tools were added.
//
// Build one shell script; on any failure the block emits
// "FAIL <tool> :: <cmd>" and exits 1. On success the block emits
// "OK <tool>". The final exit 0 is only reached when every tool
// passed.
func (a *Architect) probeTools(ctx context.Context, containerID string, tools []dependency.Tool) ([]string, error) {
	if len(tools) == 0 {
		return nil, nil
	}

	var b strings.Builder
	b.WriteString("set -e\n")
	for _, t := range tools {
		for _, check := range t.Verify() {
			fmt.Fprintf(&b,
				"if ! %s >/dev/null 2>&1; then echo 'FAIL %s :: %s'; exit 1; fi\n",
				check, t.Name(), check,
			)
		}
		fmt.Fprintf(&b, "echo 'OK %s'\n", t.Name())
	}

	output, err := a.backend.ExecOutput(ctx, containerID, []string{"sh", "-c", b.String()})
	if err != nil {
		failLine := strings.TrimSpace(output)
		for _, line := range strings.Split(failLine, "\n") {
			if strings.HasPrefix(line, "FAIL ") {
				failLine = line
				break
			}
		}
		return nil, fmt.Errorf(
			"world tool verification failed (%s).\n"+
				"Hint: rebuild with --force-rebuild, or remove the tool from the agent's tools list",
			failLine,
		)
	}

	verified := make([]string, 0, len(tools))
	for _, line := range strings.Split(output, "\n") {
		if name := strings.TrimPrefix(line, "OK "); name != line {
			verified = append(verified, strings.TrimSpace(name))
		}
	}
	return verified, nil
}

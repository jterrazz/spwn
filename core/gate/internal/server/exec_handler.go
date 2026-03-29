package server

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"spwn.sh/core/gate/internal/bridge"
)

// ExecHandler returns an InvokeHandler that executes bridge sources on the host.
// It builds a lookup map from element name to GateBridge, and for each invocation
// resolves the source binary, checks capabilities, and runs it via os/exec.
func ExecHandler(bridges []bridge.GateBridge) InvokeHandler {
	lookup := make(map[string]bridge.GateBridge, len(bridges))
	for _, b := range bridges {
		lookup[b.As] = b
	}

	return func(element string, args []string) (InvokeResult, error) {
		b, ok := lookup[element]
		if !ok {
			return InvokeResult{
				ExitCode: 1,
				Stderr:   fmt.Sprintf("element bridge %q: not registered", element),
			}, nil
		}

		// Check capability: if args are provided, the first arg is treated as the operation.
		if len(args) > 0 {
			if err := bridge.CheckCapability(b, args[0]); err != nil {
				return InvokeResult{
					ExitCode: 1,
					Stderr:   err.Error(),
				}, nil
			}
		}

		source := b.Source
		if source == "" {
			return InvokeResult{
				ExitCode: 1,
				Stderr:   fmt.Sprintf("element bridge %q: empty source path", element),
			}, nil
		}

		// Split source into command and any embedded arguments.
		parts := strings.Fields(source)
		cmdName := parts[0]
		cmdArgs := append(parts[1:], args...)

		cmd := exec.Command(cmdName, cmdArgs...)

		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err := cmd.Run()

		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			} else {
				return InvokeResult{
					ExitCode: 1,
					Stderr:   fmt.Sprintf("exec %q: %v", cmdName, err),
				}, nil
			}
		}

		return InvokeResult{
			ExitCode: exitCode,
			Stdout:   stdout.String(),
			Stderr:   stderr.String(),
		}, nil
	}
}

package kit

import (
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
)

func usage(message, hint string) *fail.Error {
	return &fail.Error{Message: message, Hint: hint, Exit: fail.ExitUsage}
}

func failed(message, hint string, cause error) *fail.Error {
	return &fail.Error{Message: message, Hint: hint, Exit: fail.ExitFailure, Cause: cause}
}

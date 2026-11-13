package kit

import (
	"errors"

	"github.com/jterrazz/studio-station/apps/engine/core/fail"
)

// value: a failure an adapter already worded for the owner, with its repair.
func saidFailure(err error) (*fail.Error, bool) {
	var said *fail.Said
	if errors.As(err, &said) && said.Hint != "" {
		return failed(said.Message, said.Hint, err), true
	}
	return nil, false
}

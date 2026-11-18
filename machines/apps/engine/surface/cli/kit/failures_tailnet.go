package kit

import (
	"errors"

	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/ports"
)

func tailnetFailure(err error) (*fail.Error, bool) {
	var absent *ports.Absent
	var refused *ports.Refused
	switch {
	case errors.As(err, &absent):
		return failed(err.Error(), "station apps install "+absent.App, err), true
	case errors.As(err, &refused):
		return failed(refused.Error(), "docs/reference/ports.md#https", err), true
	}
	return nil, false
}
